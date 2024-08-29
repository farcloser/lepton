/*
   Copyright The containerd Authors.

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

package image

import (
	"fmt"
	"testing"
	"time"

	"github.com/farcloser/lepton/cmd/lepton/helpers"
	"github.com/farcloser/lepton/pkg/testutil"
)

func TestImagePrune(t *testing.T) {
	testutil.RequiresBuild(t)
	testutil.RegisterBuildCacheCleanup(t)

	base := testutil.NewBase(t)
	imageName := testutil.Identifier(t)
	defer base.Cmd("rmi", imageName).AssertOK()

	dockerfile := fmt.Sprintf(`FROM %s
	CMD ["echo", "nerdctl-test-image-prune"]`, testutil.CommonImage)

	buildCtx := helpers.CreateBuildContext(t, dockerfile)

	base.Cmd("build", buildCtx).AssertOK()
	base.Cmd("build", "-t", imageName, buildCtx).AssertOK()
	base.Cmd("images").AssertOutContainsAll(imageName, "<none>")

	base.Cmd("image", "prune", "--force").AssertOutNotContains(imageName)
	base.Cmd("images").AssertOutNotContains("<none>")
	base.Cmd("images").AssertOutContains(imageName)
}

func TestImagePruneAll(t *testing.T) {
	testutil.RequiresBuild(t)
	testutil.RegisterBuildCacheCleanup(t)

	base := testutil.NewBase(t)
	imageName := testutil.Identifier(t)

	dockerfile := fmt.Sprintf(`FROM %s
	CMD ["echo", "nerdctl-test-image-prune"]`, testutil.CommonImage)

	buildCtx := helpers.CreateBuildContext(t, dockerfile)

	base.Cmd("build", "-t", imageName, buildCtx).AssertOK()
	// The following commands will clean up all images, so it should fail at this point.
	defer base.Cmd("rmi", imageName).AssertFail()
	base.Cmd("images").AssertOutContains(imageName)

	tID := testutil.Identifier(t)
	base.Cmd("run", "--name", tID, imageName).AssertOK()
	base.Cmd("image", "prune", "--force", "--all").AssertOutNotContains(imageName)
	base.Cmd("images").AssertOutContains(imageName)

	base.Cmd("rm", "-f", tID).AssertOK()
	base.Cmd("image", "prune", "--force", "--all").AssertOutContains(imageName)
	base.Cmd("images").AssertOutNotContains(imageName)
}

func TestImagePruneFilterLabel(t *testing.T) {
	testutil.RequiresBuild(t)
	testutil.RegisterBuildCacheCleanup(t)

	base := testutil.NewBase(t)
	imageName := testutil.Identifier(t)
	t.Cleanup(func() { base.Cmd("rmi", "--force", imageName) })

	dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "nerdctl-test-image-prune-filter-label"]
LABEL foo=bar
LABEL version=0.1`, testutil.CommonImage)

	buildCtx := helpers.CreateBuildContext(t, dockerfile)

	base.Cmd("build", "-t", imageName, buildCtx).AssertOK()
	base.Cmd("images", "--all").AssertOutContains(imageName)

	base.Cmd("image", "prune", "--force", "--all", "--filter", "label=foo=baz").AssertOK()
	base.Cmd("images", "--all").AssertOutContains(imageName)

	base.Cmd("image", "prune", "--force", "--all", "--filter", "label=foo=bar").AssertOK()
	base.Cmd("images", "--all").AssertOutNotContains(imageName)
}

func TestImagePruneFilterUntil(t *testing.T) {
	testutil.RequiresBuild(t)
	testutil.RegisterBuildCacheCleanup(t)

	base := testutil.NewBase(t)
	// For deterministically testing the filter, set the image's created timestamp to 2 hours in the past.
	base.Env = append(base.Env, fmt.Sprintf("SOURCE_DATE_EPOCH=%d", time.Now().Add(-2*time.Hour).Unix()))

	imageName := testutil.Identifier(t)
	teardown := func() {
		// Image should have been pruned; but cleanup on failure.
		base.Cmd("rmi", "--force", imageName).Run()
	}
	t.Cleanup(teardown)
	teardown()

	dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "nerdctl-test-image-prune-filter-until"]`, testutil.CommonImage)

	buildCtx := helpers.CreateBuildContext(t, dockerfile)

	base.Cmd("build", "-t", imageName, buildCtx).AssertOK()
	base.Cmd("images", "--all").AssertOutContains(imageName)

	base.Cmd("image", "prune", "--force", "--all", "--filter", "until=12h").AssertOK()
	base.Cmd("images", "--all").AssertOutContains(imageName)

	// Pause to ensure enough time has passed for the image to be cleaned on next prune.
	time.Sleep(3 * time.Second)

	base.Cmd("image", "prune", "--force", "--all", "--filter", "until=10ms").AssertOK()
	base.Cmd("images", "--all").AssertOutNotContains(imageName)
}
