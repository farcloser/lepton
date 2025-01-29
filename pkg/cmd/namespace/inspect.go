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

package namespace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/clientutil"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
	"github.com/containerd/nerdctl/v2/pkg/inspecttypes/native"
	"github.com/containerd/nerdctl/v2/pkg/mountutil/volumestore"
)

type namespaceInspectOutput struct {
	Name       string                   `json:"name"`
	Containers []client.Container       `json:"containers"`
	Images     []images.Image           `json:"images"`
	Volumes    map[string]native.Volume `json:"volumes"`
	Labels     map[string]string        `json:"labels,omitempty"`
}

func Inspect(ctx context.Context, client *client.Client, output io.Writer, globalOptions *options.Global, opts *options.NamespaceInspect) error {
	namespaces, errs := namespace.Inspect(ctx, client, opts.NamesList)
	if len(errs) > 0 {
		for _, err := range errs {
			log.G(ctx).WithError(err).Error()
		}
	}

	dataStore, err := clientutil.DataStore(globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return err
	}

	result := []*namespaceInspectOutput{}
	for _, ns := range namespaces {
		entry := &namespaceInspectOutput{
			Name:   ns.Name,
			Labels: ns.Labels,
		}

		nsCtx := namespace.NamespacedContext(ctx, ns.Name)

		cntnrs, err := client.Containers(nsCtx)
		if err != nil {
			log.L.Warn(err)
		}

		entry.Containers = cntnrs

		images, err := client.ImageService().List(nsCtx)
		if err != nil {
			log.L.Warn(err)
		}

		entry.Images = images

		volStore, err := volumestore.New(dataStore, ns.Name)
		if err != nil {
			log.L.Warn(err)
		} else {
			entry.Volumes, err = volStore.List(false)
			if err != nil {
				log.L.Warn(err)
			}
		}

		result = append(result, entry)
	}

	switch opts.Format {
	case formatter.FormatNone, formatter.FormatTable, formatter.FormatWide:
	default:
		tmpl, err := formatter.ParseTemplate(opts.Format)
		if err != nil {
			return err
		}

		fErr := tmpl.Execute(output, result)
		if fErr != nil {
			return fErr
		}

		if len(errs) > 0 {
			return errors.New("")
		}

		return nil
	}

	// no "NETWORKS", because networks are global objects
	if len(result) > 0 {
		w := tabwriter.NewWriter(output, 4, 8, 4, ' ', 0)

		_, err = fmt.Fprintln(w, "NAME\tCONTAINERS\tIMAGES\tVOLUMES\tLABELS")
		if err != nil {
			log.G(ctx).WithError(err)
		}

		for _, ns := range result {
			labels := []string{}
			for k, v := range ns.Labels {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v))
			}

			_, err = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%v\t\n", ns.Name, len(ns.Containers),
				len(ns.Images), len(ns.Volumes), strings.Join(labels, ","))
			if err != nil {
				log.G(ctx).WithError(err)
			}
		}
		fErr := w.Flush()
		if fErr != nil {
			return fErr
		}
	}

	if len(errs) > 0 {
		return errors.New("")
	}

	return nil
}
