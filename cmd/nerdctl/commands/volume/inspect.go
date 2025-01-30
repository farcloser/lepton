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

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/volume"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
)

func inspectCommand() *cobra.Command {
	volumeInspectCommand := &cobra.Command{
		Use:               "inspect [flags] VOLUME [VOLUME...]",
		Short:             "Display detailed information on one or more volumes",
		Args:              cobra.MinimumNArgs(1),
		RunE:              inspectAction,
		ValidArgsFunction: volumeInspectShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	volumeInspectCommand.Flags().StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")
	volumeInspectCommand.Flags().BoolP("size", "s", false, "Display the disk usage of the volume")
	volumeInspectCommand.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})
	return volumeInspectCommand
}

func inspectOptions(cmd *cobra.Command, _ []string) (*options.VolumeInspect, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}
	volumeSize, err := cmd.Flags().GetBool("size")
	if err != nil {
		return nil, err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}
	return &options.VolumeInspect{
		GOptions: globalOptions,
		Format:   format,
		Size:     volumeSize,
		Stdout:   cmd.OutOrStdout(),
	}, nil
}

func inspectAction(cmd *cobra.Command, args []string) error {
	options, err := inspectOptions(cmd, args)
	if err != nil {
		return err
	}
	return volume.Inspect(cmd.Context(), args, options)
}

func volumeInspectShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show volume names
	return completion.VolumeNames(cmd)
}
