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

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/volume"
)

func removeCommand() *cobra.Command {
	volumeRmCommand := &cobra.Command{
		Use:               "rm [flags] VOLUME [VOLUME...]",
		Aliases:           []string{"remove"},
		Short:             "Remove one or more volumes",
		Long:              "NOTE: You cannot remove a volume that is in use by a container.",
		Args:              cobra.MinimumNArgs(1),
		RunE:              removeAction,
		ValidArgsFunction: volumeRmShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	volumeRmCommand.Flags().BoolP("force", "f", false, "(unimplemented yet)")
	return volumeRmCommand
}

func removeOptions(cmd *cobra.Command, _ []string) (*options.VolumeRemove, error) {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return nil, err
	}

	return &options.VolumeRemove{
		Force:  force,
		Stdout: cmd.OutOrStdout(),
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

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return volume.Remove(ctx, cli, args, globalOptions, opts)
}

func volumeRmShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show volume names
	return completion.VolumeNames(cmd)
}
