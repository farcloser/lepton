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
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/cmd/compose"
	"go.farcloser.world/lepton/pkg/composer"
)

func logsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "logs [flags] [SERVICE...]",
		Short:         "Show logs of running containers",
		RunE:          logsAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("follow", "f", false, "Follow log output.")
	cmd.Flags().BoolP("timestamps", "t", false, "Show timestamps")
	cmd.Flags().String("tail", "all", "Number of lines to show from the end of the logs")
	cmd.Flags().Bool("no-color", false, "Produce monochrome output")
	cmd.Flags().Bool("no-log-prefix", false, "Don't print prefix in logs")

	return cmd
}

func logsAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return err
	}
	timestamps, err := cmd.Flags().GetBool("timestamps")
	if err != nil {
		return err
	}
	tail, err := cmd.Flags().GetString("tail")
	if err != nil {
		return err
	}
	noColor, err := cmd.Flags().GetBool("no-color")
	if err != nil {
		return err
	}
	noLogPrefix, err := cmd.Flags().GetBool("no-log-prefix")
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}
	defer cancel()
	options, err := getComposeOptions(cmd, globalOptions.DebugFull, globalOptions.Experimental)
	if err != nil {
		return err
	}
	c, err := compose.New(cli, globalOptions, options, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	lo := composer.LogsOptions{
		Follow:      follow,
		Timestamps:  timestamps,
		Tail:        tail,
		NoColor:     noColor,
		NoLogPrefix: noLogPrefix,
	}
	return c.Logs(ctx, lo, args)
}
