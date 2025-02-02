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

package internal

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/ocihook"
)

func ociHookCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "oci-hook",
		Short:         "OCI hook",
		RunE:          internalOCIHookAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func internalOCIHookAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	event := ""
	if len(args) > 0 {
		event = args[0]
	}

	if event == "" {
		return errors.New("event type needs to be passed")
	}

	dataStore, err := clientutil.DataStore(globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return err
	}

	return ocihook.Run(os.Stdin, os.Stderr, event,
		dataStore,
		globalOptions.CNIPath,
		globalOptions.CNINetConfPath,
		globalOptions.BridgeIP,
	)
}
