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

func InfoCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "info",
		Args:          cobra.NoArgs,
		Short:         "Display information about the system",
		RunE:          infoAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().String(flagMode, "dockercompat", `Information mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output`)
	cmd.Flags().StringP(flagFormat, "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")

	_ = cmd.RegisterFlagCompletionFunc(flagMode, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dockercompat", "native"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc(flagFormat, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func infoOptions(cmd *cobra.Command, _ []string) (*options.SystemInfo, error) {
	mode, err := cmd.Flags().GetString(flagMode)
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
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

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return system.Info(ctx, client, cmd.OutOrStdout(), globalOptions, opts)
}
