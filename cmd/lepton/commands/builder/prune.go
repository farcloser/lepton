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

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/builder"
)

func pruneCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "prune",
		Args:          cobra.NoArgs,
		Short:         "Clean up BuildKit build cache",
		RunE:          pruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	helpers.AddStringFlag(cmd, "buildkit-host", nil, "", "BUILDKIT_HOST", "BuildKit address")
	cmd.Flags().BoolP("all", "a", false, "Remove all unused build cache, not just dangling ones")
	cmd.Flags().BoolP("force", "f", false, "Do not prompt for confirmation")

	return cmd
}

func pruneOptions(cmd *cobra.Command, _ []string) (*options.BuilderPrune, error) {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return nil, err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return nil, err
	}

	if !force {
		var msg string

		if all {
			msg = "This will remove all build cache."
		} else {
			msg = "This will remove any dangling build cache."
		}

		if err := helpers.Confirm(cmd, fmt.Sprintf("WARNING! %s.", msg)); err != nil {
			return nil, err
		}
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

	opts.BuildKitHost, err = GetBuildkitHostOption(cmd, globalOptions.Namespace)
	if err != nil {
		return err
	}

	prunedObjects, err := builder.Prune(cmd.Context(), globalOptions, opts)
	if err != nil {
		return err
	}

	var totalReclaimedSpace int64

	for _, prunedObject := range prunedObjects {
		totalReclaimedSpace += prunedObject.Size
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Total:  %s\n", units.BytesSize(float64(totalReclaimedSpace)))

	return err
}
