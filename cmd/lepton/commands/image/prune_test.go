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

package image_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestImagePrune(t *testing.T) {
	testCase := nerdtest.Setup()

	// Cannot use a custom namespace with buildkitd right now, so, no parallel it is
	testCase.NoParallel = true
	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		// Stop and remove all running containers. This is to ensure we can remove all
		contList := strings.TrimSpace(helpers.Capture("ps", "-aq"))
		if contList != "" {
			helpers.Ensure(append([]string{"rm", "-f"}, strings.Split(contList, "\n")...)...)
		}

		// We need to delete everything here for prune to make any sense
		imgList := strings.TrimSpace(helpers.Capture("images", "--no-trunc", "-aq"))
		if imgList != "" {
			helpers.Ensure(append([]string{"rmi", "-f"}, strings.Split(imgList, "\n")...)...)
		}
	}
	testCase.SubTests = []*test.Case{
		{
			Description: "without all",
			NoParallel:  true,
			Require: require.All(
				// This never worked with Docker - the only reason we ever got <none> was side effects from other tests
				// See inline comments.
				require.Not(nerdtest.Docker),
				nerdtest.Build,
			),
			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rmi", "-f", data.Identifier())
			},
			Setup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				dockerfile := fmt.Sprintf(`FROM %s
				CMD ["echo", "test-image-prune"]
					`, testutil.CommonImage)

				buildCtx := data.TempDir()
				err := os.WriteFile(
					filepath.Join(buildCtx, "Dockerfile"),
					[]byte(dockerfile),
					0o600,
				)
				assert.NilError(helpers.T(), err)
				helpers.Ensure("build", buildCtx)
				// After we rebuild with tag, docker will no longer show the <none> version from above
				// Swapping order does not change anything.
				helpers.Ensure("build", "-t", identifier, buildCtx)
				imgList := helpers.Capture("images")
				assert.Assert(t, strings.Contains(imgList, "<none>"), "Missing <none>")
				assert.Assert(t, strings.Contains(imgList, identifier), "Missing "+identifier)
			},
			Command: test.Command("image", "prune", "--force"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				identifier := data.Identifier()
				return &test.Expected{
					Output: expect.All(
						func(stdout, info string, t *testing.T) {
							assert.Assert(t, !strings.Contains(stdout, identifier), info)
						},
						func(stdout, info string, t *testing.T) {
							imgList := helpers.Capture("images")
							assert.Assert(t, !strings.Contains(imgList, "<none>"), imgList)
							assert.Assert(t, strings.Contains(imgList, identifier), info)
						},
					),
				}
			},
		},
		{
			Description: "with all",
			Require: require.All(
				// Same as above
				require.Not(nerdtest.Docker),
				nerdtest.Build,
			),
			// Cannot use a custom namespace with buildkitd right now, so, no parallel it is
			NoParallel: true,
			Cleanup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				helpers.Anyhow("rmi", "-f", identifier)
				helpers.Anyhow("rm", "-f", identifier)
			},
			Setup: func(data test.Data, helpers test.Helpers) {
				identifier := data.Identifier()
				dockerfile := fmt.Sprintf(`FROM %s
				CMD ["echo", "test-image-prune"]
					`, testutil.CommonImage)

				buildCtx := data.TempDir()
				err := os.WriteFile(
					filepath.Join(buildCtx, "Dockerfile"),
					[]byte(dockerfile),
					0o600,
				)
				assert.NilError(helpers.T(), err)
				helpers.Ensure("build", buildCtx)
				helpers.Ensure("build", "-t", identifier, buildCtx)
				imgList := helpers.Capture("images")
				assert.Assert(t, strings.Contains(imgList, "<none>"), "Missing <none>")
				assert.Assert(t, strings.Contains(imgList, identifier), "Missing "+identifier)
				helpers.Ensure("run", "--quiet", "--name", identifier, identifier)
			},
			Command: test.Command("image", "prune", "--force", "--all"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.All(
						func(stdout, info string, t *testing.T) {
							assert.Assert(t, !strings.Contains(stdout, data.Identifier()), info)
						},
						func(stdout, info string, t *testing.T) {
							imgList := helpers.Capture("images")
							assert.Assert(t, strings.Contains(imgList, data.Identifier()), info)
							assert.Assert(t, !strings.Contains(imgList, "<none>"), imgList)
							helpers.Ensure("rm", "-f", data.Identifier())
							removed := helpers.Capture("image", "prune", "--force", "--all")
							assert.Assert(t, strings.Contains(removed, data.Identifier()), info)
							imgList = helpers.Capture("images")
							assert.Assert(t, !strings.Contains(imgList, data.Identifier()), info)
						},
					),
				}
			},
		},
		{
			Description: "with filter label",
			Require:     nerdtest.Build,
			// Cannot use a custom namespace with buildkitd right now, so, no parallel it is
			NoParallel: true,
			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rmi", "-f", data.Identifier())
			},
			Setup: func(data test.Data, helpers test.Helpers) {
				dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "test-image-prune-filter-label"]
LABEL foo=bar
LABEL version=0.1`, testutil.CommonImage)
				buildCtx := data.TempDir()
				err := os.WriteFile(
					filepath.Join(buildCtx, "Dockerfile"),
					[]byte(dockerfile),
					0o600,
				)
				assert.NilError(helpers.T(), err)
				helpers.Ensure("build", "-t", data.Identifier(), buildCtx)
				imgList := helpers.Capture("images")
				assert.Assert(
					t,
					strings.Contains(imgList, data.Identifier()),
					"Missing "+data.Identifier(),
				)
			},
			Command: test.Command(
				"image",
				"prune",
				"--force",
				"--all",
				"--filter",
				"label=foo=baz",
			),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.All(
						func(stdout, info string, t *testing.T) {
							assert.Assert(t, !strings.Contains(stdout, data.Identifier()), info)
						},
						func(stdout, info string, t *testing.T) {
							imgList := helpers.Capture("images")
							assert.Assert(t, strings.Contains(imgList, data.Identifier()), info)
						},
						func(stdout, info string, t *testing.T) {
							prune := helpers.Capture(
								"image",
								"prune",
								"--force",
								"--all",
								"--filter",
								"label=foo=bar",
							)
							assert.Assert(t, strings.Contains(prune, data.Identifier()), info)
							imgList := helpers.Capture("images")
							assert.Assert(t, !strings.Contains(imgList, data.Identifier()), info)
						},
					),
				}
			},
		},
		{
			Description: "with until",
			Require:     nerdtest.Build,
			// Cannot use a custom namespace with buildkitd right now, so, no parallel it is
			NoParallel: true,
			Cleanup: func(data test.Data, helpers test.Helpers) {
				helpers.Anyhow("rmi", "-f", data.Identifier())
			},
			Setup: func(data test.Data, helpers test.Helpers) {
				dockerfile := fmt.Sprintf(`FROM %s
RUN echo "Anything, so that we create actual content for docker to set the current time for CreatedAt"
CMD ["echo", "test-image-prune-until"]`, testutil.CommonImage)
				buildCtx := data.TempDir()
				err := os.WriteFile(
					filepath.Join(buildCtx, "Dockerfile"),
					[]byte(dockerfile),
					0o600,
				)
				assert.NilError(helpers.T(), err)
				helpers.Ensure("build", "-t", data.Identifier(), buildCtx)
				imgList := helpers.Capture("images")
				assert.Assert(
					t,
					strings.Contains(imgList, data.Identifier()),
					"Missing "+data.Identifier(),
				)
				data.Set("imageID", data.Identifier())
			},
			Command: test.Command("image", "prune", "--force", "--all", "--filter", "until=12h"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.All(
						expect.DoesNotContain(data.Get("imageID")),
						func(stdout, info string, t *testing.T) {
							imgList := helpers.Capture("images")
							assert.Assert(t, strings.Contains(imgList, data.Get("imageID")), info)
						},
					),
				}
			},
			SubTests: []*test.Case{
				{
					Description: "Wait and remove until=10ms",
					NoParallel:  true,
					Setup: func(data test.Data, helpers test.Helpers) {
						time.Sleep(1 * time.Second)
					},
					Command: test.Command(
						"image",
						"prune",
						"--force",
						"--all",
						"--filter",
						"until=10ms",
					),
					Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
						return &test.Expected{
							Output: expect.All(
								expect.Contains(data.Get("imageID")),
								func(stdout, info string, t *testing.T) {
									imgList := helpers.Capture("images")
									assert.Assert(
										t,
										!strings.Contains(imgList, data.Get("imageID")),
										imgList,
										info,
									)
								},
							),
						}
					},
				},
			},
		},
	}

	testCase.Run(t)
}
