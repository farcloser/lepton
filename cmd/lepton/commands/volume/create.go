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
	"fmt"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/volume"
)

func createCommand() *cobra.Command {
	volumeCreateCommand := &cobra.Command{
		Use:           "create [flags] [VOLUME]",
		Short:         "Create a volume",
		Args:          cobra.MaximumNArgs(1),
		RunE:          createAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	volumeCreateCommand.Flags().StringArray("label", nil, "Set a label on the volume")
	return volumeCreateCommand
}

func createOptions(cmd *cobra.Command, _ []string) (*options.VolumeCreate, error) {
	labels, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		if label == "" {
			return &options.VolumeCreate{}, fmt.Errorf("labels cannot be empty (%w)", errs.ErrInvalidArgument)
		}
	}

	return &options.VolumeCreate{
		Labels: labels,
		Stdout: cmd.OutOrStdout(),
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

	volumeName := ""
	if len(args) > 0 {
		volumeName = args[0]
	}

	_, err = volume.Create(volumeName, globalOptions, opts)

	return err
}
