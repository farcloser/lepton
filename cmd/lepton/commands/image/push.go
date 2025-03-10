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

const (
	allowNonDistFlag = "allow-nondistributable-artifacts"
)

func PushCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "push [flags] NAME[:TAG]",
		Short:             "Push an image or a repository to a registry.",
		Args:              helpers.IsExactArgs(1),
		RunE:              pushAction,
		ValidArgsFunction: pushShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().StringSlice("platform", []string{}, "Push content for a specific platform")
	cmd.Flags().Bool("all-platforms", false, "Push content for all platforms")
	cmd.Flags().String("sign", "none", "Sign the image (none|cosign|notation")
	cmd.Flags().String("cosign-key", "", "Path to the private key file, KMS URI or Kubernetes Secret for --sign=cosign")
	cmd.Flags().String("notation-key-name", "", "Signing key name for a key previously added to notation's key list for --sign=notation")
	cmd.Flags().Int64("soci-span-size", -1, "Span size that soci index uses to segment layer data. Default is 4 MiB.")
	cmd.Flags().Int64("soci-min-layer-size", -1, "Minimum layer size to build zTOC for. Smaller layers won't have zTOC and not lazy pulled. Default is 10 MiB.")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress verbose output")
	cmd.Flags().Bool(allowNonDistFlag, false, "Allow pushing images with non-distributable blobs")

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)
	_ = cmd.RegisterFlagCompletionFunc("sign", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"none", "cosign", "notation"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func pushOptions(cmd *cobra.Command, args []string) (options.ImagePush, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImagePush{}, err
	}
	platform, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return options.ImagePush{}, err
	}
	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return options.ImagePush{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return options.ImagePush{}, err
	}
	allowNonDist, err := cmd.Flags().GetBool(allowNonDistFlag)
	if err != nil {
		return options.ImagePush{}, err
	}
	signOptions, err := signOptions(cmd, args)
	if err != nil {
		return options.ImagePush{}, err
	}
	sociOptions, err := sociOptions(cmd, args)
	if err != nil {
		return options.ImagePush{}, err
	}
	return options.ImagePush{
		GOptions:                       globalOptions,
		SignOptions:                    signOptions,
		SociOptions:                    sociOptions,
		Platforms:                      platform,
		AllPlatforms:                   allPlatforms,
		Quiet:                          quiet,
		AllowNondistributableArtifacts: allowNonDist,
		Stdout:                         cmd.OutOrStdout(),
	}, nil
}

func pushAction(cmd *cobra.Command, args []string) error {
	opts, err := pushOptions(cmd, args)
	if err != nil {
		return err
	}
	rawRef := args[0]

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Push(ctx, cli, rawRef, opts)
}

func pushShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show image names
	return completion.ImageNames(cmd, args, toComplete)
}

func signOptions(cmd *cobra.Command, _ []string) (opt options.ImageSign, err error) {
	if opt.Provider, err = cmd.Flags().GetString("sign"); err != nil {
		return
	}
	if opt.CosignKey, err = cmd.Flags().GetString("cosign-key"); err != nil {
		return
	}
	if opt.NotationKeyName, err = cmd.Flags().GetString("notation-key-name"); err != nil {
		return
	}
	return
}

func sociOptions(cmd *cobra.Command, _ []string) (opt options.Soci, err error) {
	if opt.SpanSize, err = cmd.Flags().GetInt64("soci-span-size"); err != nil {
		return
	}
	if opt.MinLayerSize, err = cmd.Flags().GetInt64("soci-min-layer-size"); err != nil {
		return
	}
	return
}
