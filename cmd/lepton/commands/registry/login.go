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

package registry

import (
	"errors"
	"io"
	"strings"

	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/login"
)

func LoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "login [flags] [SERVER]",
		Args:          cobra.MaximumNArgs(1),
		Short:         "Log in to a container registry",
		RunE:          loginAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringP("username", "u", "", "Username")
	cmd.Flags().StringP("password", "p", "", "Password")
	cmd.Flags().Bool("password-stdin", false, "Take the password from stdin")

	return cmd
}

func loginOptions(cmd *cobra.Command, args []string) (*options.LoginCommand, error) {
	username, err := cmd.Flags().GetString("username")
	if err != nil {
		return nil, err
	}

	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return nil, err
	}

	passwordStdin, err := cmd.Flags().GetBool("password-stdin")
	if err != nil {
		return nil, err
	}

	if strings.Contains(username, ":") {
		return nil, errors.New("username cannot contain colons")
	}

	if password != "" {
		log.L.Warn("WARNING! Using --password via the CLI is insecure. Use --password-stdin.")
		if passwordStdin {
			return nil, errors.New("--password and --password-stdin are mutually exclusive")
		}
	}

	if passwordStdin {
		if username == "" {
			return nil, errors.New("must provide --username with --password-stdin")
		}

		contents, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, err
		}

		password = strings.TrimSuffix(string(contents), "\n")
		password = strings.TrimSuffix(password, "\r")
	}

	serverAddress := ""
	if len(args) == 1 {
		serverAddress = args[0]
	}

	return &options.LoginCommand{
		Username:      username,
		Password:      password,
		ServerAddress: serverAddress,
	}, nil
}

func loginAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := loginOptions(cmd, args)
	if err != nil {
		return err
	}

	return login.Login(cmd.Context(), cmd.OutOrStdout(), globalOptions, opts)
}
