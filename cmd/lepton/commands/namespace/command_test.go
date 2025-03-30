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

package namespace_test

import (
	"encoding/json"
	"errors"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/leptonic/api"
	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/leptonic/services/namespace"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestMain(m *testing.M) {
	testutil.M(m)
}

func TestCreateFail(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Description = "Namespace creation failure tests"

	// Docker has no concept of namespace
	testCase.Require = require.Not(nerdtest.Docker)

	testCase.SubTests = []*test.Case{
		{
			Description: "missing namespace name",
			Command:     test.Command("namespace", "create"),
			Expected:    test.Expects(1, []error{errors.New("accepts 1 arg(s), received 0")}, nil),
		},
		{
			Description: "empty namespace name",
			Command:     test.Command("namespace", "create", ""),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				nil,
			),
		},
		{
			Description: "invalid namespace name, non-ascii",
			Command:     test.Command("namespace", "create", "∞"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				nil,
			),
		},
		{
			Description: "invalid namespace name",
			Command:     test.Command("namespace", "create", "_"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				nil,
			),
		},
	}

	testCase.Run(t)
}

func TestInspectFail(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Description = "Namespace inspection failure tests"

	// Docker has no concept of namespace
	testCase.Require = require.Not(nerdtest.Docker)

	testCase.SubTests = []*test.Case{
		{
			Description: "missing namespace name",
			Command:     test.Command("namespace", "inspect"),
			Expected: test.Expects(
				1,
				[]error{errors.New("requires at least 1 arg")},
				expect.Equals(""),
			),
		},
		{
			Description: "empty namespace name",
			Command:     test.Command("namespace", "inspect", ""),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "invalid namespace name, non-ascii",
			Command:     test.Command("namespace", "inspect", "∞"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "invalid namespace name",
			Command:     test.Command("namespace", "inspect", "_"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "non existent namespace",
			Command:     test.Command("namespace", "inspect", "doesnotexistandneverwill"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrNotFound},
				expect.Equals(""),
			),
		},
		{
			Description: "mixing errors",
			Command:     test.Command("namespace", "inspect", "doesnotexistandneverwill", "_", "∞"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument, errs.ErrNotFound},
				expect.Equals(""),
			),
		},
		/*
			// FIXME looks like for some reason windows does not have the default namespace at this point
			{
				Description: "mixing errors and one good known namespace",
				// FIXME unhardcode namespace name
				Command: test.Command("namespace", "inspect", "--format", formatter.FormatJSON, "doesnotexistandneverwill", "_", "∞", "cli-test"),
				Expected: test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument, errs.ErrNotFound}, func(stdout string, info string, t *testing.T) {
					var expect []api.Namespace
					err := json.Unmarshal([]byte(stdout), &expect)
					assert.NilError(t, err, info)
					assert.Assert(t, len(expect) != 0, info)
					assert.Equal(t, expect[0].Name, "cli-test", info)
				}),
			},

		*/
	}

	testCase.Run(t)
}

func TestUpdateFail(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Description = "Namespace updating failure tests"

	// Docker has no concept of namespace
	testCase.Require = require.Not(nerdtest.Docker)

	testCase.SubTests = []*test.Case{
		{
			Description: "missing namespace name",
			Command:     test.Command("namespace", "update", "--label", "key=value"),
			Expected: test.Expects(
				1,
				[]error{errors.New("accepts 1 arg(s), received 0")},
				expect.Equals(""),
			),
		},
		{
			Description: "empty namespace name",
			Command:     test.Command("namespace", "update", "", "--label", "key=value"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "invalid namespace name, non-ascii",
			Command:     test.Command("namespace", "update", "∞", "--label", "key=value"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "invalid namespace name",
			Command:     test.Command("namespace", "update", "_", "--label", "key=value"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "non existent namespace",
			Command: test.Command(
				"namespace",
				"update",
				"doesnotexistandneverwill",
				"--label",
				"key=value",
			),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrNotFound},
				expect.Equals(""),
			),
		},
		{
			Description: "exiting namespace with no label",
			Command:     test.Command("namespace", "update", "cli-test"),
			Expected: test.Expects(
				1,
				[]error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument},
				expect.Equals(""),
			),
		},
		{
			Description: "exiting namespace with empty label key",
			Command:     test.Command("namespace", "update", "cli-test", "--label"),
			Expected: test.Expects(
				1,
				[]error{errors.New("flag needs an argument")},
				expect.Equals(""),
			),
		},
	}

	testCase.Run(t)
}

func TestCreateSuccess(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = require.Not(nerdtest.Docker)

	testCase.Description = "successful creation"
	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		data.Set("namespace", data.Identifier())
		return helpers.Command("namespace", "create", data.Identifier())
	}

	testCase.Expected = test.Expects(0, nil, expect.Equals(""))

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("namespace", "remove", data.Identifier())
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "inspect works",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command(
					"namespace",
					"inspect",
					"--format",
					formatter.FormatJSON,
					data.Get("namespace"),
				)
			},
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					ExitCode: 0,
					Errors:   nil,
					Output: func(stdout string, info string, t *testing.T) {
						var expected []api.Namespace
						err := json.Unmarshal([]byte(stdout), &expected)
						assert.NilError(t, err, info)
						assert.Assert(t, len(expected) != 0, info)
						assert.Equal(t, expected[0].Name, data.Get("namespace"), info)
						assert.Equal(t, len(expected[0].Labels), 0, info)
					},
				}
			},
		},
		{
			Description: "visible in list",
			NoParallel:  true,
			Command:     test.Command("namespace", "list", "--format", formatter.FormatJSON),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					ExitCode: 0,
					Errors:   nil,
					Output: func(stdout string, info string, t *testing.T) {
						var expected []api.Namespace
						err := json.Unmarshal([]byte(stdout), &expected)
						assert.NilError(t, err, info)
						var found string
						for _, n := range expected {
							if n.Name == data.Get("namespace") {
								found = n.Name
							}
						}
						assert.Assert(t, found != "", info)
					},
				}
			},
		},
		{
			Description: "remove works",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("namespace", "remove", data.Get("namespace"))
			},
			Expected: test.Expects(0, nil, expect.Equals("")),
		},
		{
			Description: "not visible in list anymore",
			NoParallel:  true,
			Command:     test.Command("namespace", "list", "--format", formatter.FormatJSON),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					ExitCode: 0,
					Errors:   nil,
					Output: func(stdout string, info string, t *testing.T) {
						var expected []api.Namespace
						err := json.Unmarshal([]byte(stdout), &expected)
						assert.NilError(t, err, info)
						var found string
						for _, n := range expected {
							if n.Name == data.Get("namespace") {
								found = n.Name
							}
						}
						assert.Assert(t, found == "", info)
					},
				}
			},
		},
		{
			Description: "not inspectable anymore",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("namespace", "inspect", data.Get("namespace"))
			},
			Expected: test.Expects(1, []error{errs.ErrNotFound}, expect.Equals("")),
		},
	}

	testCase.Run(t)
}

func TestCreateWithLabelsSuccess(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = require.Not(nerdtest.Docker)

	testCase.Description = "successful creation"
	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		data.Set("namespace", data.Identifier())
		return helpers.Command("namespace", "create", data.Identifier())
	}

	testCase.Expected = test.Expects(0, nil, expect.Equals(""))

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("namespace", "remove", data.Identifier())
	}

	testCase.Run(t)
}
