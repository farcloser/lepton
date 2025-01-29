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

	"github.com/containerd/nerdctl/v2/cmd/lepton/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
)

func NewNamespaceCommand() *cobra.Command {
	namespaceCommand := &cobra.Command{
		Annotations:   map[string]string{helpers.Category: helpers.Management},
		Use:           "namespace",
		Aliases:       []string{"ns"},
		Short:         "Manage containerd namespaces",
		Long:          "Unrelated to Linux namespaces and Kubernetes namespaces",
		RunE:          helpers.UnknownSubcommandAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	namespaceCommand.AddCommand(newNamespaceLsCommand())
	namespaceCommand.AddCommand(newNamespaceRmCommand())
	namespaceCommand.AddCommand(newNamespaceCreateCommand())
	namespaceCommand.AddCommand(newNamespacelabelUpdateCommand())
	namespaceCommand.AddCommand(newNamespaceInspectCommand())

	return namespaceCommand
}

func ShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	defer cancel()

	nsList, err := namespace.List(ctx, client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return nsList, cobra.ShellCompDirectiveNoFileComp
}
