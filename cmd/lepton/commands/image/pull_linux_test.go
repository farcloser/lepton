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

package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest/registry"
	"go.farcloser.world/lepton/pkg/testutil/testregistry"
	"go.farcloser.world/lepton/pkg/testutil/various"
)

func TestImagePullWithCosign(t *testing.T) {
	nerdtest.Setup()

	var reg *registry.Server
	var keyPair *various.CosignKeyPair

	testCase := &test.Case{
		Require: require.All(
			require.Linux,
			nerdtest.Build,
			nerdtest.Registry,
			require.Binary("cosign"),
			require.Not(nerdtest.Docker),
		),
		Env: map[string]string{
			"COSIGN_PASSWORD": "1",
		},
		Setup: func(data test.Data, helpers test.Helpers) {
			reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
			reg.Setup(data, helpers)

			keyPair = various.NewCosignKeyPair(t, "cosign-key-pair", "1")
			testImageRef := fmt.Sprintf("%s:%d/%s", "127.0.0.1", reg.Port, data.Identifier())
			dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "build-test-string"]
	`, testutil.CommonImage)

			buildCtx := data.TempDir()
			err := os.WriteFile(filepath.Join(buildCtx, "Dockerfile"), []byte(dockerfile), 0o600)
			assert.NilError(helpers.T(), err)
			helpers.Ensure("build", "-t", testImageRef+":one", buildCtx)
			helpers.Ensure("build", "-t", testImageRef+":two", buildCtx)
			helpers.Ensure("push", "--sign=cosign", "--cosign-key="+keyPair.PrivateKey, testImageRef+":one")
			helpers.Ensure("push", "--sign=cosign", "--cosign-key="+keyPair.PrivateKey, testImageRef+":two")
			helpers.Ensure("rmi", "-f", testImageRef)
			data.Set("imageref", testImageRef)
		},
		Cleanup: func(data test.Data, helpers test.Helpers) {
			if keyPair != nil {
				keyPair.Cleanup()
			}
			if reg != nil {
				reg.Cleanup(data, helpers)
				testImageRef := fmt.Sprintf("%s:%d/%s", "127.0.0.1", reg.Port, data.Identifier())
				helpers.Anyhow("rmi", "-f", testImageRef+":one")
				helpers.Anyhow("rmi", "-f", testImageRef+":two")
			}
		},
		SubTests: []*test.Case{
			{
				Description: "Pull with the correct key",
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("pull", "--quiet", "--verify=cosign", "--cosign-key="+keyPair.PublicKey, data.Get("imageref")+":one")
				},
				Expected: test.Expects(0, nil, nil),
			},
			{
				Description: "Pull with unrelated key",
				Env: map[string]string{
					"COSIGN_PASSWORD": "2",
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					newKeyPair := various.NewCosignKeyPair(t, "cosign-key-pair-test", "2")
					return helpers.Command("pull", "--quiet", "--verify=cosign", "--cosign-key="+newKeyPair.PublicKey, data.Get("imageref")+":two")
				},
				Expected: test.Expects(12, nil, nil),
			},
		},
	}

	testCase.Run(t)
}

func TestImagePullPlainHttpWithDefaultPort(t *testing.T) {
	nerdtest.Setup()

	var reg *testregistry.RegistryServer

	testCase := &test.Case{
		Require: require.All(
			require.Linux,
			require.Not(nerdtest.Docker),
			nerdtest.Build,
		),
		Setup: func(data test.Data, helpers test.Helpers) {
			reg = testregistry.NewWithNoAuth(data, helpers, 80, false)
			testImageRef := fmt.Sprintf("%s/%s:%s",
				reg.IP.String(), data.Identifier(), strings.Split(testutil.CommonImage, ":")[1])
			dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "build-test-string"]
	`, testutil.CommonImage)

			buildCtx := data.TempDir()
			err := os.WriteFile(filepath.Join(buildCtx, "Dockerfile"), []byte(dockerfile), 0o600)
			assert.NilError(helpers.T(), err)
			helpers.Ensure("build", "-t", testImageRef, buildCtx)
			helpers.Ensure("--insecure-registry", "push", testImageRef)
			helpers.Ensure("rmi", "-f", testImageRef)
		},
		Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
			testImageRef := fmt.Sprintf("%s/%s:%s",
				reg.IP.String(), data.Identifier(), strings.Split(testutil.CommonImage, ":")[1])
			return helpers.Command("--insecure-registry", "pull", testImageRef)
		},
		Expected: test.Expects(0, nil, nil),
		Cleanup: func(data test.Data, helpers test.Helpers) {
			if reg != nil {
				reg.Cleanup(nil)
				testImageRef := fmt.Sprintf("%s/%s:%s",
					reg.IP.String(), data.Identifier(), strings.Split(testutil.CommonImage, ":")[1])
				helpers.Anyhow("rmi", "-f", testImageRef)
			}
		},
	}

	testCase.Run(t)
}

func TestImagePullSoci(t *testing.T) {
	nerdtest.Setup()

	testCase := &test.Case{
		Require: require.All(
			require.Linux,
			require.Not(nerdtest.Docker),
			nerdtest.Soci,
		),

		// NOTE: these tests cannot be run in parallel, as they depend on the output of host `mount`
		// They also feel prone to raciness...
		SubTests: []*test.Case{
			{
				Description: "Run without specifying SOCI index",
				NoParallel:  true,
				Data: test.WithData("remoteSnapshotsExpectedCount", "11").
					Set("sociIndexDigest", ""),
				Setup: func(data test.Data, helpers test.Helpers) {
					cmd := helpers.Custom("mount")
					cmd.Run(&test.Expected{
						Output: func(stdout string, info string, t *testing.T) {
							data.Set("remoteSnapshotsInitialCount", strconv.Itoa(strings.Count(stdout, "fuse.rawBridge")))
						},
					})
					helpers.Ensure("--snapshotter=soci", "pull", testutil.FfmpegSociImage)
				},
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rmi", "-f", testutil.FfmpegSociImage)
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Custom("mount")
				},
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: func(stdout string, info string, t *testing.T) {
							remoteSnapshotsInitialCount, _ := strconv.Atoi(data.Get("remoteSnapshotsInitialCount"))
							remoteSnapshotsActualCount := strings.Count(stdout, "fuse.rawBridge")
							assert.Equal(t,
								data.Get("remoteSnapshotsExpectedCount"),
								strconv.Itoa(remoteSnapshotsActualCount-remoteSnapshotsInitialCount),
								info)
						},
					}
				},
			},
			{
				Description: "Run with bad SOCI index",
				NoParallel:  true,
				Data: test.WithData("remoteSnapshotsExpectedCount", "11").
					Set("sociIndexDigest", "sha256:thisisabadindex0000000000000000000000000000000000000000000000000"),
				Setup: func(data test.Data, helpers test.Helpers) {
					cmd := helpers.Custom("mount")
					cmd.Run(&test.Expected{
						Output: func(stdout string, info string, t *testing.T) {
							data.Set("remoteSnapshotsInitialCount", strconv.Itoa(strings.Count(stdout, "fuse.rawBridge")))
						},
					})
					helpers.Ensure("--snapshotter=soci", "pull", testutil.FfmpegSociImage)
				},
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rmi", "-f", testutil.FfmpegSociImage)
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Custom("mount")
				},
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: func(stdout string, info string, t *testing.T) {
							remoteSnapshotsInitialCount, _ := strconv.Atoi(data.Get("remoteSnapshotsInitialCount"))
							remoteSnapshotsActualCount := strings.Count(stdout, "fuse.rawBridge")
							assert.Equal(t,
								data.Get("remoteSnapshotsExpectedCount"),
								strconv.Itoa(remoteSnapshotsActualCount-remoteSnapshotsInitialCount),
								info)
						},
					}
				},
			},
		},
	}

	testCase.Run(t)
}

func TestImagePullProcessOutput(t *testing.T) {
	nerdtest.Setup()

	testCase := &test.Case{
		SubTests: []*test.Case{
			{
				Description: "Pull Image - output should be in stdout",
				NoParallel:  true,
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rmi", "-f", testutil.BusyboxImage)
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("pull", testutil.BusyboxImage)
				},
				Expected: test.Expects(0, nil, expect.Contains(testutil.BusyboxImage)),
			},
			{
				Description: "Run Container with image pull - output should be in stderr",
				NoParallel:  true,
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rmi", "-f", testutil.BusyboxImage)
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("run", "--rm", testutil.BusyboxImage)
				},
				Expected: test.Expects(0, nil, expect.DoesNotContain(testutil.BusyboxImage)),
			},
		},
	}

	testCase.Run(t)
}
