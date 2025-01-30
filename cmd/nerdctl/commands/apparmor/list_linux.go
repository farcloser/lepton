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

package apparmor

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/apparmor"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Aliases:       []string{"ls"},
		Short:         "NamesList the loaded AppArmor profiles",
		Args:          cobra.NoArgs,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP(flagQuiet, "q", false, "Only display profile names")
	cmd.Flags().String(flagFormat, "", "Format the output using the given go template")

	_ = cmd.RegisterFlagCompletionFunc(flagFormat, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON, formatter.FormatTable, formatter.FormatWide}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func listOptions(cmd *cobra.Command, _ []string) (*options.AppArmorList, error) {
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString(flagFormat)
	if err != nil {
		return nil, err
	}

	if format != formatter.FormatNone && format != formatter.FormatWide && format != formatter.FormatTable && quiet {
		return nil, errors.New("custom or json format, and 'quiet', cannot be specified together")
	}

	return &options.AppArmorList{
		Quiet:  quiet,
		Format: format,
	}, nil
}

func listAction(cmd *cobra.Command, args []string) error {
	opts, err := listOptions(cmd, args)
	if err != nil {
		return err
	}

	return apparmor.List(cmd.OutOrStdout(), opts)
}
