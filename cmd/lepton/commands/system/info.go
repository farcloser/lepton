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

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/system"
	"go.farcloser.world/lepton/pkg/formatter"
)

func InfoCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "info",
		Args:          cobra.NoArgs,
		Short:         "Display information about the system",
		RunE:          infoAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().String("mode", "dockercompat", `Information mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output`)
	cmd.Flags().StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")

	_ = cmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dockercompat", "native"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func infoOptions(cmd *cobra.Command, _ []string) (*options.SystemInfo, error) {
	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}

	return &options.SystemInfo{
		Mode:   mode,
		Format: format,
		Stderr: cmd.OutOrStderr(),
	}, nil
}

func infoAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := infoOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return system.Info(ctx, cli, cmd.OutOrStdout(), globalOptions, opts)
}
