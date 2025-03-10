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
	"errors"
	"os"

	"github.com/spf13/cobra"

	"go.farcloser.world/core/term"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/image"
)

func SaveCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "save",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Save one or more images to a tar archive (streamed to STDOUT by default)",
		Long:              "The archive implements both Docker Image Spec v1.2 and OCI Image Spec v1.0.",
		RunE:              saveAction,
		ValidArgsFunction: saveShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().StringP("output", "o", "", "Write to a file, instead of STDOUT")
	cmd.Flags().StringSlice("platform", []string{}, "Export content for a specific platform")
	cmd.Flags().Bool("all-platforms", false, "Export content for all platforms")

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

func saveOptions(cmd *cobra.Command, _ []string) (options.ImageSave, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ImageSave{}, err
	}

	allPlatforms, err := cmd.Flags().GetBool("all-platforms")
	if err != nil {
		return options.ImageSave{}, err
	}
	platform, err := cmd.Flags().GetStringSlice("platform")
	if err != nil {
		return options.ImageSave{}, err
	}

	return options.ImageSave{
		GOptions:     globalOptions,
		AllPlatforms: allPlatforms,
		Platform:     platform,
	}, err
}

func saveAction(cmd *cobra.Command, args []string) error {
	opts, err := saveOptions(cmd, args)
	if err != nil {
		return err
	}

	output := cmd.OutOrStdout()
	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	} else if outputPath != "" {
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		output = f
		defer f.Close()
	} else if out, ok := output.(*os.File); ok && term.IsTerminal(out.Fd()) {
		return errors.New("cowardly refusing to save to a terminal. Use the -o flag or redirect")
	}
	opts.Stdout = output

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	if err = image.Save(ctx, cli, args, opts); err != nil && outputPath != "" {
		os.Remove(outputPath)
	}
	return err
}

func saveShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show image names
	return completion.ImageNames(cmd, args, toComplete)
}
