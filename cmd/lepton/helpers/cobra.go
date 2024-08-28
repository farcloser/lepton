package helpers

import (
	"errors"
	"fmt"
	"os"

	"github.com/containerd/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/farcloser/lepton/pkg/api/types"
)

func ProcessRootCmdFlags(cmd *cobra.Command) (types.GlobalCommandOptions, error) {
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	address, err := cmd.Flags().GetString("address")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	// FIXME: @apostasie review below here
	snapshotter, err := cmd.Flags().GetString("snapshotter")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	cniPath, err := cmd.Flags().GetString("cni-path")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	cniConfigPath, err := cmd.Flags().GetString("cni-netconfpath")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	dataRoot, err := cmd.Flags().GetString("data-root")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	cgroupManager, err := cmd.Flags().GetString("cgroup-manager")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	hostsDir, err := cmd.Flags().GetStringSlice("hosts-dir")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	experimental, err := cmd.Flags().GetBool("experimental")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	hostGatewayIP, err := cmd.Flags().GetString("host-gateway-ip")
	if err != nil {
		return types.GlobalCommandOptions{}, err
	}

	return types.GlobalCommandOptions{
		Debug:          debug,
		Address:        address,
		Namespace:      namespace,
		Snapshotter:    snapshotter,
		CNIPath:        cniPath,
		CNINetConfPath: cniConfigPath,
		DataRoot:       dataRoot,
		CgroupManager:  cgroupManager,
		HostsDir:       hostsDir,
		Experimental:   experimental,
		HostGatewayIP:  hostGatewayIP,
	}, nil
}

// helpers.UnknownSubcommandAction is needed to let `nerdctl system non-existent-command` fail
// https://github.com/containerd/nerdctl/issues/487
//
// Ideally this should be implemented in Cobra itself.
func UnknownSubcommandAction(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	// The output mimics https://github.com/spf13/cobra/blob/v1.2.1/command.go#L647-L662
	msg := fmt.Sprintf("unknown subcommand %q for %q", args[0], cmd.Name())
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		msg += "\n\nDid you mean this?\n"
		for _, s := range suggestions {
			msg += fmt.Sprintf("\t%v\n", s)
		}
	}

	return errors.New(msg)
}

// helpers.IsExactArgs returns an error if there is not the exact number of args
func IsExactArgs(number int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == number {
			return nil
		}
		return fmt.Errorf(
			"%q requires exactly %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			number,
			"argument(s)",
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

func GlobalFlags(cmd *cobra.Command) (string, []string) {
	args0, err := os.Executable()
	if err != nil {
		log.L.WithError(err).Warnf("cannot call os.Executable(), assuming the executable to be %q", os.Args[0])
		args0 = os.Args[0]
	}
	if len(os.Args) < 2 {
		return args0, nil
	}

	rootCmd := cmd.Root()
	flagSet := rootCmd.Flags()
	args := []string{}
	flagSet.VisitAll(func(f *pflag.Flag) {
		key := f.Name
		val := f.Value.String()
		if f.Changed {
			args = append(args, "--"+key+"="+val)
		}
	})
	return args0, args
}
