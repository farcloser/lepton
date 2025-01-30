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
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/leptonic/utils"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
)

type namespaceCreateOptions struct {
	// Labels are the namespace labels
	Labels map[string]string
}

func createCommand() *cobra.Command {
	namespaceCreateCommand := &cobra.Command{
		Use:           "create NAMESPACE",
		Short:         "Create a new namespace",
		Args:          cobra.MinimumNArgs(1),
		RunE:          namespaceCreateAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	namespaceCreateCommand.Flags().StringArrayP("label", "l", nil, "Set labels for a namespace")
	return namespaceCreateCommand
}

func processNamespaceCreateCommandOption(cmd *cobra.Command) (*options.Global, *namespaceUpdateOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, nil, err
	}
	labels, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		return globalOptions, nil, err
	}
	return globalOptions, &namespaceUpdateOptions{Labels: utils.StringSlice2KVMap(labels, "=")}, nil
}

func namespaceCreateAction(cmd *cobra.Command, args []string) error {
	globalOptions, options, err := processNamespaceCreateCommandOption(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	name := args[0]

	return namespace.Create(ctx, cli, name, options.Labels)

}
