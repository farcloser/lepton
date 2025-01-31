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

// NOTE: none of this makes sense in our context. Other apps' AppArmor profiles are irrelevant,
// and cannot be loaded currently anyhow.
/*
import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/containerd/nerdctl/v2/leptonic/services/apparmor"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
)

func List(output io.Writer, options *options.AppArmorList) error {
	profiles, err := apparmor.List()
	if err != nil {
		return err
	}

	switch options.Format {
	case formatter.FormatNone, formatter.FormatTable, formatter.FormatWide:
		output = tabwriter.NewWriter(output, 4, 8, 4, ' ', 0)
		if !options.Quiet {
			if _, err = fmt.Fprintln(output, "NAME\tMODE"); err != nil {
				return err
			}
		}

		for _, profile := range profiles {
			if options.Quiet {
				_, _ = fmt.Fprintln(output, profile.Name)
			} else {
				_, _ = fmt.Fprintf(output, "%s\t%s\n", profile.Name, profile.Mode)
			}
		}

		if f, ok := output.(formatter.Flusher); ok {
			return f.Flush()
		}

	case formatter.FormatJSON:
		toPrint, err := formatter.ToJSON(profiles, "", "   ")
		if err != nil {
			return err
		}

		if _, err = fmt.Fprint(output, toPrint); err != nil {
			return err
		}

	default:
		tmpl, err := formatter.ParseTemplate(options.Format)
		if err != nil {
			return err
		}

		for _, f := range profiles {
			var buff bytes.Buffer
			if err := tmpl.Execute(&buff, f); err != nil {
				return err
			}

			if _, err = fmt.Fprintln(output, buff.String()); err != nil {
				return err
			}
		}
	}

	return nil
}
*/
