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
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/image"
)

const imageConvertHelp = `Convert an image format.

e.g., 'nerdctl image convert --oci example.com/foo:orig example.com/foo:esgz'

Use '--platform' to define the output platform.
When '--all-platforms' is given all images in a manifest list must be available.

For encryption and decryption, use 'nerdctl image (encrypt|decrypt)' command.
`

// imageConvertCommand is from https://github.com/containerd/stargz-snapshotter/blob/d58f43a8235e46da73fb94a1a35280cb4d607b2c/cmd/ctr-remote/commands/convert.go
func convertCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "convert [flags] <source_ref> <target_ref>...",
		Short:             "convert an image",
		Long:              imageConvertHelp,
		Args:              cobra.MinimumNArgs(2),
		RunE:              convertAction,
		ValidArgsFunction: completion.ImageNames,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().String("format", "", "Format the output using the given Go template, e.g, 'json'")
	cmd.Flags().Bool("zstd", false, "Convert legacy tar(.gz) layers to zstd. Should be used in conjunction with '--oci'")
	cmd.Flags().Int("zstd-compression-level", 3, "zstd compression level")
	cmd.Flags().Bool("zstdchunked", false, "Convert legacy tar(.gz) layers to zstd:chunked for lazy pulling. Should be used in conjunction with '--oci'")
	cmd.Flags().String("zstdchunked-record-in", "", "Read 'ctr-remote optimize --record-out=<FILE>' record file (EXPERIMENTAL)")
	cmd.Flags().Int("zstdchunked-compression-level", 3, "zstd:chunked compression level") // SpeedDefault; see also https://pkg.go.dev/github.com/klauspost/compress/zstd#EncoderLevel
	cmd.Flags().Int("zstdchunked-chunk-size", 0, "zstd:chunked chunk size")
	cmd.Flags().Bool("uncompress", false, "Convert tar.gz layers to uncompressed tar layers")
	cmd.Flags().Bool("oci", false, "Convert Docker media types to OCI media types")
	cmd.Flags().StringSlice("platform", []string{}, "Convert content for a specific platform")
	cmd.Flags().Bool("all-platforms", false, "Convert content for all platforms")

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

func convertOptions(cmd *cobra.Command, args []string) (*options.ImageConvert, error) {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}

	zstd, err := cmd.Flags().GetBool("zstd")
	if err != nil {
		return nil, err
	}

	zstdCompressionLevel, err := cmd.Flags().GetInt("zstd-compression-level")
	if err != nil {
		return nil, err
	}

	zstdchunked, err := cmd.Flags().GetBool("zstdchunked")
	if err != nil {
		return nil, err
	}

	zstdChunkedCompressionLevel, err := cmd.Flags().GetInt("zstdchunked-compression-level")
	if err != nil {
		return nil, err
	}

	zstdChunkedChunkSize, err := cmd.Flags().GetInt("zstdchunked-chunk-size")
	if err != nil {
		return nil, err
	}

	zstdChunkedRecordIn, err := cmd.Flags().GetString("zstdchunked-record-in")
	if err != nil {
		return nil, err
	}

	uncompress, err := cmd.Flags().GetBool("uncompress")
	if err != nil {
		return nil, err
	}

	oci, err := cmd.Flags().GetBool("oci")
	if err != nil {
		return nil, err
	}

	platforms, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return nil, err
	}

	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return nil, err
	}

	return &options.ImageConvert{
		SourceRef:                   args[0],
		DestinationRef:              args[1],
		Format:                      format,
		Zstd:                        zstd,
		ZstdCompressionLevel:        zstdCompressionLevel,
		ZstdChunked:                 zstdchunked,
		ZstdChunkedCompressionLevel: zstdChunkedCompressionLevel,
		ZstdChunkedChunkSize:        zstdChunkedChunkSize,
		ZstdChunkedRecordIn:         zstdChunkedRecordIn,
		Uncompress:                  uncompress,
		Oci:                         oci,
		Platforms:                   platforms,
		AllPlatforms:                allPlatforms,
	}, nil
}

func convertAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := convertOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return image.Convert(ctx, cli, cmd.OutOrStdout(), globalOptions, opts)
}
