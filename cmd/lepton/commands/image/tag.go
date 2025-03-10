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
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/image"
)

func TagCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "tag [flags] SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		Short:             "Create a tag TARGET_IMAGE that refers to SOURCE_IMAGE",
		Args:              helpers.IsExactArgs(2),
		RunE:              tagAction,
		ValidArgsFunction: tagShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
}

func tagAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts := options.ImageTag{
		GOptions: globalOptions,
		Source:   args[0],
		Target:   args[1],
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.Tag(ctx, cli, opts)
}

func tagShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) < 2 {
		// show image names
		return completion.ImageNames(cmd, args, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
