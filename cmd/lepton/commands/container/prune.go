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

package container

import (
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
)

func pruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove all stopped containers",
		Args:          cobra.NoArgs,
		RunE:          pruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("force", "f", false, "Do not prompt for confirmation")

	return cmd
}

func pruneOptions(cmd *cobra.Command, _ []string) (*options.ContainerPrune, error) {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return nil, err
	}

	if !force {
		if err := helpers.Confirm(cmd, "WARNING! This will remove all stopped containers."); err != nil {
			return nil, err
		}
	}

	return &options.ContainerPrune{}, nil
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

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return container.Prune(ctx, cli, cmd.OutOrStdout(), globalOptions, opts)
}
