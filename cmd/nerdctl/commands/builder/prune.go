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

package builder

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.farcloser.world/core/units"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/builder"
)

func pruneCommand() *cobra.Command {
	shortHelp := `Clean up BuildKit build cache`
	var buildPruneCommand = &cobra.Command{
		Use:           "prune",
		Args:          cobra.NoArgs,
		Short:         shortHelp,
		RunE:          pruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	helpers.AddStringFlag(buildPruneCommand, "buildkit-host", nil, "", "BUILDKIT_HOST", "BuildKit address")
	buildPruneCommand.Flags().BoolP(flagAll, "a", false, "Remove all unused build cache, not just dangling ones")
	buildPruneCommand.Flags().BoolP(flagForce, "f", false, "Do not prompt for confirmation")

	return buildPruneCommand
}

func pruneOptions(cmd *cobra.Command, _ []string) (*options.BuilderPrune, error) {
	all, err := cmd.Flags().GetBool(flagAll)
	if err != nil {
		return nil, err
	}

	force, err := cmd.Flags().GetBool(flagForce)
	if err != nil {
		return nil, err
	}

	return &options.BuilderPrune{
		Stderr: cmd.OutOrStderr(),
		All:    all,
		Force:  force,
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

	opts.BuildKitHost, err = GetBuildkitHost(cmd, globalOptions.Namespace)
	if err != nil {
		return err
	}

	if !opts.Force {
		var msg string
		if opts.All {
			msg = "This will remove all build cache."
		} else {
			msg = "This will remove any dangling build cache."
		}

		if confirmed, err := helpers.Confirm(cmd, fmt.Sprintf("WARNING! %s.", msg)); err != nil || !confirmed {
			return err
		}
	}

	prunedObjects, err := builder.Prune(cmd.Context(), opts)
	if err != nil {
		return err
	}

	var totalReclaimedSpace int64
	for _, prunedObject := range prunedObjects {
		totalReclaimedSpace += prunedObject.Size
	}

	if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Total:  %s\n", units.BytesSize(float64(totalReclaimedSpace))); err != nil {
		return err
	}

	return nil
}
