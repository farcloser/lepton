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

func TestComposeConfig(t *testing.T) {
	dockerComposeYAML := `
services:
  hello:
    image: alpine:3.13
`
	testCase := nerdtest.Setup()

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		err := os.WriteFile(
			filepath.Join(data.TempDir(), "compose.yaml"),
			[]byte(dockerComposeYAML),
			0o600,
		)
		assert.NilError(t, err)
		data.Set("compyaml", filepath.Join(data.TempDir(), "compose.yaml"))
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "config contains service name",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("compose", "-f", data.Get("compyaml"), "config")
			},
			Expected: test.Expects(0, nil, expect.Contains("hello:")),
		},
		{
			Description: "config --services is exactly service name",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command(
					"compose",
					"-f",
					data.Get("compyaml"),
					"config",
					"--services",
				)
			},
			Expected: test.Expects(0, nil, expect.Equals("hello\n")),
		},
		{
			Description: "config --hash=* contains service name",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("compose", "-f", data.Get("compyaml"), "config", "--hash=*")
			},
			Expected: test.Expects(0, nil, expect.Contains("hello")),
		},
	}

	testCase.Run(t)
}

func TestComposeConfigWithPrintServiceHash(t *testing.T) {
	dockerComposeYAML := `
services:
  hello:
    image: alpine:%s
`
	testCase := nerdtest.Setup()

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		err := os.WriteFile(
			filepath.Join(data.TempDir(), "compose.yaml"),
			[]byte(fmt.Sprintf(dockerComposeYAML, "3.13")),
			0o600,
		)

		hash := helpers.Capture(
			"compose",
			"-f",
			filepath.Join(data.TempDir(), "compose.yaml"),
			"config",
			"--hash=hello",
		)
		assert.NilError(t, err)
		data.Set("hash", hash)

		err = os.WriteFile(
			filepath.Join(data.TempDir(), "compose.yaml"),
			[]byte(fmt.Sprintf(dockerComposeYAML, "3.14")),
			0o600,
		)
		assert.NilError(t, err)
	}

	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		return helpers.Command(
			"compose",
			"-f",
			filepath.Join(data.TempDir(), "compose.yaml"),
			"config",
			"--hash=hello",
		)
	}

	testCase.Expected = func(data test.Data, helpers test.Helpers) *test.Expected {
		return &test.Expected{
			ExitCode: 0,
			Output: func(stdout string, info string, t *testing.T) {
				assert.Assert(t, data.Get("hash") != stdout, info)
			},
		}
	}

	testCase.Run(t)
}

func TestComposeConfigWithMultipleFile(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	dockerComposeYAML := `
services:
  hello1:
    image: alpine:3.13
`

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	comp.WriteFile("docker-compose.test.yml", `
services:
  hello2:
    image: alpine:3.14
`)
	comp.WriteFile("docker-compose.override.yml", `
services:
  hello1:
    image: alpine:3.14
`)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "-f", filepath.Join(comp.Dir(), "docker-compose.test.yml"), "config").
		AssertOutContains("alpine:3.14")
	base.ComposeCmd("--project-directory", comp.Dir(), "config", "--services").
		AssertOutExactly("hello1\n")
	base.ComposeCmd("--project-directory", comp.Dir(), "config").AssertOutContains("alpine:3.14")
}

func TestComposeConfigWithComposeFileEnv(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)

	dockerComposeYAML := `
services:
  hello1:
    image: alpine:3.13
`

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()

	comp.WriteFile("docker-compose.test.yml", `
services:
  hello2:
    image: alpine:3.14
`)

	base.Env = append(
		base.Env,
		"COMPOSE_FILE="+comp.YAMLFullPath()+","+filepath.Join(
			comp.Dir(),
			"docker-compose.test.yml",
		),
		"COMPOSE_PATH_SEPARATOR=,",
	)

	base.ComposeCmd("config").AssertOutContains("alpine:3.14")
	base.ComposeCmd("--project-directory", comp.Dir(), "config", "--services").
		AssertOutContainsAll("hello1\n", "hello2\n")
	base.ComposeCmd("--project-directory", comp.Dir(), "config").AssertOutContains("alpine:3.14")
}

func TestComposeConfigWithEnvFile(t *testing.T) {
	dockerComposeYAML := `
services:
  hello:
    image: ${image}
`

	envFileContent := `
image: hello-world
`

	testCase := nerdtest.Setup()

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		err := os.WriteFile(
			filepath.Join(data.TempDir(), "compose.yaml"),
			[]byte(dockerComposeYAML),
			0o600,
		)
		assert.NilError(t, err)
		err = os.WriteFile(filepath.Join(data.TempDir(), "env"), []byte(envFileContent), 0o600)
		assert.NilError(t, err)
	}

	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		return helpers.Command(
			"compose",
			"-f",
			filepath.Join(data.TempDir(), "compose.yaml"),
			"--env-file",
			filepath.Join(data.TempDir(), "env"),
			"config",
		)
	}

	testCase.Expected = test.Expects(0, nil, expect.Contains("image: hello-world"))

	testCase.Run(t)
}
