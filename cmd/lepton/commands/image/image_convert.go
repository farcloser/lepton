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
	imageConvertCommand := &cobra.Command{
		Use:               "convert [flags] <source_ref> <target_ref>...",
		Short:             "convert an image",
		Long:              imageConvertHelp,
		Args:              cobra.MinimumNArgs(2),
		RunE:              convertAction,
		ValidArgsFunction: imageConvertShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	imageConvertCommand.Flags().String("format", "", "Format the output using the given Go template, e.g, 'json'")

	// #region zstd flags
	imageConvertCommand.Flags().Bool("zstd", false, "Convert legacy tar(.gz) layers to zstd. Should be used in conjunction with '--oci'")
	imageConvertCommand.Flags().Int("zstd-compression-level", 3, "zstd compression level")
	// #endregion

	// #region zstd:chunked flags
	imageConvertCommand.Flags().Bool("zstdchunked", false, "Convert legacy tar(.gz) layers to zstd:chunked for lazy pulling. Should be used in conjunction with '--oci'")
	imageConvertCommand.Flags().String("zstdchunked-record-in", "", "Read 'ctr-remote optimize --record-out=<FILE>' record file (EXPERIMENTAL)")
	imageConvertCommand.Flags().Int("zstdchunked-compression-level", 3, "zstd:chunked compression level") // SpeedDefault; see also https://pkg.go.dev/github.com/klauspost/compress/zstd#EncoderLevel
	imageConvertCommand.Flags().Int("zstdchunked-chunk-size", 0, "zstd:chunked chunk size")
	// #endregion

	// #region generic flags
	imageConvertCommand.Flags().Bool("uncompress", false, "Convert tar.gz layers to uncompressed tar layers")
	imageConvertCommand.Flags().Bool("oci", false, "Convert Docker media types to OCI media types")
	// #endregion

	// #region platform flags
	// platform is defined as StringSlice, not StringArray, to allow specifying "--platform=amd64,arm64"
	imageConvertCommand.Flags().StringSlice("platform", []string{}, "Convert content for a specific platform")
	imageConvertCommand.RegisterFlagCompletionFunc("platform", completion.Platforms)
	imageConvertCommand.Flags().Bool("all-platforms", false, "Convert content for all platforms")
	// #endregion

	return imageConvertCommand
}

func convertOptions(cmd *cobra.Command, _ []string) (options.ImageConvert, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImageConvert{}, err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return options.ImageConvert{}, err
	}

	// #region zstd flags
	zstd, err := cmd.Flags().GetBool("zstd")
	if err != nil {
		return options.ImageConvert{}, err
	}
	zstdCompressionLevel, err := cmd.Flags().GetInt("zstd-compression-level")
	if err != nil {
		return options.ImageConvert{}, err
	}
	// #endregion

	// #region zstd:chunked flags
	zstdchunked, err := cmd.Flags().GetBool("zstdchunked")
	if err != nil {
		return options.ImageConvert{}, err
	}
	zstdChunkedCompressionLevel, err := cmd.Flags().GetInt("zstdchunked-compression-level")
	if err != nil {
		return options.ImageConvert{}, err
	}
	zstdChunkedChunkSize, err := cmd.Flags().GetInt("zstdchunked-chunk-size")
	if err != nil {
		return options.ImageConvert{}, err
	}
	zstdChunkedRecordIn, err := cmd.Flags().GetString("zstdchunked-record-in")
	if err != nil {
		return options.ImageConvert{}, err
	}
	// #endregion

	// #region generic flags
	uncompress, err := cmd.Flags().GetBool("uncompress")
	if err != nil {
		return options.ImageConvert{}, err
	}
	oci, err := cmd.Flags().GetBool("oci")
	if err != nil {
		return options.ImageConvert{}, err
	}
	// #endregion

	// #region platform flags
	platforms, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return options.ImageConvert{}, err
	}
	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return options.ImageConvert{}, err
	}
	// #endregion
	return options.ImageConvert{
		GOptions: globalOptions,
		Format:   format,
		// #region zstd flags
		Zstd:                 zstd,
		ZstdCompressionLevel: zstdCompressionLevel,
		// #endregion
		// #region zstd:chunked flags
		ZstdChunked:                 zstdchunked,
		ZstdChunkedCompressionLevel: zstdChunkedCompressionLevel,
		ZstdChunkedChunkSize:        zstdChunkedChunkSize,
		ZstdChunkedRecordIn:         zstdChunkedRecordIn,
		// #endregion
		// #region generic flags
		Uncompress: uncompress,
		Oci:        oci,
		// #endregion
		// #region platform flags
		Platforms:    platforms,
		AllPlatforms: allPlatforms,
		// #endregion
		Stdout: cmd.OutOrStdout(),
	}, nil
}

func convertAction(cmd *cobra.Command, args []string) error {
	options, err := convertOptions(cmd, args)
	if err != nil {
		return err
	}
	srcRawRef := args[0]
	destRawRef := args[1]

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Convert(ctx, cli, srcRawRef, destRawRef, options)
}

func imageConvertShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show image names
	return completion.ImageNames(cmd, args, toComplete)
}
