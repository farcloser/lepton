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
	"testing"

	"go.farcloser.world/containers/security/cgroups"

	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/testutil"
)

func TestComposeTop(t *testing.T) {
	t.Parallel()

	if rootlessutil.IsRootless() && cgroups.Version() < 2 {
		t.Skip("test skipped for rootless containers on cgroup v1")
	}

	base := testutil.NewBase(t)
	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'

services:
  svc0:
    image: %s
    command: "sleep infinity"
  svc1:
    image: %s
`, testutil.CommonImage, testutil.NginxAlpineImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").AssertOK()

	// a running container should have the process command in output
	base.ComposeCmd("-f", comp.YAMLFullPath(), "top", "svc0").AssertOutContains("sleep infinity")
	base.ComposeCmd("-f", comp.YAMLFullPath(), "top", "svc1").AssertOutContains("nginx")
}
