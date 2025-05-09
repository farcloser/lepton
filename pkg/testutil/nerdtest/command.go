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

// Package nerdtest
package nerdtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/version"
)

var defaultNamespace = testutil.Namespace

// IMPORTANT note on file writing here:
// Inside the context of a single test, there is no concurrency, as setup, command and cleanup operate in sequence
// Furthermore, the tempdir is private by definition.
// Writing files here in a non-safe manner is thus OK.
type target = string

const (
	targetNerdctl = target("nerdctl")
	targetDocker  = target("docker")
)

var targetNerdishctl = version.RootName

func getTarget() string {
	// Indirecting to testutil for now
	return testutil.GetTarget()
}

func newNerdCommand(conf test.Config, t *testing.T) *nerdCommand {
	// Decide what binary we are running
	var err error
	var binary string
	trgt := getTarget()
	envPrefix := strings.ToUpper(trgt)
	binary, err = exec.LookPath(trgt)
	if err != nil {
		t.Fatalf("unable to find binary %q: %v", trgt, err)
	}

	switch trgt {
	case targetNerdctl, targetNerdishctl:
		// Set the default namespace if we do not have something already
		if conf.Read(Namespace) == "" {
			conf.Write(Namespace, test.ConfigValue(defaultNamespace))
		}
	case targetDocker:
		if err = exec.Command("docker", "compose", "version").Run(); err != nil {
			t.Fatalf("docker does not support compose: %v", err)
		}
	default:
		t.Fatalf("unknown target %q", trgt)
	}

	// Create the base command, with the right binary, t
	ret := &nerdCommand{
		GenericCommand: *(test.NewGenericCommand().(*test.GenericCommand)),
	}

	ret.WithBinary(binary)
	ret.WithBlacklist([]string{
		// Insulate us from parent environment side effects
		"DOCKER_CONFIG",
		"CONTAINERD_SNAPSHOTTER",
		envPrefix + "_TOML",
		"CONTAINERD_ADDRESS",
		"CNI_PATH",
		"NETCONFPATH",
		envPrefix + "_EXPERIMENTAL",
		envPrefix + "_HOST_GATEWAY_IP",
		// On CI: noisy and irrelevant
		"LS_COLORS",
		"ANDROID_",
		"JAVA_",
		"HOMEBREW_",
		"GOROOT_",
		"DOTNET_",
		"SUDO_",
		"DEBIAN_",
		"MEMORY_",
		"SYSTEMD_",
		"GO",
	})
	return ret
}

type nerdCommand struct {
	test.GenericCommand

	hasWrittenToml         bool
	hasWrittenDockerConfig bool
}

func (nc *nerdCommand) Run(expect *test.Expected) {
	nc.prep()
	if getTarget() == targetDocker {
		// We are not in the business of testing docker *error* output, so, spay expectation here
		if expect != nil {
			expect.Errors = nil
		}
	}
	nc.GenericCommand.Run(expect)
}

func (nc *nerdCommand) Background() {
	nc.prep()
	nc.GenericCommand.Background()
}

// Run does override the generic command run
func (nc *nerdCommand) prep() {
	nc.T().Helper()

	// If no DOCKER_CONFIG was explicitly provided, set ourselves inside the current working directory
	if nc.Env["DOCKER_CONFIG"] == "" {
		nc.Env["DOCKER_CONFIG"] = nc.TempDir
	}

	if customDCConfig := nc.Config.Read(DockerConfig); customDCConfig != "" {
		if !nc.hasWrittenDockerConfig {
			dest := filepath.Join(nc.Env["DOCKER_CONFIG"], "config.json")
			err := os.WriteFile(dest, []byte(customDCConfig), 0o400)
			assert.NilError(nc.T(), err, "failed to write custom docker config json file for test")
			nc.hasWrittenDockerConfig = true
		}
	}

	trgt := getTarget()
	switch trgt {
	case targetDocker:
		// Allow debugging with docker syntax
		if nc.Config.Read(Debug) != "" {
			nc.PrependArgs("--log-level=debug")
		}
	case targetNerdishctl, targetNerdctl:
		// Set the namespace
		if nc.Config.Read(Namespace) != "" {
			nc.PrependArgs("--namespace=" + string(nc.Config.Read(Namespace)))
		}

		envPrefix := strings.ToUpper(trgt)
		// If no PREFIX_TOML was explicitly provided, set it to the private dir
		if nc.Env[envPrefix+"_TOML"] == "" {
			nc.Env[envPrefix+"_TOML"] = filepath.Join(nc.TempDir, trgt+".toml")
		}

		// If we have custom toml content, write it if it does not exist already
		if nc.Config.Read(NerdishctlToml) != "" {
			if !nc.hasWrittenToml {
				dest := nc.Env[envPrefix+"_TOML"]
				err := os.WriteFile(dest, []byte(nc.Config.Read(NerdishctlToml)), 0o400)
				assert.NilError(nc.T(), err, "failed to write cli toml config file")
				nc.hasWrittenToml = true
			}
		}

		if nc.Config.Read(HostsDir) != "" {
			nc.PrependArgs("--hosts-dir=" + string(nc.Config.Read(HostsDir)))
		}
		if nc.Config.Read(DataRoot) != "" {
			nc.PrependArgs("--data-root=" + string(nc.Config.Read(DataRoot)))
		}
		if nc.Config.Read(Debug) != "" {
			nc.PrependArgs("--debug-full")
		}
	}
}

func (nc *nerdCommand) Clone() test.TestableCommand {
	return &nerdCommand{
		GenericCommand:         *(nc.GenericCommand.Clone().(*test.GenericCommand)),
		hasWrittenToml:         nc.hasWrittenToml,
		hasWrittenDockerConfig: nc.hasWrittenDockerConfig,
	}
}
