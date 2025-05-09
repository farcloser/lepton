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

// Package issues is meant to document testing for complex scenarios type of issues that cannot simply be ascribed
// to a specific package.
package issues_test

import (
	"fmt"
	"testing"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest/registry"
)

func TestIssue3425(t *testing.T) {
	nerdtest.Setup()

	var reg *registry.Server

	testCase := &test.Case{
		Require: nerdtest.Registry,
		Setup: func(data test.Data, helpers test.Helpers) {
			reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
			reg.Setup(data, helpers)
		},
		Cleanup: func(data test.Data, helpers test.Helpers) {
			if reg != nil {
				reg.Cleanup(data, helpers)
			}
		},
		SubTests: []*test.Case{
			{
				Description: "with tag",
				Require:     nerdtest.Private,
				Setup: func(data test.Data, helpers test.Helpers) {
					identifier := data.Identifier()
					helpers.Ensure("image", "pull", testutil.CommonImage)
					helpers.Ensure("run", "--quiet", "-d", "--name", identifier, testutil.CommonImage)
					helpers.Ensure("image", "rm", "-f", testutil.CommonImage)
					helpers.Ensure("image", "pull", testutil.CommonImage)
					helpers.Ensure("tag", testutil.CommonImage, fmt.Sprintf("localhost:%d/%s", reg.Port, identifier))
				},
				Cleanup: func(data test.Data, helpers test.Helpers) {
					identifier := data.Identifier()
					helpers.Anyhow("rm", "-f", identifier)
					helpers.Anyhow("rmi", "-f", fmt.Sprintf("localhost:%d/%s", reg.Port, identifier))
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("push", fmt.Sprintf("localhost:%d/%s", reg.Port, data.Identifier()))
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
			},
			{
				Description: "with commit",
				Require:     nerdtest.Private,
				Setup: func(data test.Data, helpers test.Helpers) {
					identifier := data.Identifier()
					helpers.Ensure("image", "pull", testutil.CommonImage)
					helpers.Ensure(
						"run",
						"--quiet",
						"-d",
						"--name",
						identifier,
						testutil.CommonImage,
						"touch",
						"/something",
					)
					helpers.Ensure("image", "rm", "-f", testutil.CommonImage)
					helpers.Ensure("image", "pull", testutil.CommonImage)
					helpers.Ensure("commit", identifier, fmt.Sprintf("localhost:%d/%s", reg.Port, identifier))
				},
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rm", "-f", data.Identifier())
					helpers.Anyhow("rmi", "-f", fmt.Sprintf("localhost:%d/%s", reg.Port, data.Identifier()))
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("push", fmt.Sprintf("localhost:%d/%s", reg.Port, data.Identifier()))
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
			},
			{
				Description: "with save",
				Require:     nerdtest.Private,
				Setup: func(data test.Data, helpers test.Helpers) {
					helpers.Ensure("image", "pull", testutil.CommonImage)
					helpers.Ensure("run", "--quiet", "-d", "--name", data.Identifier(), testutil.CommonImage)
					helpers.Ensure("image", "rm", "-f", testutil.CommonImage)
					helpers.Ensure("image", "pull", testutil.CommonImage)
				},
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rm", "-f", data.Identifier())
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("save", testutil.CommonImage)
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
			},
			{
				Description: "with convert",
				Require: require.All(
					nerdtest.Private,
					require.Not(require.Windows),
					require.Not(nerdtest.Docker),
				),
				Setup: func(data test.Data, helpers test.Helpers) {
					helpers.Ensure("image", "pull", testutil.CommonImage)
					helpers.Ensure("run", "--quiet", "-d", "--name", data.Identifier(), testutil.CommonImage)
					helpers.Ensure("image", "rm", "-f", testutil.CommonImage)
					helpers.Ensure("image", "pull", testutil.CommonImage)
				},
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rm", "-f", data.Identifier())
					helpers.Anyhow("rmi", "-f", data.Identifier())
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command("image", "convert", "--oci", testutil.CommonImage, data.Identifier())
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
			},
		},
	}

	testCase.Run(t)
}
