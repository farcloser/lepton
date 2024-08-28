package volume

import (
	"errors"
	"fmt"
	"strings"

	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/cmd/volume"
)

func NewVolumePruneCommand() *cobra.Command {
	volumePruneCommand := &cobra.Command{
		Use:           "prune [flags]",
		Short:         "Remove all unused local volumes",
		Args:          cobra.NoArgs,
		RunE:          volumePruneAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	volumePruneCommand.Flags().BoolP(flagAll, "a", false, "Remove all unused volumes, not just anonymous ones")
	volumePruneCommand.Flags().BoolP(flagForce, "f", false, "Do not prompt for confirmation")
	volumePruneCommand.Flags().StringSliceP(flagFilter, "", []string{}, "Filter matches volumes based on given conditions")

	return volumePruneCommand
}

func processVolumePruneOptions(cmd *cobra.Command) (*types.VolumePruneOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	all, err := cmd.Flags().GetBool(flagAll)
	if err != nil {
		return nil, err
	}

	force, err := cmd.Flags().GetBool(flagForce)
	if err != nil {
		return nil, err
	}

	filters, err := cmd.Flags().GetStringSlice(flagFilter)
	if err != nil {
		return nil, err
	}

	return &types.VolumePruneOptions{
		GOptions: globalOptions,
		All:      all,
		Force:    force,
		Filters:  filters,
	}, nil
}

func volumePruneAction(cmd *cobra.Command, _ []string) error {
	options, err := processVolumePruneOptions(cmd)
	if err != nil {
		return err
	}

	if !options.Force {
		msg := "This will remove all local volumes not used by at least one container."

		var confirmed bool
		if confirmed, err = helpers.Confirm(cmd, msg); err != nil || !confirmed {
			return err
		}
	}

	client, ctx, cancel, err := clientutil.NewClient(cmd.Context(), options.GOptions.Namespace, options.GOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	removed, cannotRemove, errList, err := volume.Prune(ctx, client, options)

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
	if len(errList) > 0 || err != nil {
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
