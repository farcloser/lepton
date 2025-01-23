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
	"github.com/spf13/cobra"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/pkg/composer"
)

func NewComposeCommand() *cobra.Command {
	var composeCommand = &cobra.Command{
		Use:              "compose [flags] COMMAND",
		Short:            "Compose",
		RunE:             helpers.UnknownSubcommandAction,
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true, // required for global short hands like -f
	}
	// `-f` is a nonPersistentAlias, as it conflicts with `compose logs --follow`
	helpers.AddPersistentStringArrayFlag(composeCommand, "file", nil, []string{"f"}, nil, "", "Specify an alternate compose file")
	composeCommand.PersistentFlags().String("project-directory", "", "Specify an alternate working directory")
	composeCommand.PersistentFlags().StringP("project-name", "p", "", "Specify an alternate project name")
	composeCommand.PersistentFlags().String("env-file", "", "Specify an alternate environment file")
	composeCommand.PersistentFlags().StringArray("profile", []string{}, "Specify a profile to enable")

	composeCommand.AddCommand(
		newComposeUpCommand(),
		newComposeLogsCommand(),
		newComposeConfigCommand(),
		newComposeCopyCommand(),
		newComposeBuildCommand(),
		newComposeExecCommand(),
		newComposeImagesCommand(),
		newComposePortCommand(),
		newComposePushCommand(),
		newComposePullCommand(),
		newComposeDownCommand(),
		newComposePsCommand(),
		newComposeKillCommand(),
		newComposeRestartCommand(),
		newComposeRemoveCommand(),
		newComposeRunCommand(),
		newComposeVersionCommand(),
		newComposeStartCommand(),
		newComposeStopCommand(),
		newComposePauseCommand(),
		newComposeUnpauseCommand(),
		newComposeTopCommand(),
		newComposeCreateCommand(),
	)

	return composeCommand
}

func getComposeOptions(cmd *cobra.Command, debugFull, experimental bool) (composer.Options, error) {
	nerdctlCmd, nerdctlArgs := helpers.GlobalFlags(cmd)
	projectDirectory, err := cmd.Flags().GetString("project-directory")
	if err != nil {
		return composer.Options{}, err
	}
	envFile, err := cmd.Flags().GetString("env-file")
	if err != nil {
		return composer.Options{}, err
	}
	projectName, err := cmd.Flags().GetString("project-name")
	if err != nil {
		return composer.Options{}, err
	}
	files, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		return composer.Options{}, err
	}
	profiles, err := cmd.Flags().GetStringArray("profile")
	if err != nil {
		return composer.Options{}, err
	}

	return composer.Options{
		Project:          projectName,
		ProjectDirectory: projectDirectory,
		ConfigPaths:      files,
		Profiles:         profiles,
		EnvFile:          envFile,
		CliCmd:           nerdctlCmd,
		CliArgs:          nerdctlArgs,
		DebugPrintFull:   debugFull,
		Experimental:     experimental,
	}, nil
}
