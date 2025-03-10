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
	"fmt"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/image"
)

func pruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove unused images",
		Args:          cobra.NoArgs,
		RunE:          pruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("all", "a", false, "Remove all unused images, not just dangling ones")
	cmd.Flags().StringSlice("filter", []string{}, "Filter output based on conditions provided")
	cmd.Flags().BoolP("force", "f", false, "Do not prompt for confirmation")

	return cmd
}

func pruneOptions(cmd *cobra.Command, _ []string) (options.ImagePrune, error) {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return options.ImagePrune{}, err
	}

	var filters []string
	if cmd.Flags().Changed("filter") {
		filters, err = cmd.Flags().GetStringSlice("filter")
		if err != nil {
			return options.ImagePrune{}, err
		}
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return options.ImagePrune{}, err
	}

	return options.ImagePrune{
		Stdout:  cmd.OutOrStdout(),
		All:     all,
		Filters: filters,
		Force:   force,
	}, err
}

func pruneAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := pruneOptions(cmd, args)
	if err != nil {
		return err
	}

	if !opts.Force {
		var msg string
		if !opts.All {
			msg = "This will remove all dangling images."
		} else {
			msg = "This will remove all images without at least one container associated to them."
		}

		if err := helpers.Confirm(cmd, fmt.Sprintf("WARNING! %s.", msg)); err != nil {
			return err
		}
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Prune(ctx, cli, globalOptions, opts)
}
