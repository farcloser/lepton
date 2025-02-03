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

package volume

import (
	"errors"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/volume"
	"go.farcloser.world/lepton/pkg/utils"
)

func createCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "create [flags] [VOLUME]",
		Short:         "Create a volume",
		Args:          cobra.MaximumNArgs(1),
		RunE:          createAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringArray("label", nil, "Set a label on the volume")

	return cmd
}

func createOptions(cmd *cobra.Command, args []string) (*options.VolumeCreate, error) {
	labels, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		if label == "" {
			return nil, errors.Join(errs.ErrInvalidArgument, errors.New("labels cannot be empty"))
		}
	}

	lbls := map[string]string{}
	if len(labels) > 0 {
		lbls = utils.KeyValueStringsToMap(labels)
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	return &options.VolumeCreate{
		Name:   name,
		Labels: lbls,
	}, nil
}

func createAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := createOptions(cmd, args)
	if err != nil {
		return err
	}

	return volume.Create(cmd.Context(), cmd.OutOrStdout(), globalOptions, opts)
}
