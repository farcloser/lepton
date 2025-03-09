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

package container

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest/registry"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
	"go.farcloser.world/lepton/pkg/testutil/various"
)

func testMultiPlatformRun(base *testutil.Base, alpineImage string) {
	t := base.T
	testutil.RequireExecPlatform(t, "linux/amd64", "linux/arm64")
	testCases := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}
	for plat, expectedUnameM := range testCases {
		t.Logf("Testing %q (%q)", plat, expectedUnameM)
		cmd := base.Cmd("run", "--rm", "--platform="+plat, alpineImage, "uname", "-m")
		cmd.AssertOutExactly(expectedUnameM + "\n")
	}
}

func TestMultiPlatformRun(t *testing.T) {
	base := testutil.NewBase(t)
	testMultiPlatformRun(base, testutil.AlpineImage)
}

func TestMultiPlatformBuildPush(t *testing.T) {
	testCase := nerdtest.Setup()

	// non-buildx version of `docker build` lacks multi-platform. Also, `docker push` lacks --platform.
	testCase.Require = require.All(
		require.Not(nerdtest.Docker),
		nerdtest.Registry,
		nerdtest.Build,
		nerdtest.IsFlaky("mixed tests using both legacy and non-legacy are considered flaky"),
	)

	var reg *registry.Server

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		testutil.RequireExecPlatform(t, "linux/amd64", "linux/arm64")
		base := testutil.NewBase(t)
		tID := data.Identifier()
		reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
		reg.Setup(data, helpers)

		data.Set("imageName", fmt.Sprintf("localhost:%d/%s:latest", reg.Port, tID))

		dockerfile := fmt.Sprintf(`FROM %s
RUN echo dummy
	`, testutil.AlpineImage)

		buildCtx := various.CreateBuildContext(t, dockerfile)

		base.Cmd("build", "-t", data.Get("imageName"), "--platform=amd64,arm64", buildCtx).AssertOK()
		testMultiPlatformRun(base, data.Get("imageName"))
		base.Cmd("push", "--platform=amd64,arm64", data.Get("imageName")).AssertOK()
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rmi", data.Get("imageName"))
		if reg != nil {
			reg.Cleanup(data, helpers)
		}
	}

	testCase.Run(t)
}

// TestMultiPlatformBuildPushNoRun tests if the push succeeds in a situation where we build
// a Dockerfile without RUN, COPY, etc. commands. In such situation, BuildKit doesn't download the base image
// so we need to ensure these blobs to be locally available.
func TestMultiPlatformBuildPushNoRun(t *testing.T) {
	testCase := nerdtest.Setup()

	// non-buildx version of `docker build` lacks multi-platform. Also, `docker push` lacks --platform.
	testCase.Require = require.All(
		require.Not(nerdtest.Docker),
		nerdtest.Registry,
		nerdtest.Build,
		nerdtest.IsFlaky("mixed tests using both legacy and non-legacy are considered flaky"),
	)

	var reg *registry.Server

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
		reg.Setup(data, helpers)

		testutil.RequireExecPlatform(t, "linux/amd64", "linux/arm64")
		base := testutil.NewBase(t)
		tID := data.Identifier()

		imageName := fmt.Sprintf("localhost:%d/%s:latest", reg.Port, tID)
		defer base.Cmd("rmi", imageName).Run()

		dockerfile := fmt.Sprintf(`FROM %s
CMD echo dummy
	`, testutil.AlpineImage)

		buildCtx := various.CreateBuildContext(t, dockerfile)

		base.Cmd("build", "-t", imageName, "--platform=amd64,arm64", buildCtx).AssertOK()
		testMultiPlatformRun(base, imageName)
		base.Cmd("push", "--platform=amd64,arm64", imageName).AssertOK()
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		if reg != nil {
			reg.Cleanup(data, helpers)
		}
	}

	testCase.Run(t)
}

func TestMultiPlatformPullPushAllPlatforms(t *testing.T) {
	testCase := nerdtest.Setup()

	var reg *registry.Server

	// non-buildx version of `docker build` lacks multi-platform. Also, `docker push` lacks --platform.
	testCase.Require = require.All(
		nerdtest.Registry,
		require.Not(nerdtest.Docker),
		nerdtest.IsFlaky("mixed tests using both legacy and non-legacy are considered flaky"),
	)

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
		reg.Setup(data, helpers)

		base := testutil.NewBase(t)
		tID := data.Identifier()

		pushImageName := fmt.Sprintf("localhost:%d/%s:latest", reg.Port, tID)
		defer base.Cmd("rmi", pushImageName).Run()

		base.Cmd("pull", "--quiet", "--all-platforms", testutil.AlpineImage).AssertOK()
		base.Cmd("tag", testutil.AlpineImage, pushImageName).AssertOK()
		base.Cmd("push", "--all-platforms", pushImageName).AssertOK()
		testMultiPlatformRun(base, pushImageName)
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		if reg != nil {
			reg.Cleanup(data, helpers)
		}
	}

	testCase.Run(t)
}

func TestMultiPlatformComposeUpBuild(t *testing.T) {
	testutil.DockerIncompatible(t)
	testutil.RequiresBuild(t)
	testutil.RegisterBuildCacheCleanup(t)
	testutil.RequireExecPlatform(t, "linux/amd64", "linux/arm64")
	base := testutil.NewBase(t)

	const dockerComposeYAML = `
services:
  svc0:
    build: .
    platform: amd64
    ports:
    - 8080:80
  svc1:
    build: .
    platform: arm64
    ports:
    - 8081:80
`
	dockerfile := fmt.Sprintf(`FROM %s
RUN uname -m > /usr/share/nginx/html/index.html
`, testutil.NginxAlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	comp.WriteFile("Dockerfile", dockerfile)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d", "--build").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").Run()

	testCases := map[string]string{
		"http://127.0.0.1:8080": "x86_64",
		"http://127.0.0.1:8081": "aarch64",
	}

	for testURL, expectedIndexHTML := range testCases {
		resp, err := nettestutil.HTTPGet(testURL, 50, false)
		assert.NilError(t, err)
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		assert.NilError(t, err)
		t.Logf("respBody=%q", respBody)
		assert.Assert(t, strings.Contains(string(respBody), expectedIndexHTML))
	}
}
