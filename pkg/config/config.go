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

package config

import (
	"github.com/containerd/containerd/v2/defaults"

	"go.farcloser.world/containers/security/cgroups"

	"go.farcloser.world/lepton/leptonic/services/namespace"
	ncdefaults "go.farcloser.world/lepton/pkg/defaults"
)

// Config corresponds to the cli toml file.
// See docs/config.md .
type Config struct {
	Debug            bool            `toml:"debug"`
	DebugFull        bool            `toml:"debug_full"`
	Address          string          `toml:"address"`
	Namespace        string          `toml:"namespace"`
	Snapshotter      string          `toml:"snapshotter"`
	CNIPath          string          `toml:"cni_path"`
	CNINetConfPath   string          `toml:"cni_netconfpath"`
	DataRoot         string          `toml:"data_root"`
	CgroupManager    cgroups.Manager `toml:"cgroup_manager"`
	InsecureRegistry bool            `toml:"insecure_registry"`
	HostsDir         []string        `toml:"hosts_dir"`
	Experimental     bool            `toml:"experimental"`
	HostGatewayIP    string          `toml:"host_gateway_ip"`
	BridgeIP         string          `toml:"bridge_ip, omitempty"`
	KubeHideDupe     bool            `toml:"kube_hide_dupe"`
}

// New creates a default Config object statically,
// without interpolating CLI flags, env vars, and toml.
func New() *Config {
	return &Config{
		Debug:            false,
		DebugFull:        false,
		Address:          defaults.DefaultAddress,
		Namespace:        namespace.Default,
		Snapshotter:      defaults.DefaultSnapshotter,
		CNIPath:          ncdefaults.CNIPath(),
		CNINetConfPath:   ncdefaults.CNINetConfPath(),
		DataRoot:         ncdefaults.DataRoot(),
		CgroupManager:    cgroups.DefaultManager(),
		InsecureRegistry: false,
		HostsDir:         ncdefaults.HostsDirs(),
		Experimental:     true,
		HostGatewayIP:    ncdefaults.HostGatewayIP(),
		KubeHideDupe:     false,
	}
}
