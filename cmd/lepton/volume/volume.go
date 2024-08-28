package volume

import (
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
)

const (
	flagLabel  = "label"
	flagAll    = "all"
	flagForce  = "force"
	flagFilter = "filter"
	flagFormat = "format"
	flagQuiet  = "quiet"
	flagSize   = "size"
)

func NewVolumeCommand() *cobra.Command {
	volumeCommand := &cobra.Command{
		Annotations:   map[string]string{helpers.Category: helpers.Management},
		Use:           "volume",
		Short:         "Manage volumes",
		RunE:          helpers.UnknownSubcommandAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	volumeCommand.AddCommand(
		NewVolumeCreateCommand(),
		NewVolumeInspectCommand(),
		NewVolumeLsCommand(),
		NewVolumePruneCommand(),
		NewVolumeRmCommand(),
	)

	return volumeCommand
}
