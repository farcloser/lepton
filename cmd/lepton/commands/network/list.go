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

package network

import (
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/network"
	"go.farcloser.world/lepton/pkg/formatter"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Aliases:       []string{"list"},
		Short:         "List networks",
		Args:          cobra.NoArgs,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("quiet", "q", false, "Only display network IDs")
	cmd.Flags().StringSliceP("filter", "f", []string{}, "Provide filter values (e.g. \"name=default\")")
	cmd.Flags().String("format", "", "Format the output using the given Go template, e.g, '{{json .}}'")

	_ = cmd.RegisterFlagCompletionFunc(
		"format",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{
				formatter.FormatJSON,
				formatter.FormatTable,
				formatter.FormatWide,
			}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	return cmd
}

func listAction(cmd *cobra.Command, _ []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}

	filters, err := cmd.Flags().GetStringSlice("filter")
	if err != nil {
		return err
	}

	return network.List(cmd.Context(), globalOptions, &options.NetworkList{
		Quiet:   quiet,
		Format:  format,
		Filters: filters,
		Stdout:  cmd.OutOrStdout(),
	})
}
