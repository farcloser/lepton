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

package compose_test

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/containerd/log"
	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest/registry"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
)

func TestComposeRun(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - stty
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	const sttyPartialOutput = "speed 38400 baud"
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
		"run", "--name", containerName, "alpine").AssertOutContains(sttyPartialOutput)
}

func TestComposeRunWithRM(t *testing.T) {
	// Test does not make sense. Image may or may not be there.
	// t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - stty
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	const sttyPartialOutput = "speed 38400 baud"
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
		"run", "--name", containerName, "--rm", "alpine").AssertOutContains(sttyPartialOutput)

	psCmd := base.Cmd("ps", "-a", "--format=\"{{.Names}}\"")
	result := psCmd.Run()
	stdoutContent := result.Stdout() + result.Stderr()
	assert.Assert(psCmd.T, result.ExitCode == 0, stdoutContent)
	if strings.Contains(stdoutContent, containerName) {
		log.L.Errorf("test failed, the container %s is not removed", stdoutContent)
		t.Fail()
		return
	}
}

func TestComposeRunWithServicePorts(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  web:
    image: %s
    ports:
      - 8090:80
`, testutil.NginxAlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	go func() {
		// unbuffer(1) emulates tty, which is required by `run -t`.
		// unbuffer(1) can be installed with `apt-get install expect`.
		unbuffer := []string{"unbuffer"}
		base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
			"run", "--service-ports", "--name", containerName, "web").Run()
	}()

	checkNginx := func() error {
		resp, err := nettestutil.HTTPGet("http://127.0.0.1:8090", 10, false)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(respBody), testutil.NginxAlpineIndexHTMLSnippet) {
			t.Logf("respBody=%q", respBody)
			return fmt.Errorf("respBody does not contain %q", testutil.NginxAlpineIndexHTMLSnippet)
		}
		return nil
	}
	var nginxWorking bool
	for i := range 30 {
		t.Logf("(retry %d)", i)
		err := checkNginx()
		if err == nil {
			nginxWorking = true
			break
		}
		t.Log(err)
		time.Sleep(3 * time.Second)
	}
	if !nginxWorking {
		t.Fatal("nginx is not working")
	}
	t.Log("nginx seems functional")
}

func TestComposeRunWithPublish(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  web:
    image: %s
`, testutil.NginxAlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	go func() {
		// unbuffer(1) emulates tty, which is required by `run -t`.
		// unbuffer(1) can be installed with `apt-get install expect`.
		unbuffer := []string{"unbuffer"}
		base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
			"run", "--publish", "8091:80", "--name", containerName, "web").Run()
	}()

	checkNginx := func() error {
		resp, err := nettestutil.HTTPGet("http://127.0.0.1:8091", 10, false)
		if err != nil {
			return err
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(respBody), testutil.NginxAlpineIndexHTMLSnippet) {
			t.Logf("respBody=%q", respBody)
			return fmt.Errorf("respBody does not contain %q", testutil.NginxAlpineIndexHTMLSnippet)
		}
		return nil
	}
	var nginxWorking bool
	for i := range 30 {
		t.Logf("(retry %d)", i)
		err := checkNginx()
		if err == nil {
			nginxWorking = true
			break
		}
		t.Log(err)
		time.Sleep(3 * time.Second)
	}
	if !nginxWorking {
		t.Fatal("nginx is not working")
	}
	t.Log("nginx seems functional")
}

func TestComposeRunWithEnv(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - sh
      - -c
      - "echo $$FOO"
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	const partialOutput = "bar"
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
		"run", "-e", "FOO=bar", "--name", containerName, "alpine").AssertOutContains(partialOutput)
}

func TestComposeRunWithUser(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - id
      - -u
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	const partialOutput = "5000"
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
		"run", "--user", "5000", "--name", containerName, "alpine").AssertOutContains(partialOutput)
}

func TestComposeRunWithLabel(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - echo
      - "dummy log"
    labels:
      - "foo=bar"
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
		"run", "--label", "foo=rab", "--label", "x=y", "--name", containerName, "alpine").AssertOK()

	container := base.InspectContainer(containerName)
	if container.Config == nil {
		log.L.Errorf("test failed, cannot fetch container config")
		t.Fail()
	}
	assert.Equal(t, container.Config.Labels["foo"], "rab")
	assert.Equal(t, container.Config.Labels["x"], "y")
}

func TestComposeRunWithArgs(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - echo
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	const partialOutput = "hello world"
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
		"run", "--name", containerName, "alpine", partialOutput).AssertOutContains(partialOutput)
}

func TestComposeRunWithEntrypoint(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	// specify the name of container in order to remove
	// TODO: when `compose rm` is implemented, replace it.
	containerName := testutil.Identifier(t)

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
      - stty # should be changed
`, testutil.AlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	defer base.Cmd("rm", "-f", "-v", containerName).Run()
	const partialOutput = "hello world"
	// unbuffer(1) emulates tty, which is required by `run -t`.
	// unbuffer(1) can be installed with `apt-get install expect`.
	unbuffer := []string{"unbuffer"}
	base.ComposeCmdWithHelper(
		unbuffer,
		"-f",
		comp.YAMLFullPath(),
		"run",
		"--entrypoint",
		"echo",
		"--name",
		containerName,
		"alpine",
		partialOutput,
	).AssertOutContains(partialOutput)
}

func TestComposeRunWithVolume(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		base := testutil.NewBase(t)
		containerName := testutil.Identifier(t)

		dockerComposeYAML := fmt.Sprintf(`
version: '3.1'
services:
  alpine:
    image: %s
    entrypoint:
    - stty # no meaning, just put any command
`, testutil.AlpineImage)

		comp := testutil.NewComposeDir(t, dockerComposeYAML)
		defer comp.CleanUp()
		projectName := comp.ProjectName()
		t.Logf("projectName=%q", projectName)
		defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

		// The directory is automatically removed by Cleanup
		tmpDir := t.TempDir()
		destinationDir := "/data"
		volumeFlagStr := fmt.Sprintf("%s:%s", tmpDir, destinationDir)

		defer base.Cmd("rm", "-f", "-v", containerName).Run()
		// unbuffer(1) emulates tty, which is required by `run -t`.
		// unbuffer(1) can be installed with `apt-get install expect`.
		unbuffer := []string{"unbuffer"}
		base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(),
			"run", "--volume", volumeFlagStr, "--name", containerName, "alpine").AssertOK()

		container := base.InspectContainer(containerName)
		errMsg := fmt.Sprintf("test failed, cannot find volume: %v", container.Mounts)
		assert.Assert(t, container.Mounts != nil, errMsg)
		assert.Assert(t, len(container.Mounts) == 1, errMsg)
		assert.Assert(t, container.Mounts[0].Source == tmpDir, errMsg)
		assert.Assert(t, container.Mounts[0].Destination == destinationDir, errMsg)
	}
	testCase.Run(t)
}

func TestComposePushAndPullWithCosignVerify(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = require.All(
		require.Binary("cosign"),
		nerdtest.Build,
		nerdtest.Registry,
		require.Not(nerdtest.Docker),
	)

	var reg *registry.Server

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		if reg != nil {
			reg.Cleanup(data, helpers)
		}
	}

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
		reg.Setup(data, helpers)

		base := testutil.NewBase(t)
		base.Env = append(base.Env, "COSIGN_PASSWORD=1")

		keyPair := nerdtest.NewCosignKeyPair(data, helpers)

		tID := testutil.Identifier(t)
		testImageRefPrefix := fmt.Sprintf("127.0.0.1:%d/%s/", reg.Port, tID)

		var (
			imageSvc0 = testImageRefPrefix + "composebuild_svc0"
			imageSvc1 = testImageRefPrefix + "composebuild_svc1"
			imageSvc2 = testImageRefPrefix + "composebuild_svc2"
		)

		dockerComposeYAML := fmt.Sprintf(`
services:
  svc0:
    build: .
    image: %s
    x-nerdctl-verify: cosign
    x-nerdctl-cosign-public-key: %s
    x-nerdctl-sign: cosign
    x-nerdctl-cosign-private-key: %s
    entrypoint:
      - stty
  svc1:
    build: .
    image: %s
    x-nerdctl-verify: cosign
    x-nerdctl-cosign-public-key: dummy_pub_key
    x-nerdctl-sign: cosign
    x-nerdctl-cosign-private-key: %s
    entrypoint:
      - stty
  svc2:
    build: .
    image: %s
    x-nerdctl-verify: none
    x-nerdctl-sign: none
    entrypoint:
      - stty
`, imageSvc0, keyPair.PublicKey, keyPair.PrivateKey,
			imageSvc1, keyPair.PrivateKey, imageSvc2)

		dockerfile := "FROM " + testutil.AlpineImage

		comp := testutil.NewComposeDir(t, dockerComposeYAML)
		defer comp.CleanUp()
		comp.WriteFile("Dockerfile", dockerfile)

		projectName := comp.ProjectName()
		t.Logf("projectName=%q", projectName)
		defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

		// 1. build both services/images
		base.ComposeCmd("-f", comp.YAMLFullPath(), "build").AssertOK()
		// 2. compose push with cosign for svc0/svc1, (and none for svc2)
		base.ComposeCmd("-f", comp.YAMLFullPath(), "push").AssertOK()
		// 3. compose pull with cosign
		base.ComposeCmd("-f", comp.YAMLFullPath(), "pull", "svc0").AssertOK()   // key match
		base.ComposeCmd("-f", comp.YAMLFullPath(), "pull", "svc1").AssertFail() // key mismatch
		base.ComposeCmd("-f", comp.YAMLFullPath(), "pull", "svc2").AssertOK()   // verify passed
		// 4. compose run
		const sttyPartialOutput = "speed 38400 baud"
		// unbuffer(1) emulates tty, which is required by `run -t`.
		// unbuffer(1) can be installed with `apt-get install expect`.
		unbuffer := []string{"unbuffer"}
		base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(), "run", "svc0").
			AssertOutContains(sttyPartialOutput)
			// key match
		base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(), "run", "svc1").
			AssertFail()
			// key mismatch
		base.ComposeCmdWithHelper(unbuffer, "-f", comp.YAMLFullPath(), "run", "svc2").
			AssertOutContains(sttyPartialOutput)
			// verify passed
		// 5. compose up
		base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "svc0").AssertOK()   // key match
		base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "svc1").AssertFail() // key mismatch
		base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "svc2").AssertOK()   // verify passed
	}
	testCase.Run(t)
}
