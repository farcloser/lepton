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

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/errs"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/volume"
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

	cmd.Flags().StringArray(flagLabel, nil, "Set a label on the volume")

	return cmd
}

func createOptions(cmd *cobra.Command, args []string) (*options.VolumeCreate, error) {
	labels, err := cmd.Flags().GetStringArray(flagLabel)
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		if label == "" {
			return nil, fmt.Errorf("labels cannot be empty (%w)", errs.ErrInvalidArgument)
		}
	}

	volumeName := ""
	if len(args) > 0 {
		volumeName = args[0]
	}

	return &options.VolumeCreate{
		Name:   volumeName,
		Labels: labels,
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
