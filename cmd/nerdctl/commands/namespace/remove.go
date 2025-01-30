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
	"errors"

	"github.com/spf13/cobra"

	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
)

type namespaceRemoveOptions struct {
	// CGroup delete the namespace's cgroup
	CGroup bool
}

func removeCommand() *cobra.Command {
	namespaceRmCommand := &cobra.Command{
		Use:               "remove [flags] NAMESPACE [NAMESPACE...]",
		Aliases:           []string{"rm"},
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completion.NamespaceNames,
		Short:             "Remove one or more namespaces",
		RunE:              removeAction,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	namespaceRmCommand.Flags().BoolP("cgroup", "c", false, "delete the namespace's cgroup")

	return namespaceRmCommand
}

func removeOptions(cmd *cobra.Command, _ []string) (*options.Global, *namespaceRemoveOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, nil, err
	}

	cgroup, err := cmd.Flags().GetBool("cgroup")
	if err != nil {
		return globalOptions, nil, err
	}

	return globalOptions, &namespaceRemoveOptions{
		CGroup: cgroup,
	}, nil
}

func removeAction(cmd *cobra.Command, args []string) error {
	globalOptions, options, err := removeOptions(cmd, args)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	errs := namespace.Remove(ctx, cli, args, options.CGroup)

	if len(errs) > 0 {
		for _, err = range errs {
			log.G(ctx).WithError(err).Error()
		}

		return errors.New("failed to remove namespaces")
	}

	return nil
}
