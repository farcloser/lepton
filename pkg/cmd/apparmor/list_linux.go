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
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/containerd/nerdctl/v2/leptonic/services/apparmor"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
)

func List(out io.Writer, options *ListOptions) error {
	profiles, err := apparmor.List()
	if err != nil {
		return err
	}

	quiet := options.Quiet
	format := options.Format

	switch format {
	case formatter.FormatNone, formatter.FormatTable, formatter.FormatWide:
		out = tabwriter.NewWriter(out, 4, 8, 4, ' ', 0)
		if !quiet {
			if _, err = fmt.Fprintln(out, "NAME\tMODE"); err != nil {
				return err
			}
		}

		for _, f := range profiles {
			if quiet {
				_, _ = fmt.Fprintln(out, f.Name)
			} else {
				_, _ = fmt.Fprintf(out, "%s\t%s\n", f.Name, f.Mode)
			}
		}

		if f, ok := out.(formatter.Flusher); ok {
			return f.Flush()
		}

	case formatter.FormatJSON:
		toPrint, err := formatter.ToJSON(profiles, "", "   ")
		if err != nil {
			return err
		}

		if _, err = fmt.Fprint(out, toPrint); err != nil {
			return err
		}

	default:
		tmpl, err := formatter.ParseTemplate(format)
		if err != nil {
			return err
		}

		for _, f := range profiles {
			var b bytes.Buffer
			if err := tmpl.Execute(&b, f); err != nil {
				return err
			}

			if _, err = fmt.Fprintln(out, b.String()); err != nil {
				return err
			}
		}
	}

	return nil
}
