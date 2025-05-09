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

package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/template"

	"github.com/docker/cli/templates"
)

// Flusher is implemented by text/tabwriter.Writer
type Flusher interface {
	Flush() error
}

// FormatSlice formats the slice with `--format` flag.
//
// --format="" (default): JSON
// --format='{{json .}}': JSON lines
//
// FormatSlice is expected to be only used for `OBJECT inspect` commands.
func FormatSlice(format string, writer io.Writer, x []interface{}) error {
	var tmpl *template.Template
	switch format {
	case FormatNone:
		// Avoid escaping "<", ">", "&"
		// https://pkg.go.dev/encoding/json
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "    ")
		encoder.SetEscapeHTML(false)
		err := encoder.Encode(x)
		if err != nil {
			return err
		}
		fmt.Fprint(writer, "\n")
	case FormatTable, FormatWide:
		return errors.New("unsupported format: \"table\", and \"wide\"")
	default:
		var err error
		tmpl, err = ParseTemplate(format)
		if err != nil {
			return err
		}
		for _, f := range x {
			var b bytes.Buffer
			if err := tmpl.Execute(&b, f); err != nil {
				if _, ok := err.(template.ExecError); ok { //nolint:errorlint
					// FallBack to Raw Format
					if err = tryRawFormat(&b, f, tmpl); err != nil {
						return err
					}
				}
			}
			if _, err = fmt.Fprintln(writer, b.String()); err != nil {
				return err
			}
		}
	}
	return nil
}

// FIXME: is this really serving a purpose?
func tryRawFormat(b *bytes.Buffer, f interface{}, tmpl *template.Template) error {
	m, err := json.MarshalIndent(f, "", "    ")
	if err != nil {
		return err
	}

	var raw interface{}
	rdr := bytes.NewReader(m)
	dec := json.NewDecoder(rdr)
	dec.UseNumber()

	if rawErr := dec.Decode(&raw); rawErr != nil {
		return fmt.Errorf("unable to read inspect data: %w", rawErr)
	}

	tmplMissingKey := tmpl.Option("missingkey=error")
	if rawErr := tmplMissingKey.Execute(b, raw); rawErr != nil {
		return fmt.Errorf("template parsing error: %w", rawErr)
	}

	return nil
}

// ParseTemplate wraps github.com/docker/cli/templates.Parse() to allow `json` as an alias of `{{json .}}`.
// ParseTemplate can be removed when https://github.com/docker/cli/pull/3355 gets merged and tagged (Docker 22.XX).
// FIXME: remove this entirely - json should output valid json - {{ json . }} may output a stream
func ParseTemplate(format string) (*template.Template, error) {
	aliases := map[string]string{
		FormatJSON: "{{json .}}",
	}

	if alias, ok := aliases[format]; ok {
		format = alias
	}

	return templates.Parse(format)
}
