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

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/image"
	"github.com/containerd/nerdctl/v2/pkg/platformutil"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

func PullCommand() *cobra.Command {
	var pullCommand = &cobra.Command{
		Use:           "pull [flags] NAME[:TAG]",
		Short:         "Pull an image from a registry.",
		Args:          helpers.IsExactArgs(1),
		RunE:          pullAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pullCommand.Flags().String("unpack", "auto", "Unpack the image for the current single platform (auto/true/false)")
	pullCommand.RegisterFlagCompletionFunc("unpack", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	// #region platform flags
	// platform is defined as StringSlice, not StringArray, to allow specifying "--platform=amd64,arm64"
	pullCommand.Flags().StringSlice("platform", nil, "Pull content for a specific platform")
	pullCommand.RegisterFlagCompletionFunc("platform", completion.Platforms)
	pullCommand.Flags().Bool("all-platforms", false, "Pull content for all platforms")
	// #endregion

	// #region verify flags
	pullCommand.Flags().String("verify", "none", "Verify the image (none|cosign|notation)")
	pullCommand.RegisterFlagCompletionFunc("verify", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"none", "cosign", "notation"}, cobra.ShellCompDirectiveNoFileComp
	})
	pullCommand.Flags().String("cosign-key", "", "Path to the public key file, KMS, URI or Kubernetes Secret for --verify=cosign")
	pullCommand.Flags().String("cosign-certificate-identity", "", "The identity expected in a valid Fulcio certificate for --verify=cosign. Valid values include email address, DNS names, IP addresses, and URIs. Either --cosign-certificate-identity or --cosign-certificate-identity-regexp must be set for keyless flows")
	pullCommand.Flags().String("cosign-certificate-identity-regexp", "", "A regular expression alternative to --cosign-certificate-identity for --verify=cosign. Accepts the Go regular expression syntax described at https://golang.org/s/re2syntax. Either --cosign-certificate-identity or --cosign-certificate-identity-regexp must be set for keyless flows")
	pullCommand.Flags().String("cosign-certificate-oidc-issuer", "", "The OIDC issuer expected in a valid Fulcio certificate for --verify=cosign,, e.g. https://token.actions.githubusercontent.com or https://oauth2.sigstore.dev/auth. Either --cosign-certificate-oidc-issuer or --cosign-certificate-oidc-issuer-regexp must be set for keyless flows")
	pullCommand.Flags().String("cosign-certificate-oidc-issuer-regexp", "", "A regular expression alternative to --certificate-oidc-issuer for --verify=cosign,. Accepts the Go regular expression syntax described at https://golang.org/s/re2syntax. Either --cosign-certificate-oidc-issuer or --cosign-certificate-oidc-issuer-regexp must be set for keyless flows")
	// #endregion

	// #region socipull flags
	pullCommand.Flags().String("soci-index-digest", "", "Specify a particular index digest for SOCI. If left empty, SOCI will automatically use the index determined by the selection policy.")
	// #endregion

	pullCommand.Flags().BoolP("quiet", "q", false, "Suppress verbose output")

	return pullCommand
}

func pullOptions(cmd *cobra.Command, args []string) (options.ImagePull, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImagePull{}, err
	}
	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return options.ImagePull{}, err
	}
	platform, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return options.ImagePull{}, err
	}

	ociSpecPlatform, err := platformutil.NewOCISpecPlatformSlice(allPlatforms, platform)
	if err != nil {
		return options.ImagePull{}, err
	}

	unpackStr, err := cmd.Flags().GetString("unpack")
	if err != nil {
		return options.ImagePull{}, err
	}
	unpack, err := strutil.ParseBoolOrAuto(unpackStr)
	if err != nil {
		return options.ImagePull{}, err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return options.ImagePull{}, err
	}

	sociIndexDigest, err := cmd.Flags().GetString("soci-index-digest")
	if err != nil {
		return options.ImagePull{}, err
	}

	verifyOptions, err := helpers.ProcessImageVerifyOptions(cmd, args)
	if err != nil {
		return options.ImagePull{}, err
	}
	return options.ImagePull{
		GOptions:        globalOptions,
		VerifyOptions:   verifyOptions,
		OCISpecPlatform: ociSpecPlatform,
		Unpack:          unpack,
		Mode:            "always",
		Quiet:           quiet,
		RFlags: options.RemoteSnapshotterFlags{
			SociIndexDigest: sociIndexDigest,
		},
		Stdout:                 cmd.OutOrStdout(),
		Stderr:                 cmd.OutOrStderr(),
		ProgressOutputToStdout: true,
	}, nil
}

func pullAction(cmd *cobra.Command, args []string) error {
	options, err := pullOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Pull(ctx, cli, args[0], options)
}
