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
	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/pkg/cmd/logout"
)

func LogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "logout [flags] [SERVER]",
		Args:              cobra.MaximumNArgs(1),
		Short:             "Log out from a container registry",
		RunE:              logoutAction,
		ValidArgsFunction: logoutShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
}

func logoutAction(cmd *cobra.Command, args []string) error {
	logoutServer := ""
	if len(args) > 0 {
		logoutServer = args[0]
	}

	errGroup, err := logout.Logout(cmd.Context(), logoutServer)
	if err != nil {
		log.L.WithError(err).Errorf("Failed to erase credentials for: %s", logoutServer)
	}
	if errGroup != nil {
		log.L.Error("None of the following entries could be found")
		for _, v := range errGroup {
			log.L.Errorf("%s", v)
		}
	}

	return err
}

func logoutShellComplete(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	candidates, err := logout.ShellCompletion()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}
