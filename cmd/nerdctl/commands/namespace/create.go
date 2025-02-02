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
	"github.com/containerd/nerdctl/v2/leptonic/utils"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/namespace"
)

func createCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "create NAMESPACE",
		Short:         "Create a new namespace",
		Args:          cobra.ExactArgs(1),
		RunE:          createAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringArrayP("label", "l", nil, "Set labels for a namespace")

	return cmd
}

func createOption(cmd *cobra.Command, args []string) (*options.NamespaceCreate, error) {
	labels, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		return nil, err
	}

	return &options.NamespaceCreate{Name: args[0], Labels: utils.StringSlice2KVMap(labels, "=")}, nil
}

func createAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := createOption(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return namespace.Create(ctx, cli, globalOptions, opts)
}
