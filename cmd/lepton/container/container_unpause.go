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
	"github.com/containerd/containerd/v2/client"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
)

func UnpauseCommand() *cobra.Command {
	var unpauseCommand = &cobra.Command{
		Use:               "unpause [flags] CONTAINER [CONTAINER, ...]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Unpause all processes within one or more containers",
		RunE:              unpauseAction,
		ValidArgsFunction: unpauseShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	return unpauseCommand
}

func unpauseOptions(cmd *cobra.Command, _ []string) (options.ContainerUnpauseOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerUnpauseOptions{}, err
	}
	return options.ContainerUnpauseOptions{
		GOptions: globalOptions,
		Stdout:   cmd.OutOrStdout(),
	}, nil
}

func unpauseAction(cmd *cobra.Command, args []string) error {
	options, err := unpauseOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Unpause(ctx, cli, args, options)
}

func unpauseShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show paused container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st == client.Paused
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
