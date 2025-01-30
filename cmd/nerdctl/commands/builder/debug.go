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

package builder

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
)

func debugCommand() *cobra.Command {
	shortHelp := `Debug Dockerfile`
	var buildDebugCommand = &cobra.Command{
		Use:           "debug",
		Short:         shortHelp,
		PreRunE:       helpers.CheckExperimental("`builder debug`"),
		RunE:          debugAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	buildDebugCommand.Flags().StringP("file", "f", "", "Name of the Dockerfile")
	buildDebugCommand.Flags().String("target", "", "Set the target build stage to build")
	buildDebugCommand.Flags().StringArray("build-arg", nil, "Set build-time variables")
	buildDebugCommand.Flags().String("image", "", "Image to use for debugging stage")
	buildDebugCommand.Flags().StringArray("ssh", nil, "Allow forwarding SSH agent to the build. Format: default|<id>[=<socket>|<key>[,<key>]]")
	buildDebugCommand.Flags().StringArray("secret", nil, "Expose secret value to the build. Format: id=secretname,src=filepath")
	return buildDebugCommand
}

func debugAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("context needs to be specified")
	}

	buildgBinary, err := exec.LookPath("buildg")
	if err != nil {
		return err
	}
	buildgArgs := []string{"debug"}
	if globalOptions.Debug {
		buildgArgs = append([]string{"--debug"}, buildgArgs...)
	}

	if file, err := cmd.Flags().GetString("file"); err != nil {
		return err
	} else if file != "" {
		buildgArgs = append(buildgArgs, "--file="+file)
	}

	if target, err := cmd.Flags().GetString("target"); err != nil {
		return err
	} else if target != "" {
		buildgArgs = append(buildgArgs, "--target="+target)
	}

	if buildArgsValue, err := cmd.Flags().GetStringArray("build-arg"); err != nil {
		return err
	} else if len(buildArgsValue) > 0 {
		for _, v := range buildArgsValue {
			arr := strings.Split(v, "=")
			if len(arr) == 1 && len(arr[0]) > 0 {
				// Avoid masking default build arg value from Dockerfile if environment variable is not set
				// https://github.com/moby/moby/issues/24101
				val, ok := os.LookupEnv(arr[0])
				if ok {
					buildgArgs = append(buildgArgs, fmt.Sprintf("--build-arg=%s=%s", v, val))
				}
			} else if len(arr) > 1 && len(arr[0]) > 0 {
				buildgArgs = append(buildgArgs, "--build-arg="+v)
			} else {
				return fmt.Errorf("invalid build arg %q", v)
			}
		}
	}

	if imageValue, err := cmd.Flags().GetString("image"); err != nil {
		return err
	} else if imageValue != "" {
		buildgArgs = append(buildgArgs, "--image="+imageValue)
	}

	if sshValue, err := cmd.Flags().GetStringArray("ssh"); err != nil {
		return err
	} else if len(sshValue) > 0 {
		for _, v := range sshValue {
			buildgArgs = append(buildgArgs, "--ssh="+v)
		}
	}

	if secretValue, err := cmd.Flags().GetStringArray("secret"); err != nil {
		return err
	} else if len(secretValue) > 0 {
		for _, v := range secretValue {
			buildgArgs = append(buildgArgs, "--secret="+v)
		}
	}

	buildgCmd := exec.Command(buildgBinary, append(buildgArgs, args[0])...)
	buildgCmd.Env = os.Environ()
	buildgCmd.Stdin = cmd.InOrStdin()
	buildgCmd.Stdout = cmd.OutOrStdout()
	buildgCmd.Stderr = cmd.ErrOrStderr()
	if err := buildgCmd.Start(); err != nil {
		return err
	}

	return buildgCmd.Wait()
}
