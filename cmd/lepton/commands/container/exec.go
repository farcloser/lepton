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

package container

import (
	"errors"

	"github.com/containerd/containerd/v2/client"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/completion"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/container"
)

func ExecCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:               "exec [flags] CONTAINER COMMAND [ARG...]",
		Args:              cobra.MinimumNArgs(2),
		Short:             "Run a command in a running container",
		RunE:              execAction,
		ValidArgsFunction: execShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().BoolP("tty", "t", false, "Allocate a pseudo-TTY")
	cmd.Flags().BoolP("interactive", "i", false, "Keep STDIN open even if not attached")
	cmd.Flags().BoolP("detach", "d", false, "Detached mode: run command in the background")
	cmd.Flags().StringP("workdir", "w", "", "Working directory inside the container")
	// env needs to be StringArray, not StringSlice, to prevent "FOO=foo1,foo2" from being split to {"FOO=foo1", "foo2"}
	cmd.Flags().StringArrayP("env", "e", nil, "Set environment variables")
	// env-file is defined as StringSlice, not StringArray, to allow specifying "--env-file=FILE1,FILE2" (compatible with Podman)
	cmd.Flags().StringSlice("env-file", nil, "Set environment variables from file")
	cmd.Flags().Bool("privileged", false, "Give extended privileges to the command")
	cmd.Flags().StringP("user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")

	return cmd
}

func execOptions(cmd *cobra.Command, _ []string) (options.ContainerExec, error) {
	// We do not check if we have a terminal here, as container.Exec calling console.Current will ensure that
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return options.ContainerExec{}, err
	}

	flagI, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		return options.ContainerExec{}, err
	}
	flagT, err := cmd.Flags().GetBool("tty")
	if err != nil {
		return options.ContainerExec{}, err
	}
	flagD, err := cmd.Flags().GetBool("detach")
	if err != nil {
		return options.ContainerExec{}, err
	}

	if flagI {
		if flagD {
			return options.ContainerExec{}, errors.New("currently flag -i and -d cannot be specified together (FIXME)")
		}
	}

	if flagT {
		if flagD {
			return options.ContainerExec{}, errors.New("currently flag -t and -d cannot be specified together (FIXME)")
		}
	}

	workdir, err := cmd.Flags().GetString("workdir")
	if err != nil {
		return options.ContainerExec{}, err
	}

	envFile, err := cmd.Flags().GetStringSlice("env-file")
	if err != nil {
		return options.ContainerExec{}, err
	}
	env, err := cmd.Flags().GetStringArray("env")
	if err != nil {
		return options.ContainerExec{}, err
	}
	privileged, err := cmd.Flags().GetBool("privileged")
	if err != nil {
		return options.ContainerExec{}, err
	}
	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return options.ContainerExec{}, err
	}

	return options.ContainerExec{
		GOptions:    globalOptions,
		TTY:         flagT,
		Interactive: flagI,
		Detach:      flagD,
		Workdir:     workdir,
		Env:         env,
		EnvFile:     envFile,
		Privileged:  privileged,
		User:        user,
	}, nil
}

func execAction(cmd *cobra.Command, args []string) error {
	opts, err := execOptions(cmd, args)
	if err != nil {
		return err
	}
	// simulate the behavior of double dash
	newArg := []string{}
	if len(args) >= 2 && args[1] == "--" {
		newArg = append(newArg, args[:1]...)
		newArg = append(newArg, args[2:]...)
		args = newArg
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()

	return container.Exec(ctx, cli, args, opts)
}

func execShellComplete(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// show running container names
		statusFilterFn := func(st client.ProcessStatus) bool {
			return st == client.Running
		}
		return completion.ContainerNames(cmd, statusFilterFn)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
