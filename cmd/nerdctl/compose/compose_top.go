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

package compose

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/containerd/containerd/v2/client"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/compose"
	"github.com/containerd/nerdctl/v2/pkg/cmd/container"
	"github.com/containerd/nerdctl/v2/pkg/containerutil"
	"github.com/containerd/nerdctl/v2/pkg/labels"
)

func topCommand() *cobra.Command {
	var composeTopCommand = &cobra.Command{
		Use:                   "top [SERVICE...]",
		Short:                 "Display the running processes of service containers",
		RunE:                  topAction,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
	}
	return composeTopCommand
}

func topAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()
	opts, err := getComposeOptions(cmd, globalOptions.DebugFull, globalOptions.Experimental)
	if err != nil {
		return err
	}
	c, err := compose.New(cli, globalOptions, opts, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	serviceNames, err := c.ServiceNames(args...)
	if err != nil {
		return err
	}
	containers, err := c.Containers(ctx, serviceNames...)
	if err != nil {
		return err
	}
	stdout := cmd.OutOrStdout()
	for _, c := range containers {
		cStatus, err := containerutil.ContainerStatus(ctx, c)
		if err != nil {
			return err
		}
		if cStatus.Status != client.Running {
			continue
		}

		info, err := c.Info(ctx, client.WithoutRefreshedMetadata)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, info.Labels[labels.Name])
		// `compose ps` uses empty ps args
		err = container.Top(ctx, cli, []string{c.ID()}, options.ContainerTop{
			Stdout:   cmd.OutOrStdout(),
			GOptions: globalOptions,
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout)
	}

	return nil
}
