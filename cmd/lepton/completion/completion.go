package completion

import (
	"context"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/cmd/volume"
	"github.com/farcloser/lepton/pkg/labels"
	"github.com/farcloser/lepton/pkg/netutil"
)

func VolumeNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	candidates, err := volume.VolumesNames(globalOptions.DataRoot, globalOptions.Address, globalOptions.Namespace)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func ImageNames(cmd *cobra.Command) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	client, ctx, cancel, err := clientutil.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	defer cancel()

	// FIXME: @apostasie: return short and familiar names instead
	imageList, err := client.ImageService().List(ctx, "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	candidates := []string{}
	for _, img := range imageList {
		candidates = append(candidates, img.Name)
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func ContainerNames(cmd *cobra.Command, filterFunc func(containerd.ProcessStatus) bool) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	client, ctx, cancel, err := clientutil.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	defer cancel()

	containers, err := client.Containers(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	getStatus := func(container containerd.Container) containerd.ProcessStatus {
		toutCtx, toutCancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer toutCancel()

		task, err := container.Task(toutCtx, nil)
		if err != nil {
			return containerd.Unknown
		}

		st, err := task.Status(toutCtx)
		if err != nil {
			return containerd.Unknown
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

	netConfigs, err := e.NetworkMap()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	candidates := []string{}
	for _, network := range netConfigs {
		if _, ok := excludeMap[network.Name]; !ok {
			candidates = append(candidates, network.Name)
		}
	}

	for _, network := range []string{"host", "none"} {
		if _, ok := excludeMap[network]; !ok {
			candidates = append(candidates, network)
		}
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func PlatformNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	candidates := []string{
		"windows/amd64",
		"windows/arm64",
		"linux/amd64",
		"linux/arm64",
		"linux/riscv64",
		"linux/ppc64le",
		"linux/s390x",
		"linux/386",
		"linux/arm/v7",
		"linux/arm/v6",
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}
