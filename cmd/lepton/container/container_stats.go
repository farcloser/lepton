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

func StatsCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "stats",
		Short:             "Display a live stream of container(s) resource usage statistics.",
		RunE:              statsAction,
		ValidArgsFunction: statsShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().BoolP("all", "a", false, "Show all containers (default shows just running)")
	cmd.Flags().String("format", "", "Pretty-print images using a Go template, e.g, '{{json .}}'")
	cmd.Flags().Bool("no-stream", false, "Disable streaming stats and only pull the first result")
	cmd.Flags().Bool("no-trunc", false, "Do not truncate output")

	return cmd
}

func statsOptions(cmd *cobra.Command) (options.ContainerStats, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerStats{}, err
	}

	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return options.ContainerStats{}, err
	}

	noStream, err := cmd.Flags().GetBool("no-stream")
	if err != nil {
		return options.ContainerStats{}, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return options.ContainerStats{}, err
	}

	noTrunc, err := cmd.Flags().GetBool("no-trunc")
	if err != nil {
		return options.ContainerStats{}, err
	}

	return options.ContainerStats{
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.ErrOrStderr(),
		GOptions: globalOptions,
		All:      all,
		Format:   format,
		NoStream: noStream,
		NoTrunc:  noTrunc,
	}, nil
}

func statsAction(cmd *cobra.Command, args []string) error {
	opts, err := statsOptions(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Stats(ctx, cli, args, opts)
}

func statsShellComplete(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	// show running container names
	statusFilterFn := func(st client.ProcessStatus) bool {
		return st == client.Running
	}
	return completion.ContainerNames(cmd, statusFilterFn)
}
