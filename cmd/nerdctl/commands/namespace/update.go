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

func updateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "update [flags] NAMESPACE",
		Short:         "Update labels for a namespace",
		RunE:          updateAction,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringArrayP(flagLabel, "l", nil, "Set labels for a namespace")

	return cmd
}

func updateOptions(cmd *cobra.Command, args []string) (*options.NamespaceCreate, error) {
	labels, err := cmd.Flags().GetStringArray(flagLabel)
	if err != nil {
		return nil, err
	}

	return &options.NamespaceCreate{
		Name:   args[0],
		Labels: utils.StringSlice2KVMap(labels, "="),
	}, nil
}

func updateAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := updateOptions(cmd, args)
	if err != nil {
		return err
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return namespace.Update(ctx, client, globalOptions, opts)

}
