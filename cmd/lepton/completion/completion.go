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

package completion

import (
	"context"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/leptonic/services/image"
	"go.farcloser.world/lepton/leptonic/services/namespace"
	"go.farcloser.world/lepton/pkg/cmd/volume"
	"go.farcloser.world/lepton/pkg/labels"
	"go.farcloser.world/lepton/pkg/netutil"
)

func VolumeNames(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	volumeSize, err := cmd.Flags().GetBool("size")
	if err != nil {
		// The `volume rm` does not have the flag `size`, so set it to false as the default value.
		volumeSize = false
	}

	vols, err := volume.Volumes(globalOptions.Namespace, globalOptions.DataRoot, globalOptions.Address, volumeSize, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	candidates := []string{}
	for _, v := range vols {
		candidates = append(candidates, v.Name)
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func ImageNames(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	defer cancel()

	candidates, err := image.ListNames(ctx, cli)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func NamespaceNames(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	defer cancel()

	nsList, err := namespace.ListNames(ctx, cli)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return nsList, cobra.ShellCompDirectiveNoFileComp
}

func ContainerNames(cmd *cobra.Command, filterFunc func(status client.ProcessStatus) bool) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	defer cancel()

	containers, err := cli.Containers(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	getStatus := func(c client.Container) client.ProcessStatus {
		ctx2, cancel2 := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel2()
		task, err := c.Task(ctx2, nil)
		if err != nil {
			return client.Unknown
		}
		st, err := task.Status(ctx2)
		if err != nil {
			return client.Unknown
		}
		return st.Status
	}

	candidates := []string{}
	for _, c := range containers {
		if filterFunc != nil {
			if !filterFunc(getStatus(c)) {
				continue
			}
		}
		lab, err := c.Labels(ctx)
		if err != nil {
			continue
		}
		name := lab[labels.Name]
		if name != "" {
			candidates = append(candidates, name)
			continue
		}
		candidates = append(candidates, c.ID())
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

// NetworkNames includes {"bridge","host","none"}
func NetworkNames(cmd *cobra.Command, exclude []string) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	excludeMap := make(map[string]struct{}, len(exclude))
	for _, ex := range exclude {
		excludeMap[ex] = struct{}{}
	}

	e, err := netutil.NewCNIEnv(globalOptions.CNIPath, globalOptions.CNINetConfPath, netutil.WithNamespace(globalOptions.Namespace))
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	candidates := []string{}
	netConfigs, err := e.NetworkMap()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	for netName, network := range netConfigs {
		if _, ok := excludeMap[netName]; !ok {
			candidates = append(candidates, netName)
			if network.CliID != nil {
				candidates = append(candidates, *network.CliID)
				candidates = append(candidates, (*network.CliID)[0:12])
			}
		}
	}
	for _, s := range []string{"host", "none"} {
		if _, ok := excludeMap[s]; !ok {
			candidates = append(candidates, s)
		}
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func Platforms(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"amd64",
		"arm64",
		"riscv64",
	}, cobra.ShellCompDirectiveNoFileComp
}
