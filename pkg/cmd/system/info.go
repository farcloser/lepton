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

package system

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/containerd/containerd/api/services/introspection/v1"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"go.farcloser.world/core/units"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/infoutil"
	"go.farcloser.world/lepton/pkg/inspecttypes/dockercompat"
	"go.farcloser.world/lepton/pkg/inspecttypes/native"
	"go.farcloser.world/lepton/pkg/logging"
	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/strutil"
)

func Info(
	ctx context.Context,
	client *containerd.Client,
	output io.Writer,
	globalOptions *options.Global,
	opts *options.SystemInfo,
) error {
	var (
		tmpl *template.Template
		err  error
	)
	if opts.Format != "" {
		tmpl, err = formatter.ParseTemplate(opts.Format)
		if err != nil {
			return err
		}
	}

	var (
		infoNative *native.Info
		infoCompat *dockercompat.Info
	)
	switch opts.Mode {
	case "native":
		di, err := infoutil.NativeDaemonInfo(ctx, client)
		if err != nil {
			return err
		}
		infoNative = fulfillNativeInfo(di, globalOptions)
	case "dockercompat":
		infoCompat, err = infoutil.Info(ctx, client, globalOptions.Snapshotter, globalOptions.CgroupManager)
		if err != nil {
			return err
		}
		infoCompat.Plugins.Log = logging.Drivers()
	default:
		return fmt.Errorf("unknown mode %q", opts.Mode)
	}

	if tmpl != nil {
		var x interface{} = infoNative
		if infoCompat != nil {
			x = infoCompat
		}
		w := output
		if err := tmpl.Execute(w, x); err != nil {
			return err
		}
		_, err = fmt.Fprintln(w)
		return err
	}

	switch opts.Mode {
	case "native":
		return prettyPrintInfoNative(output, infoNative)
	case "dockercompat":
		return prettyPrintInfoDockerCompat(output, opts.Stderr, infoCompat, globalOptions)
	}
	return nil
}

func fulfillNativeInfo(di *native.DaemonInfo, globalOptions *options.Global) *native.Info {
	info := &native.Info{
		Daemon: di,
	}
	info.Namespace = globalOptions.Namespace
	info.Snapshotter = globalOptions.Snapshotter
	info.CgroupManager = globalOptions.CgroupManager
	info.Rootless = rootlessutil.IsRootless()
	return info
}

func prettyPrintInfoNative(w io.Writer, info *native.Info) error {
	fmt.Fprintf(w, "Namespace:          %s\n", info.Namespace)
	fmt.Fprintf(w, "Snapshotter:        %s\n", info.Snapshotter)
	fmt.Fprintf(w, "Cgroup Manager:     %s\n", info.CgroupManager)
	fmt.Fprintf(w, "Rootless:           %v\n", info.Rootless)
	fmt.Fprintf(w, "containerd Version: %s (%s)\n", info.Daemon.Version.Version, info.Daemon.Version.Revision)
	fmt.Fprintf(w, "containerd UUID:    %s\n", info.Daemon.Server.UUID)
	var disabledPlugins, enabledPlugins []*introspection.Plugin
	for _, f := range info.Daemon.Plugins.Plugins {
		if f.InitErr == nil {
			enabledPlugins = append(enabledPlugins, f)
		} else {
			disabledPlugins = append(disabledPlugins, f)
		}
	}
	sorter := func(x []*introspection.Plugin) func(int, int) bool {
		return func(i, j int) bool {
			return x[i].Type+"."+x[i].ID < x[j].Type+"."+x[j].ID
		}
	}
	sort.Slice(enabledPlugins, sorter(enabledPlugins))
	sort.Slice(disabledPlugins, sorter(disabledPlugins))
	fmt.Fprintln(w, "containerd Plugins:")
	for _, f := range enabledPlugins {
		fmt.Fprintf(w, " - %s.%s\n", f.Type, f.ID)
	}
	fmt.Fprintf(w, "containerd Plugins (disabled):\n")
	for _, f := range disabledPlugins {
		fmt.Fprintf(w, " - %s.%s\n", f.Type, f.ID)
	}
	return nil
}

func prettyPrintInfoDockerCompat(
	stdout io.Writer,
	stderr io.Writer,
	info *dockercompat.Info,
	globalOptions *options.Global,
) error {
	w := stdout
	debug := globalOptions.Debug
	fmt.Fprintf(w, "Client:\n")
	fmt.Fprintf(w, " Namespace:\t%s\n", globalOptions.Namespace)
	fmt.Fprintf(w, " Debug Mode:\t%v\n", debug)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Server:\n")
	fmt.Fprintf(w, " Server Version: %s\n", info.ServerVersion)
	// Storage Driver is not really Server concept, but mimics `docker info` output
	fmt.Fprintf(w, " Storage Driver: %s\n", info.Driver)
	fmt.Fprintf(w, " Logging Driver: %s\n", info.LoggingDriver)
	printF(w, " Cgroup Driver: ", string(info.CgroupDriver))
	printF(w, " Cgroup Version: ", info.CgroupVersion)
	fmt.Fprintf(w, " Plugins:\n")
	fmt.Fprintf(w, "  Log:     %s\n", strings.Join(info.Plugins.Log, " "))
	fmt.Fprintf(w, "  Storage: %s\n", strings.Join(info.Plugins.Storage, " "))

	// print Security options
	printSecurityOptions(w, info.SecurityOptions)

	fmt.Fprintf(w, " Kernel Version:   %s\n", info.KernelVersion)
	fmt.Fprintf(w, " Operating System: %s\n", info.OperatingSystem)
	fmt.Fprintf(w, " OSType:           %s\n", info.OSType)
	fmt.Fprintf(w, " Architecture:     %s\n", info.Architecture)
	fmt.Fprintf(w, " CPUs:             %d\n", info.NCPU)
	fmt.Fprintf(w, " Total Memory:     %s\n", units.BytesSize(float64(info.MemTotal)))
	fmt.Fprintf(w, " Name:             %s\n", info.Name)
	fmt.Fprintf(w, " ID:               %s\n", info.ID)

	fmt.Fprintln(w)
	if len(info.Warnings) > 0 {
		fmt.Fprintln(stderr, strings.Join(info.Warnings, "\n"))
	}
	return nil
}

func printF(w io.Writer, label, dockerCompatInfo string) {
	if dockerCompatInfo == "" {
		return
	}
	fmt.Fprintf(w, "%s%s\n", label, dockerCompatInfo)
}

func printSecurityOptions(w io.Writer, securityOptions []string) {
	if len(securityOptions) == 0 {
		return
	}

	fmt.Fprintf(w, " Security Options:\n")
	for _, s := range securityOptions {
		m, err := strutil.ParseCSVMap(s)
		if err != nil {
			log.L.WithError(err).Warnf("unparsable security option %q", s)
			continue
		}
		name := m["name"]
		if name == "" {
			log.L.Warnf("unparsable security option %q", s)
			continue
		}
		fmt.Fprintf(w, "  %s\n", name)
		for k, v := range m {
			if k == "name" {
				continue
			}
			fmt.Fprintf(w, "   %s: %s\n", cases.Title(language.English).String(k), v)
		}
	}
}
