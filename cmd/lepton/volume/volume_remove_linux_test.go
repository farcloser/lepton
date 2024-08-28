package volume

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/icmd"

	"github.com/farcloser/lepton/pkg/errs"
	"github.com/farcloser/lepton/pkg/testutil"
)

// TestVolumeRemove does test a large variety of volume remove situations, except conditions deriving from
// hard filesystem errors.
func TestVolumeRemove(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	cobraRequireArg := "requires at least 1 arg"

	testCases := []struct {
		description        string
		command            func(tID string) *testutil.Cmd
		setup              func(tID string)
		cleanup            func(tID string)
		expected           func(tID string) icmd.Expected
		inspect            func(t *testing.T, stdout string, stderr string)
		dockerIncompatible bool
	}{
		{
			description: "arg missing should fail",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm")
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Err:      cobraRequireArg,
				}
			},
		},
		{
			description: "empty volume name should fail",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", "")
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Err:      errs.ErrInvalidArgument.Error(),
				}
			},
		},
		{
			description: "invalid identifier should fail",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", "∞")
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Err:      errs.ErrInvalidArgument.Error(),
				}
			},
		},
		{
			description: "non existent volume should fail",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", "doesnotexist")
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Err:      errs.ErrNotFound.Error(),
				}
			},
		},
		{
			description: "busy volume should fail",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", tID)
			},
			setup: func(tID string) {
				base.Cmd("volume", "create", tID).AssertOK()
				base.Cmd("run", "-v", fmt.Sprintf("%s:/volume", tID), "--name", tID, testutil.CommonImage).AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("rm", "-f", tID).Run()
				base.Cmd("volume", "rm", "-f", tID).Run()
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Err:      errs.ErrFailedPrecondition.Error(),
				}

			},
		},
		{
			description: "busy anonymous volume should fail",
			command: func(tID string) *testutil.Cmd {
				// Inspect the container and find the anonymous volume id
				inspect := base.InspectContainer(tID)
				var anonName string
				for _, v := range inspect.Mounts {
					if v.Destination == "/volume" {
						anonName = v.Name
						break
					}
				}
				assert.Assert(t, anonName != "", "Failed to find anonymous volume id")

				// Try to remove that anon volume
				return base.Cmd("volume", "rm", anonName)
			},
			setup: func(tID string) {
				base.Cmd("run", "-v", fmt.Sprintf("%s:/volume", tID), "--name", tID, testutil.CommonImage).AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("rm", "-f", tID).Run()
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Err:      errs.ErrFailedPrecondition.Error(),
				}

			},
		},
		{
			description: "freed volume should succeed",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", tID)
			},
			setup: func(tID string) {
				base.Cmd("volume", "create", tID).AssertOK()
				base.Cmd("run", "-v", fmt.Sprintf("%s:/volume", tID), "--name", tID, testutil.CommonImage).AssertOK()
				base.Cmd("rm", "-f", tID).AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("rm", "-f", tID).Run()
				base.Cmd("volume", "rm", "-f", tID).Run()
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					Out: tID,
				}
			},
		},
		{
			description: "dangling volume should succeed",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", tID)
			},
			setup: func(tID string) {
				base.Cmd("volume", "create", tID).AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("volume", "rm", "-f", tID).Run()
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					Out: tID,
				}
			},
		},
		{
			description: "part success multi-remove",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", "invalid∞", "nonexistent", tID+"-busy", tID)
			},
			setup: func(tID string) {
				base.Cmd("volume", "create", tID).AssertOK()
				base.Cmd("volume", "create", tID+"-busy").AssertOK()
				base.Cmd("run", "-v", fmt.Sprintf("%s:/volume", tID+"-busy"), "--name", tID, testutil.CommonImage).AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("rm", "-f", tID).Run()
				base.Cmd("volume", "rm", "-f", tID).Run()
				base.Cmd("volume", "rm", "-f", tID+"-busy").Run()
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
					Out:      tID,
				}
			},
			inspect: func(t *testing.T, stdout string, stderr string) {
				assert.Assert(t, strings.Contains(stderr, errs.ErrNotFound.Error()))
				assert.Assert(t, strings.Contains(stderr, errs.ErrFailedPrecondition.Error()))
				assert.Assert(t, strings.Contains(stderr, errs.ErrInvalidArgument.Error()))
			},
		},
		{
			description: "success multi-remove",
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", tID+"-1", tID+"-2")
			},
			setup: func(tID string) {
				base.Cmd("volume", "create", tID+"-1").AssertOK()
				base.Cmd("volume", "create", tID+"-2").AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("volume", "rm", "-f", tID+"-1", tID+"-2").Run()
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					Out: tID + "-1\n" + tID + "-2",
				}
			},
		},
		{
			description: "failing multi-remove",
			setup: func(tID string) {
				base.Cmd("volume", "create", tID+"-busy").AssertOK()
				base.Cmd("run", "-v", fmt.Sprintf("%s:/volume", tID+"-busy"), "--name", tID, testutil.CommonImage).AssertOK()
			},
			cleanup: func(tID string) {
				base.Cmd("rm", "-f", tID).Run()
				base.Cmd("volume", "rm", "-f", tID+"-busy").Run()
			},
			command: func(tID string) *testutil.Cmd {
				return base.Cmd("volume", "rm", "invalid∞", "nonexistent", tID+"-busy")
			},
			expected: func(tID string) icmd.Expected {
				return icmd.Expected{
					ExitCode: 1,
				}
			},
			inspect: func(t *testing.T, stdout string, stderr string) {
				assert.Assert(t, strings.Contains(stderr, errs.ErrNotFound.Error()))
				assert.Assert(t, strings.Contains(stderr, errs.ErrFailedPrecondition.Error()))
				assert.Assert(t, strings.Contains(stderr, errs.ErrInvalidArgument.Error()))
			},
		},
	}

	for _, test := range testCases {
		currentTest := test
		t.Run(currentTest.description, func(tt *testing.T) {
			if currentTest.dockerIncompatible {
				testutil.DockerIncompatible(tt)
			}

			tt.Parallel()

			tID := testutil.Identifier(tt)

			if currentTest.cleanup != nil {
				currentTest.cleanup(tID)
				tt.Cleanup(func() {
					currentTest.cleanup(tID)
				})
			}
			if currentTest.setup != nil {
				currentTest.setup(tID)
			}

			// See https://github.com/containerd/nerdctl/issues/3130
			// We run first to capture the underlying icmd command and output
			cmd := currentTest.command(tID)
			res := cmd.Run()

			expect := currentTest.expected(tID)
			if base.Target == testutil.Docker {
				expect.Err = ""
			}

			cmd.Assert(expect)

			if currentTest.inspect != nil {
				currentTest.inspect(tt, res.Stdout(), res.Stderr())
			}
		})
	}
}
