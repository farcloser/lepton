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
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/clientutil"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
	"github.com/containerd/nerdctl/v2/pkg/mountutil/volumestore"
)

type namespaceListOptions struct {
	Quiet  bool
	Format string
}

func listCommand() *cobra.Command {
	namespaceLsCommand := &cobra.Command{
		Use:           "ls",
		Aliases:       []string{"list"},
		Short:         "ListNames containerd namespaces",
		RunE:          namespaceLsAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	namespaceLsCommand.Flags().BoolP("quiet", "q", false, "Only display names")
	namespaceLsCommand.Flags().String("format", "", "Format the output using the given Go template, e.g, '{{json .}}'")

	return namespaceLsCommand
}

func processNamespaceListCommandOption(cmd *cobra.Command) (*options.Global, *namespaceListOptions, error) {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return nil, nil, err
	}

	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return globalOptions, nil, err
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return globalOptions, nil, err
	}

	return globalOptions, &namespaceListOptions{
		Quiet:  quiet,
		Format: format,
	}, nil
}

type namespaceListOutput struct {
	Name       string            `json:"name"`
	Containers int               `json:"containers"`
	Images     int               `json:"images"`
	Volumes    int               `json:"volumes"`
	Labels     map[string]string `json:"labels,omitempty"`
}

func namespaceLsAction(cmd *cobra.Command, args []string) error {
	globalOptions, options, err := processNamespaceListCommandOption(cmd)
	if err != nil {
		return err
	}

	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), globalOptions.Namespace, globalOptions.Address)
	if err != nil {
		return err
	}

	defer cancel()

	namespaces, err := namespace.ListNames(ctx, cli)
	if err != nil {
		return err
	}

	if options.Quiet {
		for _, ns := range namespaces {
			if _, err = fmt.Fprintln(cmd.OutOrStdout(), ns); err != nil {
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

		containers, err := cli.Containers(nsCtx)
		if err != nil {
			log.L.Warn(err)
		}

		entry.Containers = len(containers)

		images, err := cli.ImageService().List(nsCtx)
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

		entry.Labels, err = cli.NamespaceService().Labels(nsCtx, ns)
		if err != nil {
			return err
		}

		result = append(result, entry)
	}

	switch options.Format {
	case formatter.FormatNone, formatter.FormatTable, formatter.FormatWide:
	default:
		tmpl, err := formatter.ParseTemplate(options.Format)
		if err != nil {
			return err
		}
		return tmpl.Execute(cmd.OutOrStdout(), result)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 4, 8, 4, ' ', 0)

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
