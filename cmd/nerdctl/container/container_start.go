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

	"github.com/containerd/containerd/v2/client"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/container"
	"github.com/containerd/nerdctl/v2/pkg/consoleutil"
)

func StartCommand() *cobra.Command {
	var startCommand = &cobra.Command{
		Use:               "start [flags] CONTAINER [CONTAINER, ...]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Start one or more running containers",
		RunE:              startAction,
		ValidArgsFunction: startShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	startCommand.Flags().SetInterspersed(false)
	startCommand.Flags().BoolP("attach", "a", false, "Attach STDOUT/STDERR and forward signals")
	startCommand.Flags().String("detach-keys", consoleutil.DefaultDetachKeys, "Override the default detach keys")

	return startCommand
}

func startOptions(cmd *cobra.Command) (options.ContainerStart, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerStart{}, err
	}
	attach, err := cmd.Flags().GetBool("attach")
	if err != nil {
		return options.ContainerStart{}, err
	}
	detachKeys, err := cmd.Flags().GetString("detach-keys")
	if err != nil {
		return options.ContainerStart{}, err
	}
	return options.ContainerStart{
		Stdout:     cmd.OutOrStdout(),
		GOptions:   globalOptions,
		Attach:     attach,
		DetachKeys: detachKeys,
	}, nil
}

func startAction(cmd *cobra.Command, args []string) error {
	options, err := startOptions(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Start(ctx, cli, args, options)
}

func startShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show non-running container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st != client.Running && st != client.Unknown
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
