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

package inspect

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	imageCmd "go.farcloser.world/lepton/cmd/lepton/commands/image"
	"go.farcloser.world/lepton/cmd/lepton/completion"
	containerCmd "go.farcloser.world/lepton/cmd/lepton/container"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
	"go.farcloser.world/lepton/pkg/cmd/image"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
	"go.farcloser.world/lepton/pkg/idutil/imagewalker"
)

func Command() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "inspect",
		Short:             "Return low-level information on objects.",
		Args:              cobra.MinimumNArgs(1),
		RunE:              inspectAction,
		ValidArgsFunction: inspectShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().BoolP("size", "s", false, "Display total file sizes (for containers)")
	cmd.Flags().StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")
	cmd.Flags().String("type", "", "Return JSON for specified type")
	cmd.Flags().String("mode", "dockercompat", `Inspect mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output`)

	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"image", "container", ""}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("mode", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dockercompat", "native"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

var validInspectType = map[string]bool{
	"container": true,
	"image":     true,
}

func inspectAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	namespace := globalOptions.Namespace
	address := globalOptions.Address
	inspectType, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}

	if len(inspectType) > 0 && !validInspectType[inspectType] {
		return fmt.Errorf("%q is not a valid value for --type", inspectType)
	}

	// container and image inspect can share the same cli, since no `platform`
	// flag will be passed for image inspect.
	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), namespace, address)
	if err != nil {
		return err
	}
	defer cancel()

	iwalker := &imagewalker.ImageWalker{
		Client: cli,
		OnFound: func(ctx context.Context, found imagewalker.Found) error {
			return nil
		},
	}

	cwalker := &containerwalker.ContainerWalker{
		Client: cli,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			return nil
		},
	}

	inspectImage := len(inspectType) == 0 || inspectType == "image"
	inspectContainer := len(inspectType) == 0 || inspectType == "container"

	var imageInspectOptions options.ImageInspect
	var containerInspectOptions options.ContainerInspect
	if inspectImage {
		platform := ""
		imageInspectOptions, err = imageCmd.ProcessImageInspectOptions(cmd, &platform)
		if err != nil {
			return err
		}
	}
	if inspectContainer {
		containerInspectOptions, err = containerCmd.ProcessContainerInspectOptions(cmd, args)
		if err != nil {
			return err
		}
	}

	var errs []error
	for _, req := range args {
		var ni int
		var nc int

		if inspectImage {
			ni, err = iwalker.Walk(ctx, req)
			if err != nil {
				return err
			}
		}
		if inspectContainer {
			nc, err = cwalker.Walk(ctx, req)
			if err != nil {
				return err
			}
		}

		if ni == 0 && nc == 0 {
			errs = append(errs, fmt.Errorf("no such object %s", req))
		} else if ni > 0 {
			if err := image.Inspect(ctx, cli, []string{req}, imageInspectOptions); err != nil {
				errs = append(errs, err)
			}
		} else if nc > 0 {
			if err := container.Inspect(ctx, cli, []string{req}, containerInspectOptions); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d errors: %v", len(errs), errs)
	}

	return nil
}

func inspectShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show container names
	containers, _ := completion.ContainerNames(cmd, nil)
	// show image names
	images, _ := completion.ImageNames(cmd, args, toComplete)
	return append(containers, images...), cobra.ShellCompDirectiveNoFileComp
}
