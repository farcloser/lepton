package volume

import (
	"errors"

	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/completion"
	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/cmd/volume"
	"github.com/farcloser/lepton/pkg/formatter"
)

func NewVolumeInspectCommand() *cobra.Command {
	volumeInspectCommand := &cobra.Command{
		Use:               "inspect [flags] VOLUME [VOLUME...]",
		Short:             "Display detailed information on one or more volumes",
		Args:              cobra.MinimumNArgs(1),
		RunE:              volumeInspectAction,
		SilenceUsage:      true,
		SilenceErrors:     true,
		ValidArgsFunction: completion.VolumeNames,
	}

	volumeInspectCommand.Flags().StringP(flagFormat, "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")
	// FIXME: @apostasie remove this
	volumeInspectCommand.Flags().BoolP(flagSize, "s", false, "Display the disk usage of the volume")

	volumeInspectCommand.RegisterFlagCompletionFunc(flagFormat, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json"}, cobra.ShellCompDirectiveNoFileComp
	})

	return volumeInspectCommand
}

func processVolumeInspectOptions(cmd *cobra.Command) (*types.VolumeInspectOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, err
	}

	volumeSize, err := cmd.Flags().GetBool(flagSize)
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return nil, err
	}

	return &types.VolumeInspectOptions{
		GOptions: globalOptions,
		Format:   format,
		Size:     volumeSize,
	}, nil
}

func volumeInspectAction(cmd *cobra.Command, args []string) error {
	options, err := processVolumeInspectOptions(cmd)
	if err != nil {
		return err
	}

	result, warns, err := volume.Inspect(cmd.Context(), args, options)
	if err != nil {
		return err
	}

	err = formatter.FormatSlice(options.Format, cmd.OutOrStdout(), result)
	if err != nil {
		return err
	}
	for _, warn := range warns {
		log.G(cmd.Context()).Warn(warn)
	}

	if len(warns) != 0 {
		return errors.New("some volumes could not be inspected")
	}

	return nil
}
