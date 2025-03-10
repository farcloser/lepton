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

package volume_test

import (
	"errors"
	"testing"

	"github.com/containerd/errdefs"
	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/leptonic/errs"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

// TestVolumeRemove does test a large variety of volume remove situations, albeit none of them being
// hard filesystem errors.
// Behavior in such cases is largely unspecified, as there is no easy way to compare with Docker.
// Anyhow, borked filesystem conditions is not something we should be expected to deal with in a smart way.
func TestVolumeRemove(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.SubTests = []*test.Case{
		{
			Description: "arg missing should fail",
			Command:     test.Command("volume", "rm"),
			Expected:    test.Expects(1, []error{errors.New("requires at least 1 arg")}, nil),
		},
		{
			Description: "invalid identifier should fail",
			Command:     test.Command("volume", "rm", "∞"),
			Expected:    test.Expects(1, []error{errs.ErrInvalidArgument}, nil),
		},
		{
			Description: "non existent volume should fail",
			Command:     test.Command("volume", "rm", "doesnotexist"),
			Expected:    test.Expects(1, []error{errs.ErrNotFound}, nil),
		},
		{
			Description: "busy volume should fail",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("volume", "create", data.Identifier())
				helpers.Ensure("run", "--quiet", "-v", data.Identifier()+":/volume",
					"--name", data.Identifier(), testutil.CommonImage)
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rm", "-f", data.Identifier())
				helpers.Anyhow("volume", "rm", "-f", data.Identifier())
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("volume", "rm", data.Identifier())
			},

			Expected: test.Expects(1, []error{errdefs.ErrFailedPrecondition}, nil),
		},
		{
			Description: "busy anonymous volume should fail",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("run", "--quiet", "-v", "/volume", "--name", data.Identifier(), testutil.CommonImage)
				// Inspect the container and find the anonymous volume id
				inspect := nerdtest.InspectContainer(helpers, data.Identifier())
				var anonName string
				for _, v := range inspect.Mounts {
					if v.Destination == "/volume" {
						anonName = v.Name
						break
					}
				}
				assert.Assert(t, anonName != "", "Failed to find anonymous volume id", inspect)
				data.Set("anonName", anonName)
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rm", "-f", data.Identifier())
				helpers.Anyhow("volume", "rm", "-f", data.Get("anonName"))
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				// Try to remove that anon volume
				return helpers.Command("volume", "rm", data.Get("anonName"))
			},

			Expected: test.Expects(1, []error{errdefs.ErrFailedPrecondition}, nil),
		},
		{
			Description: "freed volume should succeed",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("volume", "create", data.Identifier())
				helpers.Ensure("run", "--quiet", "-v", data.Identifier()+":/volume", "--name", data.Identifier(), testutil.CommonImage)
				helpers.Ensure("rm", "-f", data.Identifier())
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rm", "-f", data.Identifier())
				helpers.Anyhow("volume", "rm", "-f", data.Identifier())
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("volume", "rm", data.Identifier())
			},

			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Equals(data.Identifier() + "\n"),
				}
			},
		},
		{
			Description: "dangling volume should succeed",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("volume", "create", data.Identifier())
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("volume", "rm", "-f", data.Identifier())
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("volume", "rm", data.Identifier())
			},

			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Equals(data.Identifier() + "\n"),
				}
			},
		},
		{
			Description: "part success multi-remove",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("volume", "create", data.Identifier())
				helpers.Ensure("volume", "create", data.Identifier("busy"))
				helpers.Ensure("run", "--quiet", "-v", data.Identifier("busy")+":/volume", "--name", data.Identifier(), testutil.CommonImage)
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rm", "-f", data.Identifier())
				helpers.Anyhow("volume", "rm", "-f", data.Identifier())
				helpers.Anyhow("volume", "rm", "-f", data.Identifier("busy"))
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("volume", "rm", "invalid∞", "nonexistent", data.Identifier("busy"), data.Identifier())
			},

			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					ExitCode: 1,
					Errors: []error{
						errs.ErrNotFound,
						errdefs.ErrFailedPrecondition,
						errs.ErrInvalidArgument,
					},
					Output: expect.Equals(data.Identifier() + "\n"),
				}
			},
		},
		{
			Description: "success multi-remove",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("volume", "create", data.Identifier("1"))
				helpers.Ensure("volume", "create", data.Identifier("2"))
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("volume", "rm", "-f", data.Identifier("1"), data.Identifier("2"))
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("volume", "rm", data.Identifier("1"), data.Identifier("2"))
			},

			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Equals(data.Identifier("1") + "\n" + data.Identifier("2") + "\n"),
				}
			},
		},
		{
			Description: "failing multi-remove",

			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("volume", "create", data.Identifier("busy"))
				helpers.Ensure("run", "--quiet", "-v", data.Identifier("busy")+":/volume", "--name", data.Identifier(), testutil.CommonImage)
			},

			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rm", "-f", data.Identifier())
				helpers.Anyhow("volume", "rm", "-f", data.Identifier("busy"))
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("volume", "rm", "invalid∞", "nonexistent", data.Identifier("busy"))
			},

			Expected: test.Expects(1, []error{
				errs.ErrNotFound,
				errdefs.ErrFailedPrecondition,
				errs.ErrInvalidArgument,
			}, nil),
		},
	}

	testCase.Run(t)
}
