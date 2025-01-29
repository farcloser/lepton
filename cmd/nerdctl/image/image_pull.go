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
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/cmd/image"
	"github.com/containerd/nerdctl/v2/pkg/platformutil"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

func ProcessImageVerifyOptions(cmd *cobra.Command, args []string) (opt types.ImageVerifyOptions, err error) {
	if opt.Provider, err = cmd.Flags().GetString("verify"); err != nil {
		return
	}
	if opt.CosignKey, err = cmd.Flags().GetString("cosign-key"); err != nil {
		return
	}
	if opt.CosignCertificateIdentity, err = cmd.Flags().GetString("cosign-certificate-identity"); err != nil {
		return
	}
	if opt.CosignCertificateIdentityRegexp, err = cmd.Flags().GetString("cosign-certificate-identity-regexp"); err != nil {
		return
	}
	if opt.CosignCertificateOidcIssuer, err = cmd.Flags().GetString("cosign-certificate-oidc-issuer"); err != nil {
		return
	}
	if opt.CosignCertificateOidcIssuerRegexp, err = cmd.Flags().GetString("cosign-certificate-oidc-issuer-regexp"); err != nil {
		return
	}
	return
}

func NewPullCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "pull [flags] NAME[:TAG]",
		Short:         "Pull an image from a registry.",
		Args:          helpers.IsExactArgs(1),
		RunE:          pullAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().String(flagUnpack, "auto", "Unpack the image for the current single platform (auto/true/false)")
	cmd.Flags().StringSlice(flagPlatform, nil, "Pull content for a specific platform")
	cmd.Flags().Bool(flagAllPlatforms, false, "Pull content for all platforms")
	cmd.Flags().String(flagVerify, "none", "Verify the image (none|cosign|notation)")
	cmd.Flags().String(flagCosignKey, "", "Path to the public key file, KMS, URI or Kubernetes Secret for --verify=cosign")
	cmd.Flags().String(flagCosignCertificateIdentity, "", "The identity expected in a valid Fulcio certificate for --verify=cosign. Valid values include email address, DNS names, IP addresses, and URIs. Either --cosign-certificate-identity or --cosign-certificate-identity-regexp must be set for keyless flows")
	cmd.Flags().String(flagCosignCertificateIdentityRegexp, "", "A regular expression alternative to --cosign-certificate-identity for --verify=cosign. Accepts the Go regular expression syntax described at https://golang.org/s/re2syntax. Either --cosign-certificate-identity or --cosign-certificate-identity-regexp must be set for keyless flows")
	cmd.Flags().String(flagCosignCertificateOidcIssuer, "", "The OIDC issuer expected in a valid Fulcio certificate for --verify=cosign,, e.g. https://token.actions.githubusercontent.com or https://oauth2.sigstore.dev/auth. Either --cosign-certificate-oidc-issuer or --cosign-certificate-oidc-issuer-regexp must be set for keyless flows")
	cmd.Flags().String(flagCosignCertificateOidcIssuerRegexp, "", "A regular expression alternative to --certificate-oidc-issuer for --verify=cosign,. Accepts the Go regular expression syntax described at https://golang.org/s/re2syntax. Either --cosign-certificate-oidc-issuer or --cosign-certificate-oidc-issuer-regexp must be set for keyless flows")
	cmd.Flags().String(flagSociIndexDigest, "", "Specify a particular index digest for SOCI. If left empty, SOCI will automatically use the index determined by the selection policy.")
	cmd.Flags().BoolP(flagQuiet, "q", false, "Suppress verbose output")

	_ = cmd.RegisterFlagCompletionFunc(flagUnpack, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc(flagPlatform, completion.Platforms)
	_ = cmd.RegisterFlagCompletionFunc(flagVerify, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"none", "cosign", "notation"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func processPullCommandFlags(cmd *cobra.Command, args []string) (types.ImagePullOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return types.ImagePullOptions{}, err
	}
	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return types.ImagePullOptions{}, err
	}
	platform, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return types.ImagePullOptions{}, err
	}

	ociSpecPlatform, err := platformutil.NewOCISpecPlatformSlice(allPlatforms, platform)
	if err != nil {
		return types.ImagePullOptions{}, err
	}

	unpackStr, err := cmd.Flags().GetString("unpack")
	if err != nil {
		return types.ImagePullOptions{}, err
	}
	unpack, err := strutil.ParseBoolOrAuto(unpackStr)
	if err != nil {
		return types.ImagePullOptions{}, err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return types.ImagePullOptions{}, err
	}

	sociIndexDigest, err := cmd.Flags().GetString("soci-index-digest")
	if err != nil {
		return types.ImagePullOptions{}, err
	}

	verifyOptions, err := ProcessImageVerifyOptions(cmd, args)
	if err != nil {
		return types.ImagePullOptions{}, err
	}
	return types.ImagePullOptions{
		GOptions:        *globalOptions,
		VerifyOptions:   verifyOptions,
		OCISpecPlatform: ociSpecPlatform,
		Unpack:          unpack,
		Mode:            "always",
		Quiet:           quiet,
		RFlags: types.RemoteSnapshotterFlags{
			SociIndexDigest: sociIndexDigest,
		},
		Stdout:                 cmd.OutOrStdout(),
		Stderr:                 cmd.OutOrStderr(),
		ProgressOutputToStdout: true,
	}, nil
}

func pullAction(cmd *cobra.Command, args []string) error {
	options, err := processPullCommandFlags(cmd, args)
	if err != nil {
		return err
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Pull(ctx, client, args[0], options)
}
