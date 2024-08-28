package volume

import (
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/completion"
	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/cmd/volume"
)

func NewVolumeRmCommand() *cobra.Command {
	volumeRmCommand := &cobra.Command{
		Use:               "rm [flags] VOLUME [VOLUME...]",
		Aliases:           []string{"remove"},
		Short:             "Remove one or more volumes",
		Long:              "NOTE: You cannot remove a volume that is in use by a container.",
		Args:              cobra.MinimumNArgs(1),
		RunE:              volumeRmAction,
		SilenceUsage:      true,
		SilenceErrors:     true,
		ValidArgsFunction: completion.VolumeNames,
	}

	volumeRmCommand.Flags().BoolP(flagForce, "f", false, "(unimplemented yet)")

	return volumeRmCommand
}

func processVolumeRmOptions(cmd *cobra.Command) (*types.VolumeRemoveOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	force, err := cmd.Flags().GetBool(flagForce)
	if err != nil {
		return nil, err
	}

	return &types.VolumeRemoveOptions{
		GOptions: globalOptions,
		Force:    force,
	}, nil
}

func volumeRmAction(cmd *cobra.Command, volumesToRemove []string) error {
	options, err := processVolumeRmOptions(cmd)
	if err != nil {
		return err
	}

	client, ctx, cancel, err := clientutil.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	removed, cannotRemove, errList, err := volume.Remove(ctx, client, volumesToRemove, options)

	// Output on stdout whatever was successful
	for _, name := range removed {
		_, err = fmt.Fprintln(cmd.OutOrStdout(), name)
		if err != nil {
			return err
		}
	}

	// If we have a hard error, report it here
	if err != nil {
		log.G(ctx).Error(errors.New(strings.Replace(err.Error(), "\n", ": ", -1)))
	}

	// Log the rest as warnings
	if len(cannotRemove) > 0 {
		for _, errDetail := range errList {
			log.G(ctx).Warn(errors.New(strings.Replace(errDetail.Error(), "\n", ": ", -1)))
		}

		if len(removed) == 0 {
			err = fmt.Errorf("none of the following volumes could be removed: %s", strings.Join(cannotRemove, ", "))
		} else {
			err = fmt.Errorf("the following volumes could not be removed: %s", strings.Join(cannotRemove, ", "))
		}

		return err
	}

	return nil
}
