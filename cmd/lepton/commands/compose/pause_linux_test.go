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

	"go.farcloser.world/lepton/pkg/testutil"
)

func TestComposePauseAndUnpause(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	switch base.Info().CgroupDriver {
	case cgroups.NoneManager, "":
		t.Skip("requires cgroup (for pausing)")
	}

	dockerComposeYAML := fmt.Sprintf(`
version: '3.1'

services:
  svc0:
    image: %s
    command: "sleep infinity"
  svc1:
    image: %s
    command: "sleep infinity"
`, testutil.CommonImage, testutil.CommonImage)

	comp := testutil.NewComposeDir(t, dockerComposeYAML)
	defer comp.CleanUp()
	projectName := comp.ProjectName()
	t.Logf("projectName=%q", projectName)

	base.ComposeCmd("-f", comp.YAMLFullPath(), "up", "-d").AssertOK()
	defer base.ComposeCmd("-f", comp.YAMLFullPath(), "down", "-v").AssertOK()

	// pause a service should (only) pause its own container
	base.ComposeCmd("-f", comp.YAMLFullPath(), "pause", "svc0").AssertOK()
	base.ComposeCmd("-f", comp.YAMLFullPath(), "ps", "svc0", "-a").
		AssertOutContainsAny("Paused", "paused")
	base.ComposeCmd("-f", comp.YAMLFullPath(), "ps", "svc1").AssertOutContainsAny("Up", "running")

	// unpause should be able to recover the paused service container
	base.ComposeCmd("-f", comp.YAMLFullPath(), "unpause", "svc0").AssertOK()
	base.ComposeCmd("-f", comp.YAMLFullPath(), "ps", "svc0").AssertOutContainsAny("Up", "running")
}
