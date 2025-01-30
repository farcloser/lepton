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
	"errors"

	"github.com/spf13/cobra"
	"go.farcloser.world/containers/security/cgroups"

	"github.com/containerd/containerd/v2/client"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/clientutil"
	"github.com/containerd/nerdctl/v2/pkg/cmd/container"
	"github.com/containerd/nerdctl/v2/pkg/rootlessutil"
)

func NewTopCommand() *cobra.Command {
	var topCommand = &cobra.Command{
		Use:               "top CONTAINER [ps OPTIONS]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Display the running processes of a container",
		RunE:              topAction,
		ValidArgsFunction: topShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	topCommand.Flags().SetInterspersed(false)
	return topCommand
}

func topAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	if rootlessutil.IsRootless() && cgroups.Version() < 2 {
		return errors.New("top requires cgroup v2 for rootless containers, see https://rootlesscontaine.rs/getting-started/common/cgroup2/")
	}

	if globalOptions.CgroupManager == cgroups.NoneManager {
		return errors.New("cgroup manager must not be \"none\"")
	}
	cli, ctx, cancel, err := clientutil.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()
	return container.Top(ctx, cli, args, types.ContainerTopOptions{
		Stdout:   cmd.OutOrStdout(),
		GOptions: globalOptions,
	})

}

func topShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show running container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st == client.Running
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
