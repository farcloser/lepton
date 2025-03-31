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

func TestTop(t *testing.T) {
	testCase := nerdtest.Setup()

	// more details https://github.com/containerd/nerdctl/pull/223#issuecomment-851395178
	if runtime.GOOS == "linux" {
		testCase.Require = nerdtest.CgroupsAccessible
	}

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		// FIXME: busybox 1.36 on windows still appears to not support sleep inf. Unclear why.
		helpers.Ensure(
			"run",
			"--quiet",
			"-d",
			"--name",
			data.Identifier(),
			testutil.CommonImage,
			"sleep",
			nerdtest.Infinity,
		)
		data.Set("cID", data.Identifier())
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier())
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "with o pid,user,cmd",
			// Docker does not support top -o
			Require: require.Not(nerdtest.Docker),
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("top", data.Get("cID"), "-o", "pid,user,cmd")
			},

			Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
		},
		{
			Description: "simple",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("top", data.Get("cID"))
			},

			Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
		},
	}

	testCase.Run(t)
}

func TestTopHyperVContainer(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = nerdtest.HyperV

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		// FIXME: busybox 1.36 on windows still appears to not support sleep inf. Unclear why.
		helpers.Ensure(
			"run",
			"--quiet",
			"--isolation",
			"hyperv",
			"-d",
			"--name",
			data.Identifier("container"),
			testutil.CommonImage,
			"sleep",
			nerdtest.Infinity,
		)
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier("container"))
	}

	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		return helpers.Command("top", data.Identifier("container"))
	}

	testCase.Expected = test.Expects(expect.ExitCodeSuccess, nil, nil)

	testCase.Run(t)
}
