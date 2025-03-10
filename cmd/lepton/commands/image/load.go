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
	"go.farcloser.world/lepton/pkg/imgutil/load"
)

func LoadCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "load",
		Args:          cobra.NoArgs,
		Short:         "Load an image from a tar archive or STDIN",
		Long:          "Supports both Docker Image Spec v1.2 and OCI Image Spec v1.0.",
		RunE:          loadAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringP("input", "i", "", "Read from tar archive file, instead of STDIN")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress the load output")
	cmd.Flags().StringSlice("platform", []string{}, "Import content for a specific platform")
	cmd.Flags().Bool("all-platforms", false, "Import content for all platforms")

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

func loadOptions(cmd *cobra.Command) (options.ImageLoad, error) {
	input, err := cmd.Flags().GetString("input")
	if err != nil {
		return options.ImageLoad{}, err
	}
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImageLoad{}, err
	}
	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return options.ImageLoad{}, err
	}
	platform, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return options.ImageLoad{}, err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return options.ImageLoad{}, err
	}
	return options.ImageLoad{
		GOptions:     globalOptions,
		Input:        input,
		Platform:     platform,
		AllPlatforms: allPlatforms,
		Stdout:       cmd.OutOrStdout(),
		Stdin:        cmd.InOrStdin(),
		Quiet:        quiet,
	}, nil
}

func loadAction(cmd *cobra.Command, _ []string) error {
	opts, err := loadOptions(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	_, err = load.FromArchive(ctx, cli, opts)
	return err
}
