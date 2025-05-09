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

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/composer"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "compose [flags] COMMAND",
		Short:            "Compose",
		RunE:             helpers.UnknownSubcommandAction,
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true, // required for global short hands like -f
	}

	// `-f` is a nonPersistentAlias, as it conflicts with `compose logs --follow`
	helpers.AddPersistentStringArrayFlag(cmd, "file", nil, []string{"f"}, nil, "", "Specify an alternate compose file")
	cmd.PersistentFlags().String("project-directory", "", "Specify an alternate working directory")
	cmd.PersistentFlags().StringP("project-name", "p", "", "Specify an alternate project name")
	cmd.PersistentFlags().String("env-file", "", "Specify an alternate environment file")
	cmd.PersistentFlags().StringArray("profile", []string{}, "Specify a profile to enable")

	cmd.AddCommand(
		upCommand(),
		logsCommand(),
		configCommand(),
		copyCommand(),
		buildCommand(),
		execCommand(),
		imagesCommand(),
		portCommand(),
		pushCommand(),
		pullCommand(),
		downCommand(),
		psCommand(),
		killCommand(),
		restartCommand(),
		removeCommand(),
		runCommand(),
		versionCommand(),
		startCommand(),
		stopCommand(),
		pauseCommand(),
		unpauseCommand(),
		topCommand(),
		createCommand(),
	)

	return cmd
}

func getComposeOptions(cmd *cobra.Command, debugFull, experimental bool) (*composer.Options, error) {
	cliCmd, cliArgs := helpers.GlobalFlags(cmd)

	projectDirectory, err := cmd.Flags().GetString("project-directory")
	if err != nil {
		return nil, err
	}

	envFile, err := cmd.Flags().GetString("env-file")
	if err != nil {
		return nil, err
	}

	projectName, err := cmd.Flags().GetString("project-name")
	if err != nil {
		return nil, err
	}

	files, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		return nil, err
	}

	profiles, err := cmd.Flags().GetStringArray("profile")
	if err != nil {
		return nil, err
	}

	return &composer.Options{
		Project:          projectName,
		ProjectDirectory: projectDirectory,
		ConfigPaths:      files,
		Profiles:         profiles,
		EnvFile:          envFile,
		CliCmd:           cliCmd,
		CliArgs:          cliArgs,
		DebugPrintFull:   debugFull,
		Experimental:     experimental,
	}, nil
}
