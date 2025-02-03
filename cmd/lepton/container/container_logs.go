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
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
	"go.farcloser.world/lepton/pkg/version"
)

func LogsCommand() *cobra.Command {
	var shortUsage = fmt.Sprintf("Fetch the logs of a container. Expected to be used with '%s run -d'.", version.RootName)
	var longUsage = fmt.Sprintf(`Fetch the logs of a container.

The following containers are supported:
- Containers created with '%s run -d'. The log is currently empty for containers created without '-d'.
- Containers created with '%s compose'.
- Containers created with Kubernetes (EXPERIMENTAL).
`, version.RootName, version.RootName)
	var cmd = &cobra.Command{
		Use:               "logs [flags] CONTAINER",
		Args:              helpers.IsExactArgs(1),
		Short:             shortUsage,
		Long:              longUsage,
		RunE:              logsAction,
		ValidArgsFunction: logsShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().BoolP("timestamps", "t", false, "Show timestamps")
	cmd.Flags().StringP("tail", "n", "all", "Number of lines to show from the end of the logs")
	cmd.Flags().String("since", "", "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	cmd.Flags().String("until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")

	return cmd
}

func logsOptions(cmd *cobra.Command, _ []string) (options.ContainerLogs, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerLogs{}, err
	}
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return options.ContainerLogs{}, err
	}
	tailArg, err := cmd.Flags().GetString("tail")
	if err != nil {
		return options.ContainerLogs{}, err
	}
	var tail uint
	if tailArg != "" {
		tail, err = getTailArgAsUint(tailArg)
		if err != nil {
			return options.ContainerLogs{}, err
		}
	}
	timestamps, err := cmd.Flags().GetBool("timestamps")
	if err != nil {
		return options.ContainerLogs{}, err
	}
	since, err := cmd.Flags().GetString("since")
	if err != nil {
		return options.ContainerLogs{}, err
	}
	until, err := cmd.Flags().GetString("until")
	if err != nil {
		return options.ContainerLogs{}, err
	}
	return options.ContainerLogs{
		Stdout:     cmd.OutOrStdout(),
		Stderr:     cmd.OutOrStderr(),
		GOptions:   globalOptions,
		Follow:     follow,
		Timestamps: timestamps,
		Tail:       tail,
		Since:      since,
		Until:      until,
	}, nil
}

func logsAction(cmd *cobra.Command, args []string) error {
	options, err := logsOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Logs(ctx, cli, args[0], options)
}

func logsShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show container names (TODO: only show containers with logs)
	return completion.ContainerNames(cmd, nil)
}

// Attempts to parse the argument given to `-n/--tail` as an uint.
func getTailArgAsUint(arg string) (uint, error) {
	if arg == "all" {
		return 0, nil
	}
	num, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("failed to parse `-n/--tail` argument %q: %w", arg, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("`-n/--tail` argument must be positive, got: %d", num)
	}
	return uint(num), nil
}
