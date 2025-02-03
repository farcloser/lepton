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
	"go.farcloser.world/lepton/pkg/consoleutil"
)

func StartCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "start [flags] CONTAINER [CONTAINER, ...]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Start one or more running containers",
		RunE:              startAction,
		ValidArgsFunction: startShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().BoolP("attach", "a", false, "Attach STDOUT/STDERR and forward signals")
	cmd.Flags().String("detach-keys", consoleutil.DefaultDetachKeys, "Override the default detach keys")

	return cmd
}

func startOptions(cmd *cobra.Command, _ []string) (*options.ContainerStart, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	attach, err := cmd.Flags().GetBool("attach")
	if err != nil {
		return nil, err
	}

	detachKeys, err := cmd.Flags().GetString("detach-keys")
	if err != nil {
		return nil, err
	}

	return &options.ContainerStart{
		Stdout:     cmd.OutOrStdout(),
		GOptions:   globalOptions,
		Attach:     attach,
		DetachKeys: detachKeys,
	}, nil
}

func startAction(cmd *cobra.Command, args []string) error {
	opts, err := startOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return container.Start(ctx, cli, args, opts)
}

func startShellComplete(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	// show non-running container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st != client.Running && st != client.Unknown
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
