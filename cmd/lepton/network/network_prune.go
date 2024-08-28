/*
   Copyright The containerd Authors.

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

package network

import (
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/cmd/network"
)

func NewNetworkPruneCommand() *cobra.Command {
	networkPruneCommand := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove all unused networks",
		Args:          cobra.NoArgs,
		RunE:          networkPruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	networkPruneCommand.Flags().BoolP("force", "f", false, "Do not prompt for confirmation")
	return networkPruneCommand
}

func networkPruneAction(cmd *cobra.Command, _ []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return err
	}

	if !force {
		msg := "This will remove all custom networks not used by at least one container."

		var confirmed bool
		if confirmed, err = helpers.Confirm(cmd, msg); err != nil || !confirmed {
			return err
		}
	}
	options := types.NetworkPruneOptions{
		GOptions:             globalOptions,
		NetworkDriversToKeep: helpers.NetworkDriversToKeep,
		Stdout:               cmd.OutOrStdout(),
	}

	client, ctx, cancel, err := clientutil.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return network.Prune(ctx, client, options)
}
