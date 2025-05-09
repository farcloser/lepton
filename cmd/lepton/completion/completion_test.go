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

package completion_test

import (
	"testing"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestMain(m *testing.M) {
	testutil.M(m)
}

func TestCompletion(t *testing.T) {
	nerdtest.Setup()

	testCase := &test.Case{
		Require: require.Not(nerdtest.Docker),
		Setup: func(data test.Data, helpers test.Helpers) {
			identifier := data.Identifier()
			helpers.Ensure("pull", "--quiet", testutil.CommonImage)
			helpers.Ensure("network", "create", identifier)
			helpers.Ensure("volume", "create", identifier)
			data.Set("identifier", identifier)
		},
		Cleanup: func(data test.Data, helpers test.Helpers) {
			identifier := data.Identifier()
			helpers.Anyhow("network", "rm", identifier)
			helpers.Anyhow("volume", "rm", identifier)
		},
		SubTests: []*test.Case{
			// Namespace commands
			{
				Description: "namespace",
				Require:     require.Not(require.Windows),
				Command:     test.Command("__complete", "namespace", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("inspect")),
			},
			{
				Description: "namespace",
				Require:     require.Not(require.Windows),
				Command:     test.Command("__complete", "namespace", "inspect", "--format"),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains(formatter.FormatJSON)),
			},
			{
				Description: "namespace",
				Require:     require.Not(require.Windows),
				Command:     test.Command("__complete", "namespace", "inspect", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("cli-test")),
			},
			// Others
			{
				Description: "--cgroup-manager",
				Require:     require.Not(require.Windows),
				Command:     test.Command("__complete", "--cgroup-manager", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("systemd\n")),
			},
			{
				Description: "--snapshotter",
				Require:     require.Not(require.Windows),
				Command:     test.Command("__complete", "--snapshotter", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("native\n")),
			},
			{
				Description: "empty",
				Command:     test.Command("__complete", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("run\t")),
			},
			{
				Description: "build --network",
				Command:     test.Command("__complete", "build", "--network", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("default\n")),
			},
			{
				Description: "run -",
				Command:     test.Command("__complete", "run", "-"),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("--network\t")),
			},
			{
				Description: "run --n",
				Command:     test.Command("__complete", "run", "--n"),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("--network\t")),
			},
			{
				Description: "run --ne",
				Command:     test.Command("__complete", "run", "--ne"),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("--network\t")),
			},
			{
				Description: "run --net",
				Command:     test.Command("__complete", "run", "--net", ""),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: expect.All(
							expect.Contains("host\n"),
							expect.Contains(data.Get("identifier")+"\n"),
						),
					}
				},
			},
			{
				Description: "run -it --net",
				Command:     test.Command("__complete", "run", "-it", "--net", ""),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: expect.All(
							expect.Contains("host\n"),
							expect.Contains(data.Get("identifier")+"\n"),
						),
					}
				},
			},
			{
				Description: "run -ti --rm --net",
				Command:     test.Command("__complete", "run", "-it", "--rm", "--net", ""),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: expect.All(
							expect.Contains("host\n"),
							expect.Contains(data.Get("identifier")+"\n"),
						),
					}
				},
			},
			{
				Description: "run --restart",
				Command:     test.Command("__complete", "run", "--restart", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("always\n")),
			},
			{
				Description: "network --rm",
				Command:     test.Command("__complete", "network", "rm", ""),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: expect.All(
							expect.DoesNotContain("host\n"),
							expect.Contains(data.Get("identifier")+"\n"),
						),
					}
				},
			},
			{
				Description: "run --cap-add",
				Require:     require.Not(require.Windows),
				Command:     test.Command("__complete", "run", "--cap-add", ""),
				Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.All(
					expect.Contains("sys_admin\n"),
					expect.DoesNotContain("CAP_SYS_ADMIN\n"),
				)),
			},
			{
				Description: "volume inspect",
				Command:     test.Command("__complete", "volume", "inspect", ""),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: expect.Contains(data.Get("identifier") + "\n"),
					}
				},
			},
			{
				Description: "volume rm",
				Command:     test.Command("__complete", "volume", "rm", ""),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						Output: expect.Contains(data.Get("identifier") + "\n"),
					}
				},
			},
			{
				Description: "no namespace --cgroup-manager",
				Require:     require.Not(require.Windows),
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Custom(nerdtest.Binary(), "__complete", "--cgroup-manager", "")
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("systemd\n")),
			},
			{
				Description: "no namespace empty",
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Custom(nerdtest.Binary(), "__complete", "")
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("run\t")),
			},
			{
				Description: "namespace space empty",
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					// mind {"--namespace=test"} vs {"--namespace", "test"}
					return helpers.Custom(
						nerdtest.Binary(),
						"__complete",
						"--namespace",
						string(helpers.Read(nerdtest.Namespace)),
						"",
					)
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("run\t")),
			},
			{
				Description: "run -i",
				Command:     test.Command("__complete", "run", "-i", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains(testutil.CommonImage)),
			},
			{
				Description: "run -it",
				Command:     test.Command("__complete", "run", "-it", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains(testutil.CommonImage)),
			},
			{
				Description: "run -it --rm",
				Command:     test.Command("__complete", "run", "-it", "--rm", ""),
				Expected:    test.Expects(expect.ExitCodeSuccess, nil, expect.Contains(testutil.CommonImage)),
			},
			{
				Description: "namespace run -i",
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					// mind {"--namespace=test"} vs {"--namespace", "test"}
					return helpers.Custom(
						nerdtest.Binary(),
						"__complete",
						"--namespace",
						string(helpers.Read(nerdtest.Namespace)),
						"run",
						"-i",
						"",
					)
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains(testutil.CommonImage+"\n")),
			},
		},
	}

	testCase.Run(t)
}
