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

package compose

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"go.farcloser.world/core/term"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/cmd/compose"
	"go.farcloser.world/lepton/pkg/composer"
)

func execCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "exec [flags] SERVICE COMMAND [ARGS...]",
		Short:         "Execute a command in a running container of the service",
		Args:          cobra.MinimumNArgs(2),
		RunE:          execAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().BoolP("no-TTY", "T", !term.IsTerminal(os.Stdout.Fd()), "Disable pseudo-TTY allocation. By default compose exec allocates a TTY.")
	cmd.Flags().BoolP("detach", "d", false, "Detached mode: Run containers in the background")
	cmd.Flags().StringP("workdir", "w", "", "Working directory inside the container")
	// env needs to be StringArray, not StringSlice, to prevent "FOO=foo1,foo2" from being split to {"FOO=foo1", "foo2"}
	cmd.Flags().StringArrayP("env", "e", nil, "Set environment variables")
	cmd.Flags().Bool("privileged", false, "Give extended privileges to the command")
	cmd.Flags().StringP("user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	cmd.Flags().Int("index", 1, "index of the container if the service has multiple instances.")
	cmd.Flags().BoolP("interactive", "i", true, "Keep STDIN open even if not attached")
	// The -t does not have effect to keep the compatibility with docker.
	// The proposal of -t is to keep "muscle memory" with compose v1: https://github.com/docker/compose/issues/9207
	// FYI: https://github.com/docker/compose/blob/v2.23.1/cmd/compose/exec.go#L77
	cmd.Flags().BoolP("tty", "t", true, "Allocate a pseudo-TTY")

	_ = cmd.Flags().MarkHidden("interactive")
	_ = cmd.Flags().MarkHidden("tty")

	return cmd
}

func execAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	interactive, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		return err
	}
	noTty, err := cmd.Flags().GetBool("no-TTY")
	if err != nil {
		return err
	}
	detach, err := cmd.Flags().GetBool("detach")
	if err != nil {
		return err
	}
	workdir, err := cmd.Flags().GetString("workdir")
	if err != nil {
		return err
	}
	env, err := cmd.Flags().GetStringArray("env")
	if err != nil {
		return err
	}
	privileged, err := cmd.Flags().GetBool("privileged")
	if err != nil {
		return err
	}
	user, err := cmd.Flags().GetString("user")
	if err != nil {
		return err
	}
	index, err := cmd.Flags().GetInt("index")
	if err != nil {
		return err
	}

	if index < 1 {
		return errors.New("index starts from 1 and should be equal or greater than 1")
	}
	if interactive && detach {
		return errors.New("currently flag -i and -d cannot be specified together (FIXME)")
	}
	if !noTty && detach {
		return errors.New("currently flag -d should be specified with --no-TTY (FIXME)")
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()
	options, err := getComposeOptions(cmd, globalOptions.DebugFull, globalOptions.Experimental)
	if err != nil {
		return err
	}
	c, err := compose.New(cli, globalOptions, options, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	eo := composer.ExecOptions{
		ServiceName: args[0],
		Index:       index,

		Interactive: interactive,
		Tty:         !noTty,
		Detach:      detach,
		WorkDir:     workdir,
		Env:         env,
		Privileged:  privileged,
		User:        user,
		Args:        args[1:],
	}

	return c.Exec(ctx, eo)
}
