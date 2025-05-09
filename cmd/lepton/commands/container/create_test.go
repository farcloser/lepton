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

package container_test

import (
	"encoding/json"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/inspecttypes/dockercompat"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestCreate(t *testing.T) {
	testCase := nerdtest.Setup()
	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		helpers.Ensure(
			"create",
			"--quiet",
			"--name",
			data.Identifier("container"),
			testutil.CommonImage,
			"echo",
			"foo",
		)
		data.Set("cID", data.Identifier("container"))
	}
	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier("container"))
	}

	testCase.Require = nerdtest.IsFlaky("https://github.com/containerd/nerdctl/issues/3717")

	testCase.SubTests = []*test.Case{
		{
			Description: "ps -a",
			NoParallel:  true,
			Command:     test.Command("ps", "-a"),
			// FIXME: this might get a false positive if other tests have created a container
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("Created")),
		},
		{
			Description: "start",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("start", data.Get("cID"))
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
		},
		{
			Description: "logs",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("logs", data.Get("cID"))
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("foo")),
		},
	}

	testCase.Run(t)
}

func TestCreateHyperVContainer(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = nerdtest.HyperV

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		helpers.Ensure(
			"create",
			"--quiet",
			"--isolation",
			"hyperv",
			"--name",
			data.Identifier("container"),
			testutil.CommonImage,
			"echo",
			"foo",
		)
		data.Set("cID", data.Identifier("container"))
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier("container"))
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "ps -a",
			NoParallel:  true,
			Command:     test.Command("ps", "-a"),
			// FIXME: this might get a false positive if other tests have created a container
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("Created")),
		},
		{
			Description: "start",
			NoParallel:  true,
			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("start", data.Get("cID"))
				ran := false
				for i := 0; i < 10 && !ran; i++ {
					helpers.Command("container", "inspect", data.Get("cID")).
						Run(&test.Expected{
							ExitCode: expect.ExitCodeNoCheck,
							Output: func(stdout, info string, t *testing.T) {
								var dc []dockercompat.Container
								err := json.Unmarshal([]byte(stdout), &dc)
								if err != nil || len(dc) == 0 {
									return
								}
								assert.Equal(
									t,
									len(dc),
									1,
									"Unexpectedly got multiple results\n"+info,
								)
								ran = dc[0].State.Status == "exited"
							},
						})
					time.Sleep(time.Second)
				}
				assert.Assert(t, ran, "container did not ran after 10 seconds")
			},
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("logs", data.Get("cID"))
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("foo")),
		},
	}

	testCase.Run(t)
}
