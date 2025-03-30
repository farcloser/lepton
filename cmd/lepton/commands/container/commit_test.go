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
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestCommit(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.SubTests = []*test.Case{
		{
			Description: "with pause",
			Require:     nerdtest.CGroup,
			Cleanup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				helpers.Anyhow("rm", "-f", identifier)
				helpers.Anyhow("rmi", "-f", identifier)
			},
			Setup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				helpers.Ensure(
					"run",
					"--quiet",
					"-d",
					"--name",
					identifier,
					testutil.CommonImage,
					"sleep",
					nerdtest.Infinity,
				)
				helpers.Ensure("exec", identifier, "sh", "-euxc", `echo hello-test-commit > /foo`)
			},
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				identifier := data.Identifier()
				helpers.Ensure(
					"commit",
					"-c", `CMD ["/foo"]`,
					"-c", `ENTRYPOINT ["cat"]`,
					"--pause=true",
					identifier, identifier)
				return helpers.Command("run", "--rm", identifier)
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Equals("hello-test-commit\n")),
		},
		{
			Description: "no pause",
			Require:     require.Not(require.Windows),
			Cleanup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				helpers.Anyhow("rm", "-f", identifier)
				helpers.Anyhow("rmi", "-f", identifier)
			},
			Setup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				helpers.Ensure(
					"run",
					"--quiet",
					"-d",
					"--name",
					identifier,
					testutil.CommonImage,
					"sleep",
					nerdtest.Infinity,
				)
				nerdtest.EnsureContainerStarted(helpers, identifier)
				helpers.Ensure("exec", identifier, "sh", "-euxc", `echo hello-test-commit > /foo`)
			},
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				identifier := data.Identifier()
				helpers.Ensure(
					"commit",
					"-c", `CMD ["/foo"]`,
					"-c", `ENTRYPOINT ["cat"]`,
					"--pause=false",
					identifier, identifier)
				return helpers.Command("run", "--rm", identifier)
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Equals("hello-test-commit\n")),
		},
	}

	testCase.Run(t)
}
