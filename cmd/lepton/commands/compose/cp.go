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
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/cmd/compose"
	"go.farcloser.world/lepton/pkg/composer"
	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/version"
)

func copyCommand() *cobra.Command {
	usage := fmt.Sprintf(`cp [OPTIONS] SERVICE:SRC_PATH DEST_PATH|-
       %s compose cp [OPTIONS] SRC_PATH|- SERVICE:DEST_PATH`, version.RootName)
	var cmd = &cobra.Command{
		Use:           usage,
		Short:         "Copy files/folders between a service container and the local filesystem",
		Args:          cobra.ExactArgs(2),
		RunE:          copyAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().Bool("dry-run", false, "Execute command in dry run mode")
	cmd.Flags().BoolP("follow-link", "L", false, "Always follow symbol link in SRC_PATH")
	cmd.Flags().Int("index", 0, "index of the container if service has multiple replicas")

	return cmd
}

func copyAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	source := args[0]
	if source == "" {
		return errors.New("source can not be empty")
	}
	destination := args[1]
	if destination == "" {
		return errors.New("destination can not be empty")
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	followLink, err := cmd.Flags().GetBool("follow-link")
	if err != nil {
		return err
	}
	index, err := cmd.Flags().GetInt("index")
	if err != nil {
		return err
	}

	// rootless cp runs in the host namespaces, so the address is different
	if rootlessutil.IsRootless() {
		globalOptions.Address, err = rootlessutil.RootlessContainredSockAddress()
		if err != nil {
			return err
		}
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

	co := composer.CopyOptions{
		Source:      source,
		Destination: destination,
		Index:       index,
		FollowLink:  followLink,
		DryRun:      dryRun,
	}
	return c.Copy(ctx, co)

}
