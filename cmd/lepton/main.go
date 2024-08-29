/*
   Copyright The containerd Authors.

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

package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/containerd/log"
	"github.com/fatih/color"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/farcloser/lepton/cmd/lepton/apparmor"
	"github.com/farcloser/lepton/cmd/lepton/builder"
	"github.com/farcloser/lepton/cmd/lepton/completion"
	"github.com/farcloser/lepton/cmd/lepton/compose"
	"github.com/farcloser/lepton/cmd/lepton/container"
	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/cmd/lepton/image"
	"github.com/farcloser/lepton/cmd/lepton/inspect"
	"github.com/farcloser/lepton/cmd/lepton/internal"
	"github.com/farcloser/lepton/cmd/lepton/login"
	"github.com/farcloser/lepton/cmd/lepton/namespace"
	"github.com/farcloser/lepton/cmd/lepton/network"
	"github.com/farcloser/lepton/cmd/lepton/system"
	version2 "github.com/farcloser/lepton/cmd/lepton/version"
	"github.com/farcloser/lepton/cmd/lepton/volume"
	"github.com/farcloser/lepton/pkg/config"
	ncdefaults "github.com/farcloser/lepton/pkg/defaults"
	"github.com/farcloser/lepton/pkg/errutil"
	"github.com/farcloser/lepton/pkg/logging"
	"github.com/farcloser/lepton/pkg/rootlessutil"
	"github.com/farcloser/lepton/pkg/version"
)

var (
	// To print Bold Text
	Bold = color.New(color.Bold).SprintfFunc()
)

// usage was derived from https://github.com/spf13/cobra/blob/v1.2.1/command.go#L491-L514
func usage(c *cobra.Command) error {
	s := "Usage: "
	if c.Runnable() {
		s += c.UseLine() + "\n"
	} else {
		s += c.CommandPath() + " [command]\n"
	}
	s += "\n"
	if len(c.Aliases) > 0 {
		s += "Aliases: " + c.NameAndAliases() + "\n"
	}
	if c.HasExample() {
		s += "Example:\n"
		s += c.Example + "\n"
	}

	var managementCommands, nonManagementCommands []*cobra.Command
	for _, f := range c.Commands() {
		f := f
		if f.Hidden {
			continue
		}
		if f.Annotations[helpers.Category] == helpers.Management {
			managementCommands = append(managementCommands, f)
		} else {
			nonManagementCommands = append(nonManagementCommands, f)
		}
	}
	printCommands := func(title string, commands []*cobra.Command) string {
		if len(commands) == 0 {
			return ""
		}
		var longest int
		for _, f := range commands {
			if l := len(f.Name()); l > longest {
				longest = l
			}
		}

		title = Bold(title)
		t := title + ":\n"
		for _, f := range commands {
			t += "  "
			t += f.Name()
			t += strings.Repeat(" ", longest-len(f.Name()))
			t += "  " + f.Short + "\n"
		}
		t += "\n"
		return t
	}
	s += printCommands("Management commands", managementCommands)
	s += printCommands("Commands", nonManagementCommands)

	s += Bold("Flags") + ":\n"
	s += c.LocalFlags().FlagUsages() + "\n"

	if c == c.Root() {
		s += "Run '" + c.CommandPath() + " COMMAND --help' for more information on a command.\n"
	} else {
		s += "See also '" + c.Root().CommandPath() + " --help' for the global flags such as '--namespace', '--snapshotter', and '--cgroup-manager'."
	}
	fmt.Fprintln(c.OutOrStdout(), s)
	return nil
}

func main() {
	if err := xmain(); err != nil {
		errutil.HandleExitCoder(err)
		log.L.Fatal(err)
	}
}

func xmain() error {
	if len(os.Args) == 3 && os.Args[1] == logging.MagicArgv1 {
		// containerd runtime v2 logging plugin mode.
		// "binary://BIN?KEY=VALUE" URI is parsed into Args {BIN, KEY, VALUE}.
		return logging.Main(os.Args[2])
	}
	// nerdctl CLI mode
	app, err := newApp()
	if err != nil {
		return err
	}
	return app.Execute()
}

func initRootCmdFlags(rootCmd *cobra.Command, tomlPath string) (*pflag.FlagSet, error) {
	cfg := config.New()
	if r, err := os.Open(tomlPath); err == nil {
		log.L.Debugf("Loading config from %q", tomlPath)
		defer r.Close()
		dec := toml.NewDecoder(r).DisallowUnknownFields() // set Strict to detect typo
		if err := dec.Decode(cfg); err != nil {
			return nil, fmt.Errorf("failed to load config (not daemon config) from %q (Hint: don't mix up daemon's `config.toml` with `lepton.toml`): %w", tomlPath, err)
		}
		log.L.Debugf("Loaded config %+v", cfg)
	} else {
		log.L.WithError(err).Debugf("Not loading config from %q", tomlPath)
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	aliasToBeInherited := pflag.NewFlagSet(rootCmd.Name(), pflag.ExitOnError)

	rootCmd.PersistentFlags().Bool("debug", cfg.Debug, "debug mode")
	// -a is aliases (conflicts with nerdctl images -a)
	AddPersistentStringFlag(rootCmd, "address", []string{"a", "H"}, nil, []string{"host"}, aliasToBeInherited, cfg.Address, "CONTAINERD_ADDRESS", `containerd address, optionally with "unix://" prefix`)
	// -n is aliases (conflicts with nerdctl logs -n)
	AddPersistentStringFlag(rootCmd, "namespace", []string{"n"}, nil, nil, aliasToBeInherited, cfg.Namespace, "CONTAINERD_NAMESPACE", `containerd namespace, such as "moby" for Docker, "k8s.io" for Kubernetes`)
	rootCmd.RegisterFlagCompletionFunc("namespace", completion.NamespaceNames)
	AddPersistentStringFlag(rootCmd, "snapshotter", nil, nil, []string{"storage-driver"}, aliasToBeInherited, cfg.Snapshotter, "CONTAINERD_SNAPSHOTTER", "containerd snapshotter")
	rootCmd.RegisterFlagCompletionFunc("snapshotter", completion.SnapshotterNames)
	rootCmd.RegisterFlagCompletionFunc("storage-driver", completion.SnapshotterNames)
	AddPersistentStringFlag(rootCmd, "cni-path", nil, nil, nil, aliasToBeInherited, cfg.CNIPath, "CNI_PATH", "cni plugins binary directory")
	AddPersistentStringFlag(rootCmd, "cni-netconfpath", nil, nil, nil, aliasToBeInherited, cfg.CNINetConfPath, "NETCONFPATH", "cni config directory")
	rootCmd.PersistentFlags().String("data-root", cfg.DataRoot, "Root directory of persistent nerdctl state (managed by nerdctl, not by containerd)")
	rootCmd.PersistentFlags().String("cgroup-manager", cfg.CgroupManager, `Cgroup manager to use ("cgroupfs"|"systemd")`)
	rootCmd.RegisterFlagCompletionFunc("cgroup-manager", completion.CgroupManagerNames)
	// hosts-dir is defined as StringSlice, not StringArray, to allow specifying "--hosts-dir=/etc/containerd/certs.d,/etc/docker/certs.d"
	rootCmd.PersistentFlags().StringSlice("hosts-dir", cfg.HostsDir, "A directory that contains <HOST:PORT>/hosts.toml (containerd style) or <HOST:PORT>/{ca.cert, cert.pem, key.pem} (docker style)")
	// Experimental enable experimental feature, see in https://github.com/containerd/nerdctl/blob/main/docs/experimental.md
	AddPersistentBoolFlag(rootCmd, "experimental", nil, nil, cfg.Experimental, "NERDCTL_EXPERIMENTAL", "Control experimental: https://github.com/containerd/nerdctl/blob/main/docs/experimental.md")
	AddPersistentStringFlag(rootCmd, "host-gateway-ip", nil, nil, nil, aliasToBeInherited, cfg.HostGatewayIP, "NERDCTL_HOST_GATEWAY_IP", "IP address that the special 'host-gateway' string in --add-host resolves to. Defaults to the IP address of the host. It has no effect without setting --add-host")
	return aliasToBeInherited, nil
}

func newApp() (*cobra.Command, error) {

	tomlPath := ncdefaults.NerdctlTOML()
	if v, ok := os.LookupEnv("NERDCTL_TOML"); ok {
		tomlPath = v
	}

	short := "nerdctl is a command line interface for containerd"
	long := fmt.Sprintf(`%s

Config file ($NERDCTL_TOML): %s
`, short, tomlPath)
	var rootCmd = &cobra.Command{
		Use:              "nerdctl",
		Short:            short,
		Long:             long,
		Version:          strings.TrimPrefix(version.GetVersion(), "v"),
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true, // required for global short hands like -a, -H, -n
	}

	rootCmd.SetUsageFunc(usage)
	aliasToBeInherited, err := initRootCmdFlags(rootCmd, tomlPath)
	if err != nil {
		return nil, err
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
		if err != nil {
			return err
		}
		debug := globalOptions.Debug
		if debug {
			log.SetLevel(log.DebugLevel.String())
		}
		address := globalOptions.Address
		if strings.Contains(address, "://") && !strings.HasPrefix(address, "unix://") {
			return fmt.Errorf("invalid address %q", address)
		}
		cgroupManager := globalOptions.CgroupManager
		if runtime.GOOS == "linux" {
			switch cgroupManager {
			case "systemd", "cgroupfs", "none":
			default:
				return fmt.Errorf("invalid cgroup-manager %q (supported values: \"systemd\", \"cgroupfs\", \"none\")", cgroupManager)
			}
		}
		if appNeedsRootlessParentMain(cmd, args) {
			// reexec /proc/self/exe with `nsenter` into RootlessKit namespaces
			return rootlessutil.ParentMain(globalOptions.HostGatewayIP)
		}
		return nil
	}
	rootCmd.RunE = helpers.UnknownSubcommandAction
	rootCmd.AddCommand(
		container.NewCreateCommand(),
		// #region Run & Exec
		container.NewRunCommand(),
		container.NewUpdateCommand(),
		container.NewExecCommand(),
		// #endregion

		// #region Container management
		container.NewPsCommand(),
		container.NewLogsCommand(),
		container.NewPortCommand(),
		container.NewStopCommand(),
		container.NewStartCommand(),
		container.NewDiffCommand(),
		container.NewRestartCommand(),
		container.NewKillCommand(),
		container.NewRmCommand(),
		container.NewPauseCommand(),
		container.NewUnpauseCommand(),
		container.NewCommitCommand(),
		container.NewWaitCommand(),
		container.NewRenameCommand(),
		container.NewAttachCommand(),
		// #endregion

		// Build
		builder.NewBuildCommand(),

		// #region Image management
		image.NewImagesCommand(),
		image.NewPullCommand(),
		image.NewPushCommand(),
		image.NewLoadCommand(),
		image.NewSaveCommand(),
		image.NewTagCommand(),
		image.NewRmiCommand(),
		image.NewHistoryCommand(),
		// #endregion

		// #region System
		system.NewEventsCommand(),
		system.NewInfoCommand(),
		version2.NewVersionCommand(),
		// #endregion

		// Inspect
		inspect.NewInspectCommand(),

		// stats
		container.NewTopCommand(),
		container.NewStatsCommand(),

		// #region Management
		container.NewContainerCommand(),
		image.NewImageCommand(),
		network.NewNetworkCommand(),
		volume.NewVolumeCommand(),
		system.NewSystemCommand(),
		namespace.NewNamespaceCommand(),
		builder.NewBuilderCommand(),
		// #endregion

		// Internal
		internal.NewInternalCommand(),

		// login
		login.NewLoginCommand(),

		// Logout
		login.NewLogoutCommand(),

		// Compose
		compose.NewComposeCommand(),
	)

	ac := apparmor.NewApparmorCommand()
	if ac != nil {
		rootCmd.AddCommand(ac)
	}

	container.AddCpCommand(rootCmd)

	// add aliasToBeInherited to subCommand(s) InheritedFlags
	for _, subCmd := range rootCmd.Commands() {
		subCmd.InheritedFlags().AddFlagSet(aliasToBeInherited)
	}
	return rootCmd, nil
}

// AddPersistentStringFlag is similar to AddStringFlag but persistent.
// See https://github.com/spf13/cobra/blob/main/user_guide.md#persistent-flags to learn what is "persistent".
func AddPersistentStringFlag(cmd *cobra.Command, name string, aliases, localAliases, persistentAliases []string, aliasToBeInherited *pflag.FlagSet, value string, env, usage string) {
	if env != "" {
		usage = fmt.Sprintf("%s [$%s]", usage, env)
	}
	if envV, ok := os.LookupEnv(env); ok {
		value = envV
	}
	aliasesUsage := fmt.Sprintf("Alias of --%s", name)
	p := new(string)

	// flags is full set of flag(s)
	// flags can redefine alias already used in subcommands
	flags := cmd.Flags()
	for _, a := range aliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			flags.StringVarP(p, a, a, value, aliasesUsage)
		} else {
			flags.StringVar(p, a, value, aliasesUsage)
		}
		// non-persistent flags are not added to the InheritedFlags, so we should add them manually
		f := flags.Lookup(a)
		aliasToBeInherited.AddFlag(f)
	}

	// localFlags are local to the rootCmd
	localFlags := cmd.LocalFlags()
	for _, a := range localAliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			localFlags.StringVarP(p, a, a, value, aliasesUsage)
		} else {
			localFlags.StringVar(p, a, value, aliasesUsage)
		}
	}

	// persistentFlags cannot redefine alias already used in subcommands
	persistentFlags := cmd.PersistentFlags()
	persistentFlags.StringVar(p, name, value, usage)
	for _, a := range persistentAliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			persistentFlags.StringVarP(p, a, a, value, aliasesUsage)
		} else {
			persistentFlags.StringVar(p, a, value, aliasesUsage)
		}
	}
}

// AddPersistentBoolFlag is similar to AddBoolFlag but persistent.
// See https://github.com/spf13/cobra/blob/main/user_guide.md#persistent-flags to learn what is "persistent".
func AddPersistentBoolFlag(cmd *cobra.Command, name string, aliases, nonPersistentAliases []string, value bool, env, usage string) {
	if env != "" {
		usage = fmt.Sprintf("%s [$%s]", usage, env)
	}
	if envV, ok := os.LookupEnv(env); ok {
		var err error
		value, err = strconv.ParseBool(envV)
		if err != nil {
			log.L.WithError(err).Warnf("Invalid boolean value for `%s`", env)
		}
	}
	aliasesUsage := fmt.Sprintf("Alias of --%s", name)
	p := new(bool)
	flags := cmd.Flags()
	for _, a := range nonPersistentAliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			flags.BoolVarP(p, a, a, value, aliasesUsage)
		} else {
			flags.BoolVar(p, a, value, aliasesUsage)
		}
	}

	persistentFlags := cmd.PersistentFlags()
	persistentFlags.BoolVar(p, name, value, usage)
	for _, a := range aliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			persistentFlags.BoolVarP(p, a, a, value, aliasesUsage)
		} else {
			persistentFlags.BoolVar(p, a, value, aliasesUsage)
		}
	}
}
