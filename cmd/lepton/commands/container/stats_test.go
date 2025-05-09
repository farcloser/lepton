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
	"runtime"
	"testing"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestStats(t *testing.T) {
	testCase := nerdtest.Setup()

	// FIXME: does not seem to work on windows
	testCase.Require = require.Not(require.Windows)

	if runtime.GOOS == "linux" {
		// this comment is for `ps` but it also valid for `stats` :
		// https://github.com/containerd/nerdctl/pull/223#issuecomment-851395178
		testCase.Require = require.All(
			testCase.Require,
			nerdtest.CgroupsAccessible,
		)
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier("container"))
		helpers.Anyhow("rm", "-f", data.Identifier("memlimited"))
		helpers.Anyhow("rm", "-f", data.Identifier("exited"))
	}

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		helpers.Ensure(
			"run",
			"--quiet",
			"-d",
			"--name",
			data.Identifier("container"),
			testutil.CommonImage,
			"sleep",
			nerdtest.Infinity,
		)
		helpers.Ensure(
			"run",
			"--quiet",
			"-d",
			"--name",
			data.Identifier("memlimited"),
			"--memory",
			"1g",
			testutil.CommonImage,
			"sleep",
			nerdtest.Infinity,
		)
		helpers.Ensure(
			"run",
			"--quiet",
			"--name",
			data.Identifier("exited"),
			testutil.CommonImage,
			"echo",
			"'exited'",
		)
		data.Set("id", data.Identifier("container"))
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "stats",
			Command:     test.Command("stats", "--no-stream", "--no-trunc"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Contains(data.Get("id")),
				}
			},
		},
		{
			Description: "container stats",
			Command:     test.Command("container", "stats", "--no-stream", "--no-trunc"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Contains(data.Get("id")),
				}
			},
		},
		{
			Description: "stats ID",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("stats", "--no-stream", data.Get("id"))
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
		},
		{
			Description: "container stats ID",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("container", "stats", "--no-stream", data.Get("id"))
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
		},
		{
			Description: "no mem limit set",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("stats", "--no-stream")
			},
			// https://github.com/containerd/nerdctl/issues/1240
			// used to print UINT64_MAX as the memory limit, so, ensure it does no more
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.DoesNotContain("16EiB")),
		},
		{
			Description: "mem limit set",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("stats", "--no-stream")
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("1GiB")),
		},
	}

	testCase.Run(t)
}
