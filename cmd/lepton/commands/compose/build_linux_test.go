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

package compose_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestComposeBuild(t *testing.T) {
	const imageSvc0 = "composebuild_svc0"
	const imageSvc1 = "composebuild_svc1"

	dockerComposeYAML := fmt.Sprintf(`
services:
  svc0:
    build: .
    image: %s
    ports:
    - 8080:80
    depends_on:
    - svc1
  svc1:
    build: .
    image: %s
    ports:
    - 8081:80
`, imageSvc0, imageSvc1)

	dockerfile := "FROM " + testutil.AlpineImage

	testCase := nerdtest.Setup()

	testCase.Require = nerdtest.Build

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		err := os.WriteFile(filepath.Join(data.TempDir(), "compose.yaml"), []byte(dockerComposeYAML), 0o600)
		assert.NilError(t, err)
		err = os.WriteFile(filepath.Join(data.TempDir(), "Dockerfile"), []byte(dockerfile), 0o600)
		assert.NilError(t, err)
		data.Set("composeYaml", filepath.Join(data.TempDir(), "compose.yaml"))
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "build svc0",
			NoParallel:  true,
			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("compose", "-f", data.Get("composeYaml"), "build", "svc0")
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("images")
			},

			Expected: test.Expects(0, nil, expect.All(
				expect.Contains(imageSvc0),
				expect.DoesNotContain(imageSvc1),
			)),
		},
		{
			Description: "build svc0 and svc1",
			NoParallel:  true,
			Setup: func(data test.Data, helpers test.Helpers) {
				helpers.Ensure("compose", "-f", data.Get("composeYaml"), "build", "svc0", "svc1")
			},

			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("images")
			},

			Expected: test.Expects(0, nil, expect.All(
				expect.Contains(imageSvc0),
				expect.Contains(imageSvc1),
			)),
		},
		{
			Description: "build no arg",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("compose", "-f", data.Get("composeYaml"), "build")
			},

			Expected: test.Expects(0, nil, nil),
		},
		{
			Description: "build bogus",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("compose", "-f", data.Get("composeYaml"), "build", "svc0", "svc100")
			},

			Expected: test.Expects(1, []error{errors.New("no such service: svc100")}, nil),
		},
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rmi", imageSvc0, imageSvc1)
	}

	testCase.Run(t)
}
