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

package image

import (
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/commands/builder"
	"go.farcloser.world/lepton/cmd/lepton/helpers"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Annotations:   map[string]string{helpers.Category: helpers.Management},
		Use:           "image",
		Short:         "Manage images",
		RunE:          helpers.UnknownSubcommandAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		builder.BuildCommand(),
		listCommand(),
		HistoryCommand(),
		PullCommand(),
		PushCommand(),
		LoadCommand(),
		SaveCommand(),
		TagCommand(),
		removeCommand(),
		convertCommand(),
		inspectCommand(),
		encryptCommand(),
		decryptCommand(),
		pruneCommand(),
	)

	return cmd
}

func listCommand() *cobra.Command {
	cmd := ListCommand()
	cmd.Use = "list"
	cmd.Aliases = []string{"ls"}

	return cmd
}

func removeCommand() *cobra.Command {
	cmd := RemoveCommand()
	cmd.Use = "remove"
	cmd.Aliases = []string{"rm"}

	return cmd
}
