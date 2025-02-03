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

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/errs"
)

func debugCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "debug",
		Short:         "Debug Dockerfile",
		PreRunE:       helpers.RequireExperimental("`builder debug`"),
		Args:          cobra.MinimumNArgs(1),
		RunE:          debugAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringP("file", "f", "", "Name of the Dockerfile")
	cmd.Flags().String("target", "", "Set the target build stage to build")
	cmd.Flags().StringArray("build-arg", nil, "Set build-time variables")
	cmd.Flags().String("image", "", "Image to use for debugging stage")
	cmd.Flags().StringArray("ssh", nil, "Allow forwarding SSH agent to the build. Format: default|<id>[=<socket>|<key>[,<key>]]")
	cmd.Flags().StringArray("secret", nil, "Expose secret value to the build. Format: id=secretname,src=filepath")

	return cmd
}

func debugOptions(cmd *cobra.Command, _ []string) ([]string, error) {
	buildgArgs := []string{"debug"}

	if file, err := cmd.Flags().GetString("file"); err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	} else if file != "" {
		buildgArgs = append(buildgArgs, "--file="+file)
	}

	if target, err := cmd.Flags().GetString("target"); err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	} else if target != "" {
		buildgArgs = append(buildgArgs, "--target="+target)
	}

	if buildArgsValue, err := cmd.Flags().GetStringArray("build-arg"); err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	} else if len(buildArgsValue) > 0 {
		for _, v := range buildArgsValue {
			arr := strings.Split(v, "=")
			if len(arr) == 1 && len(arr[0]) > 0 {
				// Avoid masking default build arg value from Dockerfile if environment variable is not set
				// https://github.com/moby/moby/issues/24101
				if val, ok := os.LookupEnv(arr[0]); ok {
					buildgArgs = append(buildgArgs, fmt.Sprintf("--build-arg=%s=%s", v, val))
				}
			} else if len(arr) > 1 && len(arr[0]) > 0 {
				buildgArgs = append(buildgArgs, "--build-arg="+v)
			} else {
				return nil, errors.Join(errs.ErrInvalidArgument, fmt.Errorf("invalid build arg %q", v))
			}
		}
	}

	if imageValue, err := cmd.Flags().GetString("image"); err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	} else if imageValue != "" {
		buildgArgs = append(buildgArgs, "--image="+imageValue)
	}

	if sshValue, err := cmd.Flags().GetStringArray("ssh"); err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	} else if len(sshValue) > 0 {
		for _, v := range sshValue {
			buildgArgs = append(buildgArgs, "--ssh="+v)
		}
	}

	if secretValue, err := cmd.Flags().GetStringArray("secret"); err != nil {
		return nil, errors.Join(errs.ErrInvalidArgument, err)
	} else if len(secretValue) > 0 {
		for _, v := range secretValue {
			buildgArgs = append(buildgArgs, "--secret="+v)
		}
	}

	return buildgArgs, nil
}

func debugAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	buildgBinary, err := exec.LookPath("buildg")
	if err != nil {
		return errors.Join(errs.ErrFailedPrecondition, err)
	}

	buildgArgs, err := debugOptions(cmd, args)
	if err != nil {
		return err
	}

	if globalOptions.Debug {
		buildgArgs = append([]string{"--debug"}, buildgArgs...)
	}

	buildgCmd := exec.Command(buildgBinary, append(buildgArgs, args[0])...)
	buildgCmd.Env = os.Environ()
	buildgCmd.Stdin = cmd.InOrStdin()
	buildgCmd.Stdout = cmd.OutOrStdout()
	buildgCmd.Stderr = cmd.ErrOrStderr()

	if err := buildgCmd.Start(); err != nil {
		return errors.Join(errs.ErrSystemFailure, err)
	}

	return buildgCmd.Wait()
}
