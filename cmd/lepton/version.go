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

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/containerd/log"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/cmd/lepton/helpers"
	"go.farcloser.world/lepton/leptonic/services/containerd"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/infoutil"
	"go.farcloser.world/lepton/pkg/inspecttypes/dockercompat"
	"go.farcloser.world/lepton/pkg/rootlessutil"
)

func newVersionCommand() *cobra.Command {
	versionCommand := &cobra.Command{
		Use:           "version",
		Args:          cobra.NoArgs,
		Short:         "Show version information",
		RunE:          versionAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	versionCommand.Flags().
		StringP("format", "f", "", "Format the output using the given Go template, e.g, '{{json .}}'")
	versionCommand.RegisterFlagCompletionFunc(
		"format",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{formatter.FormatJSON}, cobra.ShellCompDirectiveNoFileComp
		},
	)
	return versionCommand
}

func versionAction(cmd *cobra.Command, _ []string) error {
	var w io.Writer = os.Stdout
	var tmpl *template.Template
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}
	if format != "" {
		var err error
		tmpl, err = formatter.ParseTemplate(format)
		if err != nil {
			return err
		}
	}

	address := globalOptions.Address
	// rootless `version` runs in the host namespaces, so the address is different
	if rootlessutil.IsRootless() {
		address, err = rootlessutil.RootlessContainredSockAddress()
		if err != nil {
			log.L.WithError(err).Warning("failed to inspect the rootless containerd socket address")
			address = ""
		}
	}

	v, vErr := versionInfo(cmd, globalOptions.Namespace, address)
	if tmpl != nil {
		var b bytes.Buffer
		if err := tmpl.Execute(&b, v); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, b.String()); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(w, "Client:")
		fmt.Fprintf(w, " Version:\t%s\n", v.Client.Version)
		fmt.Fprintf(w, " OS/Arch:\t%s/%s\n", v.Client.Os, v.Client.Arch)
		fmt.Fprintf(w, " Git commit:\t%s\n", v.Client.GitCommit)
		for _, compo := range v.Client.Components {
			fmt.Fprintf(w, " %s:\n", compo.Name)
			fmt.Fprintf(w, "  Version:\t%s\n", compo.Version)
			for detailK, detailV := range compo.Details {
				fmt.Fprintf(w, "  %s:\t%s\n", detailK, detailV)
			}
		}
		if v.Server != nil {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Server:")
			for _, compo := range v.Server.Components {
				fmt.Fprintf(w, " %s:\n", compo.Name)
				fmt.Fprintf(w, "  Version:\t%s\n", compo.Version)
				for detailK, detailV := range compo.Details {
					fmt.Fprintf(w, "  %s:\t%s\n", detailK, detailV)
				}
			}
		}
	}
	return vErr
}

// versionInfo may return partial VersionInfo on error.
// Address can be empty to skip inspecting the server.
func versionInfo(cmd *cobra.Command, ns, address string) (dockercompat.VersionInfo, error) {
	v := dockercompat.VersionInfo{
		Client: infoutil.ClientVersion(),
	}
	if address == "" {
		return v, nil
	}
	cli, ctx, cancel, err := containerd.NewClient(cmd.Context(), ns, address)
	if err != nil {
		return v, err
	}
	defer cancel()
	v.Server, err = infoutil.ServerVersion(ctx, cli)
	return v, err
}
