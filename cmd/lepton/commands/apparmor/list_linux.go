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

// See note in pkg/cmd/apparmor
/*
import (
	"errors"

	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/apparmor"
	"go.farcloser.world/lepton/pkg/formatter"
)

func listCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Aliases:       []string{"ls"},
		Short:         "List the loaded AppArmor profiles",
		Args:          cobra.NoArgs,
		RunE:          listAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolP("quiet", "q", false, "Only display profile names")
	cmd.Flags().String("format", "", "Format the output using the given go template")

	_ = cmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{formatter.FormatJSON, formatter.FormatTable, formatter.FormatWide}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func listOptions(cmd *cobra.Command, _ []string) (*options.AppArmorList, error) {
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return nil, err
	}

	format, err := cmd.Flags().GetString("format")
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
		return errors.Join(errs.ErrInvalidArgument, err)
	}

	return apparmor.List(cmd.OutOrStdout(), opts)
}

*/
