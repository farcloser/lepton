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
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
)

func Command() *cobra.Command {
	containerCommand := &cobra.Command{
		Annotations:   map[string]string{helpers.Category: helpers.Management},
		Use:           "container",
		Short:         "Manage containers",
		RunE:          helpers.UnknownSubcommandAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	containerCommand.AddCommand(
		CreateCommand(),
		RunCommand(),
		UpdateCommand(),
		ExecCommand(),
		listCommand(),
		inspectCommand(),
		LogsCommand(),
		PortCommand(),
		RmCommand(),
		StopCommand(),
		StartCommand(),
		RestartCommand(),
		KillCommand(),
		PauseCommand(),
		DiffCommand(),
		WaitCommand(),
		UnpauseCommand(),
		CommitCommand(),
		RenameCommand(),
		pruneCommand(),
		StatsCommand(),
		AttachCommand(),
	)
	AddCpCommand(containerCommand)
	return containerCommand
}

func listCommand() *cobra.Command {
	x := PsCommand()
	x.Use = "ls"
	x.Aliases = []string{"list"}
	return x
}
