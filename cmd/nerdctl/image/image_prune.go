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

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/cmd/image"
)

func PruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove unused images",
		Args:          cobra.NoArgs,
		RunE:          imagePruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP(flagAll, "a", false, "Remove all unused images, not just dangling ones")
	cmd.Flags().StringSlice(flagFilter, []string{}, "Filter output based on conditions provided")
	cmd.Flags().BoolP(flagForce, "f", false, "Do not prompt for confirmation")

	return cmd
}

func PruneOptions(cmd *cobra.Command, args []string) (types.ImagePruneOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return types.ImagePruneOptions{}, err
	}
	all, err := cmd.Flags().GetBool(flagAll)
	if err != nil {
		return types.ImagePruneOptions{}, err
	}

	var filters []string
	if cmd.Flags().Changed("filter") {
		filters, err = cmd.Flags().GetStringSlice("filter")
		if err != nil {
			return types.ImagePruneOptions{}, err
		}
	}

	force, err := cmd.Flags().GetBool(flagForce)
	if err != nil {
		return types.ImagePruneOptions{}, err
	}

	return types.ImagePruneOptions{
		Stdout:   cmd.OutOrStdout(),
		GOptions: *globalOptions,
		All:      all,
		Filters:  filters,
		Force:    force,
	}, err
}

func imagePruneAction(cmd *cobra.Command, args []string) error {
	options, err := PruneOptions(cmd, args)
	if err != nil {
		return err
	}

	if !options.Force {
		var msg string
		if !options.All {
			msg = "This will remove all dangling images."
		} else {
			msg = "This will remove all images without at least one container associated to them."
		}

		if confirmed, err := helpers.Confirm(cmd, fmt.Sprintf("WARNING! %s.", msg)); err != nil || !confirmed {
			return err
		}
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Prune(ctx, client, options)
}
