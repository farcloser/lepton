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

package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/containerd/nerdctl/v2/pkg/testutil"
	"github.com/containerd/nerdctl/v2/pkg/testutil/nerdtest"
	"github.com/containerd/nerdctl/v2/pkg/testutil/test"
)

func TestBuilderPrune(t *testing.T) {
	nerdtest.Setup()

	testCase := &test.Case{
		NoParallel: true,
		Require: test.Require(
			nerdtest.Build,
			test.Not(test.Windows),
		),
		SubTests: []*test.Case{
			{
				Description: "PruneForce",
				NoParallel:  true,
				Setup: func(data test.Data, helpers test.Helpers) {
					dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "test-builder-prune"]`, testutil.CommonImage)
					buildCtx := data.TempDir()
					err := os.WriteFile(filepath.Join(buildCtx, "Dockerfile"), []byte(dockerfile), 0o600)
					assert.NilError(helpers.T(), err)
					helpers.Ensure("build", buildCtx)
				},
				Command:  test.Command("builder", "prune", "--force"),
				Expected: test.Expects(0, nil, nil),
			},
			{
				Description: "PruneForceAll",
				NoParallel:  true,
				Setup: func(data test.Data, helpers test.Helpers) {
					dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "test-builder-prune"]`, testutil.CommonImage)
					buildCtx := data.TempDir()
					err := os.WriteFile(filepath.Join(buildCtx, "Dockerfile"), []byte(dockerfile), 0o600)
					assert.NilError(helpers.T(), err)
					helpers.Ensure("build", buildCtx)
				},
				Command:  test.Command("builder", "prune", "--force", "--all"),
				Expected: test.Expects(0, nil, nil),
			},
		},
	}

	testCase.Run(t)
}
