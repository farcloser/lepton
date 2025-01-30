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

package volume

import (
	"github.com/spf13/cobra"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/volume"
)

func removeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove [flags] VOLUME [VOLUME...]",
		Aliases:           []string{"rm"},
		Short:             "Remove one or more volumes",
		Long:              "NOTE: You cannot remove a volume that is in use by a container.",
		Args:              cobra.MinimumNArgs(1),
		RunE:              removeAction,
		ValidArgsFunction: completion.VolumeNames,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().BoolP(flagForce, "f", false, "(unimplemented yet)")

	return cmd
}

func removeOptions(cmd *cobra.Command, args []string) (*options.VolumeRemove, error) {
	force, err := cmd.Flags().GetBool(flagForce)
	if err != nil {
		return nil, err
	}

	return &options.VolumeRemove{
		NamesList: args,
		Force:     force,
	}, nil
}

func removeAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := removeOptions(cmd, args)
	if err != nil {
		return err
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return volume.Remove(ctx, client, cmd.OutOrStdout(), globalOptions, opts)
}
