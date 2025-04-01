/*
   Copyright Farcloser.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images/converter"
	"github.com/containerd/containerd/v2/core/images/converter/uncompress"
	"github.com/containerd/log"
	"github.com/containerd/stargz-snapshotter/estargz"
	zstdchunkedconvert "github.com/containerd/stargz-snapshotter/nativeconverter/zstdchunked"
	"github.com/containerd/stargz-snapshotter/recorder"

	"go.farcloser.world/containers/reference"
	"go.farcloser.world/core/compression/zstd"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/formatter"
	converterutil "go.farcloser.world/lepton/pkg/imgutil/converter"
	"go.farcloser.world/lepton/pkg/platformutil"
)

func Convert(
	ctx context.Context,
	client *containerd.Client,
	output io.Writer,
	globalOptions *options.Global,
	opts *options.ImageConvert,
) error {
	convertOpts := []converter.Opt{}
	if opts.SourceRef == "" || opts.DestinationRef == "" {
		return errors.New("src and target image need to be specified")
	}

	parsedReference, err := reference.Parse(opts.SourceRef)
	if err != nil {
		return err
	}
	srcRef := parsedReference.String()

	parsedReference, err = reference.Parse(opts.DestinationRef)
	if err != nil {
		return err
	}
	targetRef := parsedReference.String()

	platMC, err := platformutil.NewMatchComparer(opts.AllPlatforms, opts.Platforms)
	if err != nil {
		return err
	}
	convertOpts = append(convertOpts, converter.WithPlatform(platMC))

	// Ensure all the layers are here: https://github.com/containerd/nerdctl/issues/3425
	err = EnsureAllContent(ctx, client, srcRef, platMC, globalOptions)
	if err != nil {
		return err
	}

	zstdOpts := opts.Zstd
	zstdchunked := opts.ZstdChunked

	if zstdOpts || zstdchunked {
		convertCount := 0
		if zstdOpts {
			convertCount++
		}
		if zstdchunked {
			convertCount++
		}
		if convertCount > 1 {
			return errors.New("options --zstd and --zstdchunked lead to conflict, only one of them can be used")
		}

		var convertFunc converter.ConvertFunc
		var convertType string
		switch {
		case zstdOpts:
			convertFunc, err = getZstdConverter(opts)
			if err != nil {
				return err
			}
			convertType = "zstd"
		case zstdchunked:
			convertFunc, err = getZstdchunkedConverter(globalOptions, opts)
			if err != nil {
				return err
			}
			convertType = "zstdchunked"
		}

		convertOpts = append(convertOpts, converter.WithLayerConvertFunc(convertFunc))
		if !opts.Oci {
			log.G(ctx).Warnf("option --%s should be used in conjunction with --oci", convertType)
		}
		if opts.Uncompress {
			return fmt.Errorf("option --%s conflicts with --uncompress", convertType)
		}
	}

	if opts.Uncompress {
		convertOpts = append(convertOpts, converter.WithLayerConvertFunc(uncompress.LayerConvertFunc))
	}

	if opts.Oci {
		convertOpts = append(convertOpts, converter.WithDockerToOCI(true))
	}

	// converter.Convert() gains the lease by itself
	newImg, err := converterutil.Convert(ctx, client, targetRef, srcRef, convertOpts...)
	if err != nil {
		return err
	}
	res := converterutil.ConvertedImageInfo{
		Image: newImg.Name + "@" + newImg.Target.Digest.String(),
	}

	return printConvertedImage(output, opts, res)
}

func getZstdConverter(options *options.ImageConvert) (converter.ConvertFunc, error) {
	return converterutil.ZstdLayerConvertFunc(*options)
}

func getZstdchunkedConverter(globalOptions *options.Global, opts *options.ImageConvert) (converter.ConvertFunc, error) {
	esgzOpts := []estargz.Option{
		estargz.WithChunkSize(opts.ZstdChunkedChunkSize),
	}

	if opts.ZstdChunkedRecordIn != "" {
		if !globalOptions.Experimental {
			return nil, errors.New("zstdchunked-record-in requires experimental mode to be enabled")
		}

		log.L.Warn("--zstdchunked-record-in flag is experimental and subject to change")
		paths, err := readPathsFromRecordFile(opts.ZstdChunkedRecordIn)
		if err != nil {
			return nil, err
		}
		esgzOpts = append(esgzOpts, estargz.WithPrioritizedFiles(paths))
		var ignored []string
		esgzOpts = append(esgzOpts, estargz.WithAllowPrioritizeNotFound(&ignored))
	}
	return zstdchunkedconvert.LayerConvertFuncWithCompressionLevel(
		zstd.EncoderLevelFromZstd(opts.ZstdChunkedCompressionLevel),
		esgzOpts...), nil
}

func readPathsFromRecordFile(filename string) ([]string, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	dec := json.NewDecoder(r)
	var paths []string
	added := make(map[string]struct{})
	for dec.More() {
		var e recorder.Entry
		if err := dec.Decode(&e); err != nil {
			return nil, err
		}
		if _, ok := added[e.Path]; !ok {
			paths = append(paths, e.Path)
			added[e.Path] = struct{}{}
		}
	}
	return paths, nil
}

func printConvertedImage(stdout io.Writer, options *options.ImageConvert, img converterutil.ConvertedImageInfo) error {
	switch options.Format {
	case formatter.FormatJSON:
		b, err := json.MarshalIndent(img, "", "    ")
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(b))
	default:
		for i, e := range img.ExtraImages {
			elems := strings.SplitN(e, "@", 2)
			if len(elems) < 2 {
				log.L.Errorf("extra reference %q doesn't contain digest", e)
			} else {
				log.L.Infof("Extra image(%d) %s", i, elems[0])
			}
		}
		elems := strings.SplitN(img.Image, "@", 2)
		if len(elems) < 2 {
			log.L.Errorf("reference %q doesn't contain digest", img.Image)
		} else {
			fmt.Fprintln(stdout, elems[1])
		}
	}
	return nil
}
