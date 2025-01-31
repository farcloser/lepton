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

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/commands/builder"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
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
		// commitCommand is in "container", not in "image"
		imageLsCommand(),
		HistoryCommand(),
		PullCommand(),
		PushCommand(),
		LoadCommand(),
		SaveCommand(),
		TagCommand(),
		imageRmCommand(),
		convertCommand(),
		inspectCommand(),
		encryptCommand(),
		decryptCommand(),
		pruneCommand(),
	)

	return cmd
}

func imageLsCommand() *cobra.Command {
	cmd := ImagesCommand()
	cmd.Use = "ls"
	cmd.Aliases = []string{"list"}

	return cmd
}

func imageRmCommand() *cobra.Command {
	cmd := RmiCommand()
	cmd.Use = "rm"
	cmd.Aliases = []string{"remove"}

	return cmd
}
