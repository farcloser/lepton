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
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/leptonic/services/namespace"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/mountutil/volumestore"
)

type namespaceListOutput struct {
	Name       string            `json:"name"`
	Containers int               `json:"containers"`
	Images     int               `json:"images"`
	Volumes    int               `json:"volumes"`
	Labels     map[string]string `json:"labels,omitempty"`
}

func List(ctx context.Context, client *client.Client, output io.Writer, globalOptions *options.Global, opts *options.NamespaceList) error {
	namespaces, err := namespace.ListNames(ctx, client)
	if err != nil {
		return err
	}

	if opts.Quiet {
		for _, ns := range namespaces {
			if _, err = fmt.Fprintln(output, ns); err != nil {
				log.G(ctx).WithError(err).Error()
			}
		}
		return nil
	}

	dataStore, err := clientutil.DataStore(globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return err
	}

	result := []*namespaceListOutput{}
	for _, ns := range namespaces {
		entry := &namespaceListOutput{
			Name: ns,
		}

		nsCtx := namespace.NamespacedContext(ctx, ns)

		containers, err := client.Containers(nsCtx)
		if err != nil {
			log.L.Warn(err)
		}

		entry.Containers = len(containers)

		images, err := client.ImageService().List(nsCtx)
		if err != nil {
			log.L.Warn(err)
		}

		entry.Images = len(images)

		volStore, err := volumestore.New(dataStore, ns)
		if err != nil {
			log.L.Warn(err)
		} else {
			entry.Volumes, err = volStore.Count()
			if err != nil {
				log.L.Warn(err)
			}
		}

		entry.Labels, err = client.NamespaceService().Labels(nsCtx, ns)
		if err != nil {
			return err
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
		return tmpl.Execute(output, result)
	}

	w := tabwriter.NewWriter(output, 4, 8, 4, ' ', 0)

	// no "NETWORKS", because networks are global objects
	_, err = fmt.Fprintln(w, "NAME\tCONTAINERS\tIMAGES\tVOLUMES\tLABELS")
	if err != nil {
		log.G(ctx).WithError(err)
	}

	for _, ns := range result {
		labels := []string{}
		for k, v := range ns.Labels {
			labels = append(labels, strings.Join([]string{k, v}, "="))
		}
		_, err = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%v\t\n", ns.Name, ns.Containers,
			ns.Images, ns.Volumes, strings.Join(labels, ","))
		if err != nil {
			log.G(ctx).WithError(err)
		}
	}

	// for _, ns := range nsList {
	//	_, err = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%v\t\n", ns, numContainers, numImages, numVolumes, strings.Join(labelStrings, ","))
	//	if err != nil {
	//		log.G(ctx).WithError(err)
	//	}
	//}
	//
	return w.Flush()
}
