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

func KillCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "kill [flags] CONTAINER [CONTAINER, ...]",
		Short:             "Kill one or more running containers",
		Args:              cobra.MinimumNArgs(1),
		RunE:              killAction,
		ValidArgsFunction: killShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().StringP("signal", "s", "KILL", "Signal to send to the container")

	return cmd
}

func killAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	killSignal, err := cmd.Flags().GetString("signal")
	if err != nil {
		return err
	}
	options := options.ContainerKill{
		GOptions:   globalOptions,
		KillSignal: killSignal,
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Kill(ctx, cli, args, options)
}

func killShellComplete(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	// show non-stopped container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st != client.Stopped && st != client.Created && st != client.Unknown
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
