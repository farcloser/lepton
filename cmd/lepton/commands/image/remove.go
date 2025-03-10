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
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/image"
)

func RemoveCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "rmi [flags] IMAGE [IMAGE, ...]",
		Short:             "Remove one or more images",
		Args:              cobra.MinimumNArgs(1),
		RunE:              rmiAction,
		ValidArgsFunction: rmiShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().BoolP("force", "f", false, "Force removal of the image")
	cmd.Flags().Bool("async", false, "Asynchronous mode")

	return cmd
}

func removeOptions(cmd *cobra.Command, _ []string) (options.ImageRemove, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImageRemove{}, err
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return options.ImageRemove{}, err
	}
	async, err := cmd.Flags().GetBool("async")
	if err != nil {
		return options.ImageRemove{}, err
	}

	return options.ImageRemove{
		Stdout:   cmd.OutOrStdout(),
		GOptions: globalOptions,
		Force:    force,
		Async:    async,
	}, nil
}

func rmiAction(cmd *cobra.Command, args []string) error {
	opts, err := removeOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Remove(ctx, cli, args, opts)
}

func rmiShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show image names
	return completion.ImageNames(cmd, args, toComplete)
}
