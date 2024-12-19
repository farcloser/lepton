/*
   Copyright The containerd Authors.

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

package netutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"

	ncdefaults "github.com/containerd/nerdctl/v2/pkg/defaults"
	"github.com/containerd/nerdctl/v2/pkg/labels"
	"github.com/containerd/nerdctl/v2/pkg/rootlessutil"
)

const testBridgeIP = "10.1.100.1/24"

// Tests whether the default network was properly created when required
// with a custom bridge IP and subnet.
func testDefaultNetworkCreationWithBridgeIP(t *testing.T) {
	// To prevent subnet collisions when attempting to recreate the default network
	// in the isolated CNI config dir we'll be using, we must first delete
	// the network in the default CNI config dir.
	defaultCniEnv := CNIEnv{
		Path:        ncdefaults.CNIPath(),
		NetconfPath: ncdefaults.CNINetConfPath(),
	}
	defaultNet, err := defaultCniEnv.GetDefaultNetworkConfig()
	assert.NilError(t, err)
	if defaultNet != nil {
		assert.NilError(t, defaultCniEnv.RemoveNetwork(defaultNet))
	}

	// We create a tempdir for the CNI conf path to ensure an empty env for this test.
	cniConfTestDir := t.TempDir()
	cniEnv := CNIEnv{
		Path:        ncdefaults.CNIPath(),
		NetconfPath: cniConfTestDir,
	}
	// Ensure no default network config is not present.
	defaultNetConf, err := cniEnv.GetDefaultNetworkConfig()
	assert.NilError(t, err)
	assert.Assert(t, defaultNetConf == nil)

	// Attempt to create the default network with a test bridgeIP
	err = cniEnv.ensureDefaultNetworkConfig(testBridgeIP)
	assert.NilError(t, err)

	// Ensure default network config is present now.
	defaultNetConf, err = cniEnv.GetDefaultNetworkConfig()
	assert.NilError(t, err)
	assert.Assert(t, defaultNetConf != nil)

	// Check network config file present.
	stat, err := os.Stat(defaultNetConf.File)
	assert.NilError(t, err)
	firstConfigModTime := stat.ModTime()

	// Check default network label present.
	assert.Assert(t, defaultNetConf.CliLabels != nil)
	lstr, ok := (*defaultNetConf.CliLabels)[labels.DefaultNetwork]
	assert.Assert(t, ok)
	boolv, err := strconv.ParseBool(lstr)
	assert.NilError(t, err)
	assert.Assert(t, boolv)

	// Check bridge IP is set.
	assert.Assert(t, defaultNetConf.Plugins != nil)
	assert.Assert(t, len(defaultNetConf.Plugins) > 0)
	bridgePlugin := defaultNetConf.Plugins[0]
	var bridgeConfig struct {
		Type   string `json:"type"`
		Bridge string `json:"bridge"`
		IPAM   struct {
			Ranges [][]struct {
				Gateway string `json:"gateway"`
				Subnet  string `json:"subnet"`
			} `json:"ranges"`
			Routes []struct {
				Dst string `json:"dst"`
			} `json:"routes"`
			Type string `json:"type"`
		} `json:"ipam"`
	}

	err = json.Unmarshal(bridgePlugin.Bytes, &bridgeConfig)
	if err != nil {
		t.Fatalf("Failed to parse bridge plugin config: %v", err)
	}

	// Assert on bridge plugin configuration
	assert.Equal(t, "bridge", bridgeConfig.Type)
	// Assert on IPAM configuration
	assert.Equal(t, "10.1.100.1", bridgeConfig.IPAM.Ranges[0][0].Gateway)
	assert.Equal(t, "10.1.100.0/24", bridgeConfig.IPAM.Ranges[0][0].Subnet)
	assert.Equal(t, "0.0.0.0/0", bridgeConfig.IPAM.Routes[0].Dst)
	assert.Equal(t, "host-local", bridgeConfig.IPAM.Type)

	// Ensure network isn't created twice or accidentally re-created.
	err = cniEnv.ensureDefaultNetworkConfig(testBridgeIP)
	assert.NilError(t, err)

	// Check for any other network config files.
	files := []os.FileInfo{}
	walkF := func(p string, info os.FileInfo, err error) error {
		files = append(files, info)
		return nil
	}
	err = filepath.Walk(cniConfTestDir, walkF)
	assert.NilError(t, err)
	assert.Assert(t, len(files) == 2) // files[0] is the entry for '.'
	assert.Assert(t, filepath.Join(cniConfTestDir, files[1].Name()) == defaultNetConf.File)
	assert.Assert(t, firstConfigModTime == files[1].ModTime())
}

// Tests whether the default network when required was created properly when required.
// On Linux, the default driver used will be "bridge". (netutil.DefaultNetworkName)
func TestDefaultNetworkCreation(t *testing.T) {
	if rootlessutil.IsRootless() {
		t.Skip("must be superuser to create default network for this test")
	}

	testDefaultNetworkCreation(t)
	testDefaultNetworkCreationWithBridgeIP(t)
}
