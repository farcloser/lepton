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

package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/containerd/log"
	"github.com/fatih/color"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go.farcloser.world/containers/security/cgroups"
	"go.farcloser.world/core/filesystem"

	"go.farcloser.world/lepton/cmd/lepton/commands/builder"
	image2 "go.farcloser.world/lepton/cmd/lepton/commands/image"
	"go.farcloser.world/lepton/cmd/lepton/commands/namespace"
	"go.farcloser.world/lepton/cmd/lepton/commands/network"
	"go.farcloser.world/lepton/cmd/lepton/commands/registry"
	"go.farcloser.world/lepton/cmd/lepton/commands/system"
	"go.farcloser.world/lepton/cmd/lepton/commands/volume"
	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/compose"
	"go.farcloser.world/lepton/cmd/lepton/container"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/cmd/lepton/inspect"
	"go.farcloser.world/lepton/cmd/lepton/internal"
	"go.farcloser.world/lepton/pkg/config"
	ncdefaults "go.farcloser.world/lepton/pkg/defaults"
	"go.farcloser.world/lepton/pkg/errutil"
	"go.farcloser.world/lepton/pkg/logging"
	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/version"
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
	s += printCommands("helpers.Management commands", managementCommands)
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
	logging.InitLogging()
	logging.InitLogViewer()

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
	// CLI mode
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
			return nil, fmt.Errorf("failed to load config (not daemon config) from %q (Hint: don't mix up daemon's `config.toml` with `%s.toml`): %w", tomlPath, version.RootName, err)
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
	rootCmd.PersistentFlags().Bool("debug-full", cfg.DebugFull, "debug mode (with full output)")
	// -a is aliases (conflicts with images -a)
	helpers.AddPersistentStringFlag(rootCmd, "address", []string{"a", "H"}, nil, []string{"host"}, aliasToBeInherited, cfg.Address, "CONTAINERD_ADDRESS", `containerd address, optionally with "unix://" prefix`)
	// -n is aliases (conflicts with logs -n)
	helpers.AddPersistentStringFlag(rootCmd, "namespace", []string{"n"}, nil, nil, aliasToBeInherited, cfg.Namespace, "CONTAINERD_NAMESPACE", `containerd namespace, such as "moby" for Docker, "k8s.io" for Kubernetes`)
	rootCmd.RegisterFlagCompletionFunc("namespace", completion.NamespaceNames)
	helpers.AddPersistentStringFlag(rootCmd, "snapshotter", nil, nil, []string{"storage-driver"}, aliasToBeInherited, cfg.Snapshotter, "CONTAINERD_SNAPSHOTTER", "containerd snapshotter")
	rootCmd.RegisterFlagCompletionFunc("snapshotter", completion.SnapshotterNames)
	rootCmd.RegisterFlagCompletionFunc("storage-driver", completion.SnapshotterNames)
	helpers.AddPersistentStringFlag(rootCmd, "cni-path", nil, nil, nil, aliasToBeInherited, cfg.CNIPath, "CNI_PATH", "cni plugins binary directory")
	helpers.AddPersistentStringFlag(rootCmd, "cni-netconfpath", nil, nil, nil, aliasToBeInherited, cfg.CNINetConfPath, "NETCONFPATH", "cni config directory")
	rootCmd.PersistentFlags().String("data-root", cfg.DataRoot, "Root directory of persistent state (managed by the cli, not by containerd)")
	rootCmd.PersistentFlags().String("cgroup-manager", string(cfg.CgroupManager), `Cgroup manager to use ("systemd")`)
	rootCmd.RegisterFlagCompletionFunc("cgroup-manager", completion.CgroupManagerNames)
	rootCmd.PersistentFlags().Bool("insecure-registry", cfg.InsecureRegistry, "skips verifying HTTPS certs, and allows falling back to plain HTTP")
	// hosts-dir is defined as StringSlice, not StringArray, to allow specifying "--hosts-dir=/etc/containerd/certs.d,/etc/docker/certs.d"
	rootCmd.PersistentFlags().StringSlice("hosts-dir", cfg.HostsDir, "A directory that contains <HOST:PORT>/hosts.toml (containerd style) or <HOST:PORT>/{ca.cert, cert.pem, key.pem} (docker style)")
	// Experimental enable experimental feature, see in https://github.com/farcloser/lepton/blob/main/docs/experimental.md
	helpers.AddPersistentBoolFlag(rootCmd, "experimental", nil, nil, cfg.Experimental, version.EnvPrefix+"_EXPERIMENTAL", "Control experimental: https://github.com/farcloser/lepton/blob/main/docs/experimental.md")
	helpers.AddPersistentStringFlag(rootCmd, "host-gateway-ip", nil, nil, nil, aliasToBeInherited, cfg.HostGatewayIP, version.EnvPrefix+"_HOST_GATEWAY_IP", "IP address that the special 'host-gateway' string in --add-host resolves to. Defaults to the IP address of the host. It has no effect without setting --add-host")
	helpers.AddPersistentStringFlag(rootCmd, "bridge-ip", nil, nil, nil, aliasToBeInherited, cfg.BridgeIP, version.EnvPrefix+"_BRIDGE_IP", "IP address for the default bridge network")
	rootCmd.PersistentFlags().Bool("kube-hide-dupe", cfg.KubeHideDupe, "Deduplicate images for Kubernetes with namespace k8s.io")
	return aliasToBeInherited, nil
}

func newApp() (*cobra.Command, error) {

	tomlPath := ncdefaults.CliTOML()
	if v, ok := os.LookupEnv(version.EnvPrefix + "_TOML"); ok {
		tomlPath = v
	}

	short := version.RootName + " is a command line interface for containerd"
	long := fmt.Sprintf(`%s

Config file ($%s_TOML): %s
`, short, version.EnvPrefix, tomlPath)
	var rootCmd = &cobra.Command{
		Use:              version.RootName,
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
		debug := globalOptions.DebugFull
		if !debug {
			debug = globalOptions.Debug
		}
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
			case cgroups.SystemdManager, cgroups.NoneManager:
			default:
				return fmt.Errorf("invalid cgroup-manager %q (supported values: \"systemd\", \"none\")", cgroupManager)
			}
		}

		// Since we store containers' stateful information on the filesystem per namespace, we need namespaces to be
		// valid, safe path segments. This is enforced by store.ValidatePathComponent.
		// Note that the container runtime will further enforce additional restrictions on namespace names
		// (containerd treats namespaces as valid identifiers - eg: alphanumericals + dash, starting with a letter)
		// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#path-segment-names for
		// considerations about path segments identifiers.
		if err = filesystem.ValidatePathComponent(globalOptions.Namespace); err != nil {
			return err
		}
		if appNeedsRootlessParentMain(cmd, args) {
			// reexec /proc/self/exe with `nsenter` into RootlessKit namespaces
			return rootlessutil.ParentMain(globalOptions.HostGatewayIP)
		}
		return nil
	}
	rootCmd.RunE = helpers.UnknownSubcommandAction
	rootCmd.AddCommand(
		container.CreateCommand(),
		// #region Run & Exec
		container.RunCommand(),
		container.UpdateCommand(),
		container.ExecCommand(),
		// #endregion

		// #region Container management
		container.PsCommand(),
		container.LogsCommand(),
		container.PortCommand(),
		container.StopCommand(),
		container.StartCommand(),
		container.DiffCommand(),
		container.RestartCommand(),
		container.KillCommand(),
		container.RemoveCommand(),
		container.PauseCommand(),
		container.UnpauseCommand(),
		container.CommitCommand(),
		container.WaitCommand(),
		container.RenameCommand(),
		container.AttachCommand(),
		// #endregion

		// Build
		builder.BuildCommand(),

		// #region Image management
		image2.ListCommand(),
		image2.PullCommand(),
		image2.PushCommand(),
		image2.LoadCommand(),
		image2.SaveCommand(),
		image2.TagCommand(),
		image2.RemoveCommand(),
		image2.HistoryCommand(),
		// #endregion

		// #region System
		system.EventsCommand(),
		system.InfoCommand(),
		newVersionCommand(),
		// #endregion

		// Inspect
		inspect.Command(),

		// stats
		container.TopCommand(),
		container.StatsCommand(),

		// #region helpers.Management
		container.Command(),
		image2.Command(),
		network.Command(),
		registry.Command(),
		volume.Command(),
		system.Command(),
		namespace.Command(),
		builder.Command(),
		// #endregion

		// Internal
		internal.Command(),

		// login
		registry.LoginCommand(),

		// Logout
		registry.LogoutCommand(),

		// Compose
		compose.Command(),
	)
	addApparmorCommand(rootCmd)
	container.AddCopyCommand(rootCmd)

	// add aliasToBeInherited to subCommand(s) InheritedFlags
	for _, subCmd := range rootCmd.Commands() {
		subCmd.InheritedFlags().AddFlagSet(aliasToBeInherited)
	}
	return rootCmd, nil
}
