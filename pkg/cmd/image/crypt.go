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
	"errors"
	"fmt"
	"io"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/converter"
	"github.com/containerd/imgcrypt/v2/images/encryption"
	"github.com/containerd/imgcrypt/v2/images/encryption/parsehelpers"
	"github.com/containerd/platforms"

	"go.farcloser.world/containers/reference"
	"go.farcloser.world/containers/specs"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/platformutil"
)

func layerDescs(ctx context.Context, provider content.Provider, imageTarget specs.Descriptor, platform platforms.MatchComparer) ([]specs.Descriptor, error) {
	var descs []specs.Descriptor
	err := images.Walk(ctx, images.Handlers(images.HandlerFunc(func(ctx context.Context, desc specs.Descriptor) ([]specs.Descriptor, error) {
		if images.IsLayerType(desc.MediaType) {
			descs = append(descs, desc)
		}
		return nil, nil
	}), images.FilterPlatforms(images.ChildrenHandler(provider), platform)), imageTarget)
	return descs, err
}

func Crypt(ctx context.Context, encrypt bool, cli *client.Client, output io.Writer, globalOptions *options.Global, opts options.ImageCrypt) error {
	var convertOpts = []converter.Opt{}
	if opts.SourceRef == "" || opts.DestinationRef == "" {
		return errors.New("src and target image need to be specified")
	}

	parsedRerefence, err := reference.Parse(opts.SourceRef)
	if err != nil {
		return err
	}
	srcRef := parsedRerefence.String()

	parsedRerefence, err = reference.Parse(opts.DestinationRef)
	if err != nil {
		return err
	}
	targetRef := parsedRerefence.String()

	platMC, err := platformutil.NewMatchComparer(opts.AllPlatforms, opts.Platforms)
	if err != nil {
		return err
	}
	convertOpts = append(convertOpts, converter.WithPlatform(platMC))

	imgcryptFlags, err := parseImgcryptFlags(opts, encrypt)
	if err != nil {
		return err
	}

	srcImg, err := cli.ImageService().Get(ctx, srcRef)
	if err != nil {
		return err
	}

	descs, err := layerDescs(ctx, cli.ContentStore(), srcImg.Target, platMC)
	if err != nil {
		return err
	}

	layerFilter := func(desc specs.Descriptor) bool {
		return true
	}
	var convertFunc converter.ConvertFunc

	if encrypt {
		cc, err := parsehelpers.CreateCryptoConfig(imgcryptFlags, descs)
		if err != nil {
			return err
		}
		convertFunc = encryption.GetImageEncryptConverter(&cc, layerFilter)
	} else {
		cc, err := parsehelpers.CreateDecryptCryptoConfig(imgcryptFlags, descs)
		if err != nil {
			return err
		}
		convertFunc = encryption.GetImageDecryptConverter(&cc, layerFilter)
	}
	// we have to compose the DefaultIndexConvertFunc here to match platforms.
	convertFunc = composeConvertFunc(converter.DefaultIndexConvertFunc(nil, false, platMC), convertFunc)
	convertOpts = append(convertOpts, converter.WithIndexConvertFunc(convertFunc))

	// converter.Convert() gains the lease by itself
	newImg, err := converter.Convert(ctx, cli, targetRef, srcRef, convertOpts...)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(output, newImg.Target.Digest.String())
	if err != nil {
		return err
	}

	return nil
}

// parseImgcryptFlags corresponds to https://github.com/containerd/imgcrypt/blob/v1.1.2/cmd/ctr/commands/images/crypt_utils.go#L244-L252
func parseImgcryptFlags(options options.ImageCrypt, encrypt bool) (parsehelpers.EncArgs, error) {
	var a parsehelpers.EncArgs

	a.GPGHomedir = options.GpgHomeDir
	a.GPGVersion = options.GpgVersion
	a.Key = options.Keys
	if encrypt {
		a.Recipient = options.Recipients
		if len(a.Recipient) == 0 {
			return a, errors.New("at least one recipient must be specified (e.g., --recipient=jwe:mypubkey.pem)")
		}
	}
	// While --recipient can be specified only for `image encrypt`,
	// --dec-recipient can be specified for both `image encrypt` and `image decrypt`.
	a.DecRecipient = options.DecRecipients
	return a, nil
}

func composeConvertFunc(a, b converter.ConvertFunc) converter.ConvertFunc {
	return func(ctx context.Context, cs content.Store, desc specs.Descriptor) (*specs.Descriptor, error) {
		newDesc, err := a(ctx, cs, desc)
		if err != nil {
			return newDesc, err
		}
		if newDesc == nil {
			return b(ctx, cs, desc)
		}
		return b(ctx, cs, *newDesc)
	}
}
