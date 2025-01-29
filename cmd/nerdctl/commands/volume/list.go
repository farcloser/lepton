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

	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/volume"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Aliases:       []string{"ls"},
		Short:         "List volumes",
		Args:          cobra.NoArgs,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP(flagQuiet, "q", false, "Only display volume names")
	cmd.Flags().String(flagFormat, "", "Format the output using the given go template")
	cmd.Flags().BoolP(flagSize, "s", false, "Display the disk usage of volumes. Can be slow with volumes having loads of directories.")
	cmd.Flags().StringSliceP(flagFilter, "f", []string{}, "Filter matches volumes based on given conditions")

	_ = cmd.RegisterFlagCompletionFunc(flagFormat, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON, formatter.FormatTable, formatter.FormatWide}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func listOptions(cmd *cobra.Command, _ []string) (*options.VolumeList, error) {
	quiet, err := cmd.Flags().GetBool(flagQuiet)
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return nil, err
	}

	size, err := cmd.Flags().GetBool(flagSize)
	if err != nil {
		return nil, err
	}

	filters, err := cmd.Flags().GetStringSlice(flagFilter)
	if err != nil {
		return nil, err
	}

	if quiet && size {
		log.L.Warn("cannot use --size and --quiet together, ignoring --size")
		size = false
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
