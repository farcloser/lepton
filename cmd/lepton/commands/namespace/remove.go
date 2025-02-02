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

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/namespace"
)

func removeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "remove [flags] NAMESPACE [NAMESPACE...]",
		Aliases:           []string{"rm"},
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completion.NamespaceNames,
		Short:             "Remove one or more namespaces",
		RunE:              removeAction,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().BoolP("cgroup", "c", false, "delete the namespace's cgroup")

	return cmd
}

func removeOptions(cmd *cobra.Command, args []string) (*options.Global, *options.NamespaceRemove, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, nil, err
	}

	cgroup, err := cmd.Flags().GetBool("cgroup")
	if err != nil {
		return globalOptions, nil, err
	}

	return globalOptions, &options.NamespaceRemove{
		NamesList: args,
		CGroup:    cgroup,
	}, nil
}

func removeAction(cmd *cobra.Command, args []string) error {
	globalOptions, opts, err := removeOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	return namespace.Remove(ctx, cli, globalOptions, opts)
}
