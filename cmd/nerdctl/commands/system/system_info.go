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

package system

import (
	"github.com/spf13/cobra"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/system"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
)

func NewInfoCommand() *cobra.Command {
	var infoCommand = &cobra.Command{
		Use:           "info",
		Args:          cobra.NoArgs,
		Short:         "Display information about the system",
		RunE:          infoAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	infoCommand.Flags().String("mode", "dockercompat", `Information mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output`)
	_ = infoCommand.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dockercompat", "native"}, cobra.ShellCompDirectiveNoFileComp
	})

	infoCommand.Flags().StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")
	_ = infoCommand.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})

	return infoCommand
}

func processInfoOptions(cmd *cobra.Command) (*options.SystemInfo, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}

	return &options.SystemInfo{
		GOptions: globalOptions,
		Mode:     mode,
		Format:   format,
		Stdout:   cmd.OutOrStdout(),
		Stderr:   cmd.OutOrStderr(),
	}, nil
}

func infoAction(cmd *cobra.Command, args []string) error {
	options, err := processInfoOptions(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return system.Info(ctx, cli, options)
}
