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

package system

import (
	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/commands/builder"
	"go.farcloser.world/lepton/cmd/lepton/commands/network"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/system"
)

func pruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove unused data",
		Args:          cobra.NoArgs,
		RunE:          pruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("all", "a", false, "Remove all unused images, not just dangling ones")
	cmd.Flags().BoolP("force", "f", false, "Do not prompt for confirmation")
	cmd.Flags().Bool("volumes", false, "Prune volumes")

	return cmd
}

func pruneOptions(cmd *cobra.Command, _ []string) (*options.SystemPrune, error) {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return nil, err
	}

	vFlag, err := cmd.Flags().GetBool("volumes")
	if err != nil {
		return nil, err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return nil, err
	}

	if !force {
		msg := `This will remove:
  - all stopped containers
  - all networks not used by at least one container`
		if vFlag {
			msg += `
  - all volumes not used by at least one container`
		}
		if all {
			msg += `
  - all images without at least one container associated to them
  - all build cache`
		} else {
			msg += `
  - all dangling images
  - all dangling build cache`
		}

		if err := helpers.Confirm(cmd, msg); err != nil {
			return nil, err
		}
	}

	return &options.SystemPrune{
		Stderr:               cmd.ErrOrStderr(),
		All:                  all,
		Volumes:              vFlag,
		NetworkDriversToKeep: network.NetworkDriversToKeep,
	}, nil
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

	buildkitHost, err := builder.GetBuildkitHostOption(cmd, globalOptions.Namespace)
	if err != nil {
		log.L.WithError(err).Warn("BuildKit is not running. Build caches will not be pruned.")
		buildkitHost = ""
	}

	opts.BuildKitHost = buildkitHost

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return system.Prune(ctx, cli, cmd.OutOrStdout(), globalOptions, opts)
}
