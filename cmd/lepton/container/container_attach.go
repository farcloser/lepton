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

func AttachCommand() *cobra.Command {
	const shortHelp = "Attach stdin, stdout, and stderr to a running container."
	const longHelp = `Attach stdin, stdout, and stderr to a running container. For example:

1. 'nerdctl run -it --name test busybox' to start a container with a pty
2. 'ctrl-p ctrl-q' to detach from the container
3. 'nerdctl attach test' to attach to the container

Caveats:

- Currently only one attach session is allowed. When the second session tries to attach, currently no error will be returned from nerdctl.
  However, since behind the scenes, there's only one FIFO for stdin, stdout, and stderr respectively,
  if there are multiple sessions, all the sessions will be reading from and writing to the same 3 FIFOs, which will result in mixed input and partial output.
- Until dual logging (issue #1946) is implemented,
  a container that is spun up by either 'nerdctl run -d' or 'nerdctl start' (without '--attach') cannot be attached to.`

	var cmd = &cobra.Command{
		Use:               "attach [flags] CONTAINER",
		Args:              cobra.ExactArgs(1),
		Short:             shortHelp,
		Long:              longHelp,
		RunE:              attachAction,
		ValidArgsFunction: attachShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().String("detach-keys", consoleutil.DefaultDetachKeys, "Override the default detach keys")

	return cmd
}

func attachOptions(cmd *cobra.Command, _ []string) (options.ContainerAttach, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerAttach{}, err
	}
	detachKeys, err := cmd.Flags().GetString("detach-keys")
	if err != nil {
		return options.ContainerAttach{}, err
	}
	return options.ContainerAttach{
		GOptions:   globalOptions,
		Stdin:      cmd.InOrStdin(),
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.ErrOrStderr(),
		DetachKeys: detachKeys,
	}, nil
}

func attachAction(cmd *cobra.Command, args []string) error {
	opts, err := attachOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Attach(ctx, cli, args[0], opts)
}

func attachShellComplete(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st == client.Running
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
