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

package defaults

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/v2/plugins"
	"github.com/containerd/go-cni"

	"github.com/containerd/nerdctl/v2/pkg/rootlessutil"
	"github.com/containerd/nerdctl/v2/pkg/version"
)

const (
	SeccompProfileName = "builtin"
	Runtime            = plugins.RuntimeRuncV2
)

var (
	AppArmorProfileName = version.RootName + "-default"
)

func DataRoot() string {
	if !rootlessutil.IsRootless() {
		return "/var/lib/" + version.RootName
	}
	xdh, err := rootlessutil.XDGDataHome()
	if err != nil {
		panic(err)
	}
	return filepath.Join(xdh, version.RootName)
}

func CNIPath() string {
	candidates := []string{
		cni.DefaultCNIDir, // /opt/cni/bin
		"/usr/local/libexec/cni",
		"/usr/local/lib/cni",
		"/usr/libexec/cni", // Fedora
		"/usr/lib/cni",     // debian (containernetworking-plugins)
	}
	if rootlessutil.IsRootless() {
		home := os.Getenv("HOME")
		if home == "" {
			panic("environment variable HOME is not set")
		}
		candidates = append([]string{
			// NOTE: These user paths are not defined in XDG
			filepath.Join(home, "opt/cni/bin"),
			filepath.Join(home, ".local/libexec/cni"),
			filepath.Join(home, ".local/lib/cni"),
		}, candidates...)
	}

	for _, f := range candidates {
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}

	// default: /opt/cni/bin
	return cni.DefaultCNIDir
}

func CNINetConfPath() string {
	if !rootlessutil.IsRootless() {
		return cni.DefaultNetDir
	}
	xch, err := rootlessutil.XDGConfigHome()
	if err != nil {
		panic(err)
	}
	return filepath.Join(xch, "cni/net.d")
}

func CNIRuntimeDir() (string, error) {
	if !rootlessutil.IsRootless() {
		return "/run/cni", nil
	}
	xdr, err := rootlessutil.XDGRuntimeDir()
	if err != nil {
		if rootlessutil.IsRootlessChild() {
			return "", err
		}
		xdr = fmt.Sprintf("/run/user/%d", os.Geteuid())
	}
	return filepath.Join(xdr, "cni"), nil
}

func CliTOML() string {
	if !rootlessutil.IsRootless() {
		return fmt.Sprintf("/etc/%s/%s.toml", version.RootName, version.RootName)
	}
	xch, err := rootlessutil.XDGConfigHome()
	if err != nil {
		panic(err)
	}
	return filepath.Join(xch, version.RootName, version.RootName+".toml")
}

func HostsDirs() []string {
	if !rootlessutil.IsRootless() {
		return []string{"/etc/containerd/certs.d", "/etc/docker/certs.d"}
	}
	xch, err := rootlessutil.XDGConfigHome()
	if err != nil {
		panic(err)
	}
	return []string{
		filepath.Join(xch, "containerd/certs.d"),
		filepath.Join(xch, "docker/certs.d"),
	}
}

// HostGatewayIP returns the non-loop-back host ip if available and returns empty string if running into error.
func HostGatewayIP() string {
	// no need to use [rootlessutil.WithDetachedNetNSIfAny] here
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
