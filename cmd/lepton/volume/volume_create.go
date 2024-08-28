package volume

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/cmd/volume"
	"github.com/farcloser/lepton/pkg/errs"
)

func NewVolumeCreateCommand() *cobra.Command {
	volumeCreateCommand := &cobra.Command{
		Use:           "create [flags] [VOLUME]",
		Short:         "Create a volume",
		Args:          cobra.MaximumNArgs(1),
		RunE:          volumeCreateAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	volumeCreateCommand.Flags().StringArray(flagLabel, nil, "Set a label on the volume")

	// FIXME: @apostasie implement drivers and opts

	return volumeCreateCommand
}

func processVolumeCreateOptions(cmd *cobra.Command) (*types.VolumeCreateOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	labels, err := cmd.Flags().GetStringArray(flagLabel)
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		if label == "" {
			return nil, fmt.Errorf("labels cannot be empty (%w)", errs.ErrInvalidArgument)
		}
	}

	return &types.VolumeCreateOptions{
		GOptions: globalOptions,
		Labels:   labels,
	}, nil
}

func volumeCreateAction(cmd *cobra.Command, args []string) error {
	options, err := processVolumeCreateOptions(cmd)
	if err != nil {
		return err
	}

	volumeName := ""
	if len(args) > 0 {
		volumeName = args[0]
	}

	_, err = volume.Create(volumeName, options)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), volumeName)
	return err
}
