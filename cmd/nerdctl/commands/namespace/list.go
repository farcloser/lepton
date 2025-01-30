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

package namespace

import (
	"github.com/spf13/cobra"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/namespace"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Aliases:       []string{"list"},
		Short:         "ListNames containerd namespaces",
		Args:          cobra.NoArgs,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP(flagQuiet, "q", false, "Only display names")
	cmd.Flags().String(flagFormat, "", "Format the output using the given Go template, e.g, '{{json .}}'")

	return cmd
}

func listOptions(cmd *cobra.Command) (*options.NamespaceList, error) {
	quiet, err := cmd.Flags().GetBool(flagQuiet)
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return nil, err
	}

	return &options.NamespaceList{
		Quiet:  quiet,
		Format: format,
	}, nil
}

func listAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := listOptions(cmd)
	if err != nil {
		return err
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return namespace.List(ctx, client, cmd.OutOrStdout(), globalOptions, opts)
}
