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
	"testing"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestRemoveContainer(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
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
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier())
	}

	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		containerID := data.Identifier()
		helpers.Fail("rm", containerID)

		// FIXME: should (re-)evaluate this
		// `kill` seems to return before the container actually stops
		helpers.Ensure("stop", containerID)

		return helpers.Command("rm", containerID)
	}

	testCase.Expected = test.Expects(expect.ExitCodeSuccess, nil, nil)
}
