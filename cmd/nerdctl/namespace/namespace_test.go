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

package namespace

import (
	"encoding/json"
	"errors"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/containerd/nerdctl/v2/leptonic/api"
	"github.com/containerd/nerdctl/v2/leptonic/errs"
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/pkg/testutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/nerdtest"
	"github.com/containerd/nerdctl/v2/pkg/testutil/test"
)

func TestMain(m *testing.M) {
	testutil.M(m)
}

func TestCreateFail(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Description = "Namespace creation failure tests"

	// Docker has no concept of namespace
	testCase.Require = test.Not(nerdtest.Docker)

	testCase.SubTests = []*test.Case{
		{
			Description: "missing namespace name",
			Command:     test.Command("namespace", "create"),
			Expected:    test.Expects(1, []error{errors.New("requires at least 1 arg")}, nil),
		},
		{
			Description: "empty namespace name",
			Command:     test.Command("namespace", "create", ""),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, nil),
		},
		{
			Description: "invalid namespace name, non-ascii",
			Command:     test.Command("namespace", "create", "∞"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, nil),
		},
		{
			Description: "invalid namespace name",
			Command:     test.Command("namespace", "create", "_"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, nil),
		},
	}

	testCase.Run(t)
}

func TestInspectFail(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Description = "Namespace inspection failure tests"

	// Docker has no concept of namespace
	testCase.Require = test.Not(nerdtest.Docker)

	testCase.SubTests = []*test.Case{
		{
			Description: "missing namespace name",
			Command:     test.Command("namespace", "inspect"),
			Expected:    test.Expects(1, []error{errors.New("requires at least 1 arg")}, test.Equals("")),
		},
		{
			Description: "empty namespace name",
			Command:     test.Command("namespace", "inspect", ""),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "invalid namespace name, non-ascii",
			Command:     test.Command("namespace", "inspect", "∞"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "invalid namespace name",
			Command:     test.Command("namespace", "inspect", "_"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "non existent namespace",
			Command:     test.Command("namespace", "inspect", "doesnotexistandneverwill"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrNotFound}, test.Equals("")),
		},
		{
			Description: "mixing errors",
			Command:     test.Command("namespace", "inspect", "doesnotexistandneverwill", "_", "∞"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument, errs.ErrNotFound}, test.Equals("")),
		},
		{
			Description: "mixing errors and one good known namespace",
			Command:     test.Command("namespace", "inspect", "--format", "json", "doesnotexistandneverwill", "_", "∞", "default"),
			Expected: test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument, errs.ErrNotFound}, func(stdout string, info string, t *testing.T) {
				var expect []api.Namespace
				err := json.Unmarshal([]byte(stdout), &expect)
				assert.NilError(t, err, info)
				assert.Assert(t, len(expect) != 0, info)
				assert.Equal(t, expect[0].Name, "default", info)
			}),
		},
	}

	testCase.Run(t)
}

func TestUpdateFail(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Description = "Namespace updating failure tests"

	// Docker has no concept of namespace
	testCase.Require = test.Not(nerdtest.Docker)

	testCase.SubTests = []*test.Case{
		{
			Description: "missing namespace name",
			Command:     test.Command("namespace", "update", "--label", "key=value"),
			Expected:    test.Expects(1, []error{errors.New("requires at least 1 arg")}, test.Equals("")),
		},
		{
			Description: "empty namespace name",
			Command:     test.Command("namespace", "update", "", "--label", "key=value"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "invalid namespace name, non-ascii",
			Command:     test.Command("namespace", "update", "∞", "--label", "key=value"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "invalid namespace name",
			Command:     test.Command("namespace", "update", "_", "--label", "key=value"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "non existent namespace",
			Command:     test.Command("namespace", "update", "doesnotexistandneverwill", "--label", "key=value"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrNotFound}, test.Equals("")),
		},
		{
			Description: "exiting namespace with no label",
			Command:     test.Command("namespace", "update", "default"),
			Expected:    test.Expects(1, []error{namespace.ErrServiceNamespace, errs.ErrInvalidArgument}, test.Equals("")),
		},
		{
			Description: "exiting namespace with empty label key",
			Command:     test.Command("namespace", "update", "default", "--label"),
			Expected:    test.Expects(1, []error{errors.New("flag needs an argument")}, test.Equals("")),
		},
	}

	testCase.Run(t)
}

func TestCreateSuccess(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = test.Not(nerdtest.Docker)

	testCase.Description = "successful creation"
	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		data.Set("namespace", data.Identifier())
		return helpers.Command("namespace", "create", data.Identifier())
	}

	testCase.Expected = test.Expects(0, nil, test.Equals(""))

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("namespace", "remove", data.Identifier())
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "inspect works",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("namespace", "inspect", "--format", "json", data.Get("namespace"))
			},
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					ExitCode: 0,
					Errors:   nil,
					Output: func(stdout string, info string, t *testing.T) {
						var expect []api.Namespace
						err := json.Unmarshal([]byte(stdout), &expect)
						assert.NilError(t, err, info)
						assert.Assert(t, len(expect) != 0, info)
						assert.Equal(t, expect[0].Name, data.Get("namespace"), info)
						assert.Equal(t, len(expect[0].Labels), 0, info)
					},
				}
			},
		},
		/*
			{
				Description: "visible in list",
				NoParallel:  true,
				Command:     test.Command("namespace", "list", "--format", "json"),
				Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
					return &test.Expected{
						ExitCode: 0,
						Errors:   nil,
						Output: func(stdout string, info string, t *testing.T) {
							var expect []string
							err := json.Unmarshal([]byte(stdout), &expect)
							assert.NilError(t, err, info)
							assert.Assert(t, len(expect) != 0, info)
							var found string
							for _, n := range expect {
								if n == data.Get("namespace") {
									found = n
								}
							}
							assert.Assert(t, found != "", info)
						},
					}
				},
			},
		*/
		{
			Description: "remove works",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("namespace", "remove", data.Get("namespace"))
			},
			Expected: test.Expects(0, nil, test.Equals("")),
		},
		{
			Description: "not visible in list anymore",
			NoParallel:  true,
			Command:     test.Command("namespace", "list", "--format", "json"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					ExitCode: 0,
					Errors:   nil,
					Output: func(stdout string, info string, t *testing.T) {
						var expect []api.Namespace
						err := json.Unmarshal([]byte(stdout), &expect)
						assert.NilError(t, err, info)
						assert.Assert(t, len(expect) != 0, info)
						var found string
						for _, n := range expect {
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
			Expected: test.Expects(1, []error{errs.ErrNotFound}, test.Equals("")),
		},
	}

	testCase.Run(t)
}

func TestCreateWithLabelsSuccess(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = test.Not(nerdtest.Docker)

	testCase.Description = "successful creation"
	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		data.Set("namespace", data.Identifier())
		return helpers.Command("namespace", "create", data.Identifier())
	}

	testCase.Expected = test.Expects(0, nil, test.Equals(""))

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("namespace", "remove", data.Identifier())
	}

}
