/*
   Copyright The containerd Authors.

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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/composer"
)

// AddPersistentStringArrayFlag is similar to cmd.Flags().StringArray but supports aliases and env var and persistent.
// See https://github.com/spf13/cobra/blob/main/user_guide.md#persistent-flags to learn what is "persistent".
func AddPersistentStringArrayFlag(cmd *cobra.Command, name string, aliases, nonPersistentAliases []string, value []string, env string, usage string) {
	if env != "" {
		usage = fmt.Sprintf("%s [$%s]", usage, env)
	}
	if envV, ok := os.LookupEnv(env); ok {
		value = []string{envV}
	}
	aliasesUsage := fmt.Sprintf("Alias of --%s", name)
	p := new([]string)
	flags := cmd.Flags()
	for _, a := range nonPersistentAliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			flags.StringArrayVarP(p, a, a, value, aliasesUsage)
		} else {
			flags.StringArrayVar(p, a, value, aliasesUsage)
		}
	}

	persistentFlags := cmd.PersistentFlags()
	persistentFlags.StringArrayVar(p, name, value, usage)
	for _, a := range aliases {
		if len(a) == 1 {
			// pflag doesn't support short-only flags, so we have to register long one as well here
			persistentFlags.StringArrayVarP(p, a, a, value, aliasesUsage)
		} else {
			persistentFlags.StringArrayVar(p, a, value, aliasesUsage)
		}
	}
}

func NewComposeCommand() *cobra.Command {
	var composeCommand = &cobra.Command{
		Use:              "compose [flags] COMMAND",
		Short:            "Compose",
		RunE:             helpers.UnknownSubcommandAction,
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true, // required for global short hands like -f
	}
	// `-f` is a nonPersistentAlias, as it conflicts with `nerdctl compose logs --follow`
	AddPersistentStringArrayFlag(composeCommand, "file", nil, []string{"f"}, nil, "", "Specify an alternate compose file")
	composeCommand.PersistentFlags().String("project-directory", "", "Specify an alternate working directory")
	composeCommand.PersistentFlags().StringP("project-name", "p", "", "Specify an alternate project name")
	composeCommand.PersistentFlags().String("env-file", "", "Specify an alternate environment file")
	composeCommand.PersistentFlags().StringArray("profile", []string{}, "Specify a profile to enable")

	composeCommand.AddCommand(
		NewComposeUpCommand(),
		NewComposeLogsCommand(),
		NewComposeConfigCommand(),
		NewComposeCopyCommand(),
		NewComposeBuildCommand(),
		NewComposeExecCommand(),
		NewComposeImagesCommand(),
		NewComposePortCommand(),
		NewComposePushCommand(),
		NewComposePullCommand(),
		NewComposeDownCommand(),
		NewComposePsCommand(),
		NewComposeKillCommand(),
		NewComposeRestartCommand(),
		NewComposeRemoveCommand(),
		NewComposeRunCommand(),
		NewComposeVersionCommand(),
		NewComposeStartCommand(),
		NewComposeStopCommand(),
		NewComposePauseCommand(),
		NewComposeUnpauseCommand(),
		NewComposeTopCommand(),
		NewComposeCreateCommand(),
	)

	return composeCommand
}

func getComposeOptions(cmd *cobra.Command, debug, experimental bool) (composer.Options, error) {
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
		NerdctlCmd:       nerdctlCmd,
		NerdctlArgs:      nerdctlArgs,
		DebugPrintFull:   debug,
		Experimental:     experimental,
	}, nil
}
