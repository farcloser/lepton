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
	"fmt"
	"strings"

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
	systemPruneCommand := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove unused data",
		Args:          cobra.NoArgs,
		RunE:          pruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	systemPruneCommand.Flags().BoolP("all", "a", false, "Remove all unused images, not just dangling ones")
	systemPruneCommand.Flags().BoolP("force", "f", false, "Do not prompt for confirmation")
	systemPruneCommand.Flags().Bool("volumes", false, "Prune volumes")
	return systemPruneCommand
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

	return &options.SystemPrune{
		Stdout:               cmd.OutOrStdout(),
		Stderr:               cmd.ErrOrStderr(),
		All:                  all,
		Volumes:              vFlag,
		NetworkDriversToKeep: network.NetworkDriversToKeep,
	}, nil
}

func grantSystemPrunePermission(cmd *cobra.Command, options *options.SystemPrune) (bool, error) {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return false, err
	}

	if !force {
		var confirm string
		msg := `This will remove:
  - all stopped containers
  - all networks not used by at least one container`
		if options.Volumes {
			msg += `
  - all volumes not used by at least one container`
		}
		if options.All {
			msg += `
  - all images without at least one container associated to them
  - all build cache`
		} else {
			msg += `
  - all dangling images
  - all dangling build cache`
		}

		msg += "\nAre you sure you want to continue? [y/N] "
		fmt.Fprintf(options.Stdout, "WARNING! %s", msg)
		fmt.Fscanf(cmd.InOrStdin(), "%s", &confirm)

		if strings.ToLower(confirm) != "y" {
			return false, nil
		}
	}

	return true, nil
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

	if ok, err := grantSystemPrunePermission(cmd, opts); err != nil {
		return err
	} else if !ok {
		return nil
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return system.Prune(ctx, cli, globalOptions, opts)
}
