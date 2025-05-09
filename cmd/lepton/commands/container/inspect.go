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

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
	"go.farcloser.world/lepton/pkg/formatter"
)

func inspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "inspect [flags] CONTAINER [CONTAINER, ...]",
		Short:             "Display detailed information on one or more containers.",
		Long:              "Hint: set `--mode=native` for showing the full output",
		Args:              cobra.MinimumNArgs(1),
		RunE:              inspectAction,
		ValidArgsFunction: containerInspectShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().
		String("mode", "dockercompat", `Inspect mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output`)
	cmd.Flags().BoolP("size", "s", false, "Display total file sizes")
	cmd.Flags().StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")

	_ = cmd.RegisterFlagCompletionFunc(
		"mode",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"dockercompat", "native"}, cobra.ShellCompDirectiveNoFileComp
		},
	)
	_ = cmd.RegisterFlagCompletionFunc(
		"format",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	return cmd
}

var validModeType = map[string]bool{
	"native":       true,
	"dockercompat": true,
}

func ProcessContainerInspectOptions(cmd *cobra.Command, _ []string) (opt options.ContainerInspect, err error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return
	}
	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return
	}
	if len(mode) > 0 && !validModeType[mode] {
		err = fmt.Errorf("%q is not a valid value for --mode", mode)
		return
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return
	}

	size, err := cmd.Flags().GetBool("size")
	if err != nil {
		return
	}

	return options.ContainerInspect{
		GOptions: globalOptions,
		Format:   format,
		Mode:     mode,
		Size:     size,
		Stdout:   cmd.OutOrStdout(),
	}, nil
}

func inspectAction(cmd *cobra.Command, args []string) error {
	opt, err := ProcessContainerInspectOptions(cmd, args)
	if err != nil {
		return err
	}
	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opt.GOptions.Namespace, opt.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Inspect(ctx, cli, args, opt)
}

func containerInspectShellComplete(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	// show container names
	return completion.ContainerNames(cmd, nil)
}
