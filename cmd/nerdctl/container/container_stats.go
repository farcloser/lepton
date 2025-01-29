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
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/cmd/container"
)

func NewStatsCommand() *cobra.Command {
	var statsCommand = &cobra.Command{
		Use:               "stats",
		Short:             "Display a live stream of container(s) resource usage statistics.",
		RunE:              statsAction,
		ValidArgsFunction: statsShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	addStatsFlags(statsCommand)

	return statsCommand
}

func addStatsFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("all", "a", false, "Show all containers (default shows just running)")
	cmd.Flags().String("format", "", "Pretty-print images using a Go template, e.g, '{{json .}}'")
	cmd.Flags().Bool("no-stream", false, "Disable streaming stats and only pull the first result")
	cmd.Flags().Bool("no-trunc", false, "Do not truncate output")
}

func processStatsCommandFlags(cmd *cobra.Command) (types.ContainerStatsOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return types.ContainerStatsOptions{}, err
	}

	all, err := cmd.Flags().GetBool(flagAll)
	if err != nil {
		return types.ContainerStatsOptions{}, err
	}

	noStream, err := cmd.Flags().GetBool("no-stream")
	if err != nil {
		return types.ContainerStatsOptions{}, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return types.ContainerStatsOptions{}, err
	}

	noTrunc, err := cmd.Flags().GetBool("no-trunc")
	if err != nil {
		return types.ContainerStatsOptions{}, err
	}

	return types.ContainerStatsOptions{
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.ErrOrStderr(),
		GOptions: *globalOptions,
		All:      all,
		Format:   format,
		NoStream: noStream,
		NoTrunc:  noTrunc,
	}, nil
}

func statsAction(cmd *cobra.Command, args []string) error {
	options, err := processStatsCommandFlags(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return container.Stats(ctx, cli, args, options)
}

func statsShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show running container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st == client.Running
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
