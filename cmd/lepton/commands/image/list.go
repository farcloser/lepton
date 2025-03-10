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

	"go.farcloser.world/containers/reference"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/image"
	"go.farcloser.world/lepton/pkg/formatter"
)

func ListCommand() *cobra.Command {
	longHelp := `

Properties:
- REPOSITORY: Repository
- TAG:        Tag
- NAME:       Name of the image, --names for skip parsing as repository and tag.
- IMAGE ID:   OCI Digest. Usually different from Docker image ID. Shared for multi-platform images.
- CREATED:    Created time
- PLATFORM:   Platform
- SIZE:       Size of the unpacked snapshots
- BLOB SIZE:  Size of the blobs (such as layer tarballs) in the content store
`
	var cmd = &cobra.Command{
		Use:                   "images [flags] [REPOSITORY[:TAG]]",
		Short:                 "List images",
		Long:                  longHelp,
		Args:                  cobra.MaximumNArgs(1),
		RunE:                  imagesAction,
		ValidArgsFunction:     imagesShellComplete,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
	}

	cmd.Flags().BoolP("quiet", "q", false, "Only show numeric IDs")
	cmd.Flags().Bool("no-trunc", false, "Don't truncate output")
	// Alias "-f" is reserved for "--filter"
	cmd.Flags().String("format", "", "Format the output using the given Go template, e.g, '{{json .}}', 'wide'")
	cmd.Flags().StringSliceP("filter", "f", []string{}, "Filter output based on conditions provided")
	cmd.Flags().Bool("digests", false, "Show digests (compatible with Docker, unlike ID)")
	cmd.Flags().Bool("names", false, "Show image names")
	cmd.Flags().BoolP("all", "a", true, "(unimplemented yet, always true)")

	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON, formatter.FormatTable, formatter.FormatWide}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func listOptions(cmd *cobra.Command, args []string) (*options.ImageList, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}
	var filters []string
	if len(args) > 0 {
		parsedReference, err := reference.Parse(args[0])
		if err != nil {
			return nil, err
		}
		filters = []string{"name==" + parsedReference.String()}
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return nil, err
	}
	noTrunc, err := cmd.Flags().GetBool("no-trunc")
	if err != nil {
		return nil, err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, err
	}
	var inputFilters []string
	if cmd.Flags().Changed("filter") {
		inputFilters, err = cmd.Flags().GetStringSlice("filter")
		if err != nil {
			return nil, err
		}
	}
	digests, err := cmd.Flags().GetBool("digests")
	if err != nil {
		return nil, err
	}
	names, err := cmd.Flags().GetBool("names")
	if err != nil {
		return nil, err
	}
	return &options.ImageList{
		GOptions:         globalOptions,
		Quiet:            quiet,
		NoTrunc:          noTrunc,
		Format:           format,
		Filters:          inputFilters,
		NameAndRefFilter: filters,
		Digests:          digests,
		Names:            names,
		All:              true,
		Stdout:           cmd.OutOrStdout(),
	}, nil

}

func imagesAction(cmd *cobra.Command, args []string) error {
	opts, err := listOptions(cmd, args)
	if err != nil {
		return err
	}
	if !opts.All {
		opts.All = true
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return image.ListCommandHandler(ctx, cli, opts)
}

func imagesShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// show image names
		return completion.ImageNames(cmd, args, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
