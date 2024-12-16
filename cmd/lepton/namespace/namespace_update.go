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

	"github.com/containerd/nerdctl/v2/cmd/lepton/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/leptonic/utils"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
)

type namespaceUpdateOptions namespaceCreateOptions

func newNamespacelabelUpdateCommand() *cobra.Command {
	namespaceLableCommand := &cobra.Command{
		Use:           "update [flags] NAMESPACE",
		Short:         "Update labels for a namespace",
		RunE:          labelUpdateAction,
		Args:          cobra.MinimumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	namespaceLableCommand.Flags().StringArrayP("label", "l", nil, "Set labels for a namespace")

	return namespaceLableCommand
}

func processNamespaceUpdateCommandOption(cmd *cobra.Command) (*types.GlobalCommandOptions, *namespaceUpdateOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, nil, err
	}

	labels, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		return &globalOptions, nil, err
	}

	return &globalOptions, &namespaceUpdateOptions{Labels: utils.StringSlice2KVMap(labels, "=")}, nil
}

func labelUpdateAction(cmd *cobra.Command, args []string) error {
	globalOptions, options, err := processNamespaceUpdateCommandOption(cmd)
	if err != nil {
		return err
	}

	client, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	errs := namespace.Update(ctx, client, args[0], options.Labels)
	if len(errs) > 0 {
		for _, err = range errs {
			log.G(ctx).WithError(err).Error()
		}

		return errors.New("an error occurred")
	}

	return nil
}
