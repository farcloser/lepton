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

package volume

import (
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/volume"
	"go.farcloser.world/lepton/pkg/formatter"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Aliases:       []string{"list"},
		Short:         "List volumes",
		Args:          cobra.NoArgs,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("quiet", "q", false, "Only display volume names")
	cmd.Flags().String("format", "", "Format the output using the given go template")
	cmd.Flags().BoolP("size", "s", false, "Display the disk usage of volumes. Can be slow with volumes having loads of directories.")
	cmd.Flags().StringSliceP("filter", "f", []string{}, "Filter matches volumes based on given conditions")

	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON, formatter.FormatTable, formatter.FormatWide}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func listOptions(cmd *cobra.Command, _ []string) (*options.VolumeList, error) {
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}

	size, err := cmd.Flags().GetBool("size")
	if err != nil {
		return nil, err
	}

	filters, err := cmd.Flags().GetStringSlice("filter")
	if err != nil {
		return nil, err
	}

	return &options.VolumeList{
		Quiet:   quiet,
		Format:  format,
		Size:    size,
		Filters: filters,
	}, nil
}

func listAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := listOptions(cmd, args)
	if err != nil {
		return err
	}

	return volume.List(cmd.Context(), cmd.OutOrStdout(), globalOptions, opts)
}
