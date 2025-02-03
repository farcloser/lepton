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

package image

import (
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/cmd/image"
	"go.farcloser.world/lepton/pkg/formatter"
)

func inspectCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "inspect [flags] IMAGE [IMAGE...]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Display detailed information on one or more images.",
		Long:              "Hint: set `--mode=native` for showing the full output",
		RunE:              inspectAction,
		ValidArgsFunction: completion.ImageNames,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().String("mode", "dockercompat", `Inspect mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output`)
	cmd.Flags().StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")
	cmd.Flags().String("platform", "", "Inspect a specific platform") // not a slice, and there is no --all-platforms

	_ = cmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dockercompat", "native"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

func ProcessImageInspectOptions(cmd *cobra.Command, platform *string) (options.ImageInspect, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImageInspect{}, err
	}

	mode, err := cmd.Flags().GetString("mode")
	if err != nil {
		return options.ImageInspect{}, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return options.ImageInspect{}, err
	}

	if platform == nil {
		tempPlatform, err := cmd.Flags().GetString("platform")
		if err != nil {
			return options.ImageInspect{}, err
		}
		platform = &tempPlatform
	}

	return options.ImageInspect{
		GOptions: globalOptions,
		Mode:     mode,
		Format:   format,
		Platform: *platform,
		Stdout:   cmd.OutOrStdout(),
	}, nil
}

func inspectAction(cmd *cobra.Command, args []string) error {
	options, err := ProcessImageInspectOptions(cmd, nil)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := clientutil.NewClientWithPlatform(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address, options.Platform)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Inspect(ctx, cli, args, options)
}
