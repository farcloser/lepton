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

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
)

func CommitCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "commit [flags] CONTAINER REPOSITORY[:TAG]",
		Short:             "Create a new image from a container's changes",
		Args:              helpers.IsExactArgs(2),
		RunE:              commitAction,
		ValidArgsFunction: commitShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().StringP("author", "a", "", `Author (e.g., "contributor <dev@example.com>")`)
	cmd.Flags().StringP("message", "m", "", "Commit message")
	cmd.Flags().StringArrayP("change", "c", nil, "Apply Dockerfile instruction to the created image (supported directives: [CMD, ENTRYPOINT])")
	cmd.Flags().BoolP("pause", "p", true, "Pause container during commit")

	return cmd
}

func commitOptions(cmd *cobra.Command, _ []string) (options.ContainerCommit, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerCommit{}, err
	}

	author, err := cmd.Flags().GetString("author")
	if err != nil {
		return options.ContainerCommit{}, err
	}

	message, err := cmd.Flags().GetString("message")
	if err != nil {
		return options.ContainerCommit{}, err
	}

	pause, err := cmd.Flags().GetBool("pause")
	if err != nil {
		return options.ContainerCommit{}, err
	}

	change, err := cmd.Flags().GetStringArray("change")
	if err != nil {
		return options.ContainerCommit{}, err
	}

	return options.ContainerCommit{
		Stdout:   cmd.OutOrStdout(),
		GOptions: globalOptions,
		Author:   author,
		Message:  message,
		Pause:    pause,
		Change:   change,
	}, nil

}

func commitAction(cmd *cobra.Command, args []string) error {
	opts, err := commitOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return container.Commit(ctx, cli, args[1], args[0], opts)
}

func commitShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completion.ContainerNames(cmd, nil)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
