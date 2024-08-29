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

package container

import (
	"strings"
	"testing"

	"gotest.tools/v3/icmd"

	"github.com/farcloser/lepton/pkg/testutil"
)

/*
This test below is meant to assert that https://github.com/containerd/nerdctl/issues/827 is NOT fixed.
Obviously, once we fix the issue, it should be replaced by something that assert it works.
Unfortunately, this is flaky.
It will regularly succeed or fail, making random PR fail the Kube check.
*/

func TestKubeCommitPush(t *testing.T) {
	t.Parallel()

	t.Skip("Test that confirm that #827 is still broken is too flaky")

	base := testutil.NewBaseForKubernetes(t)
	tID := testutil.Identifier(t)

	var containerID string

	setup := func() {
		testutil.KubectlHelper(base, "run", "--image", testutil.CommonImage, tID, "--", "sleep", "Inf").
			AssertOK()

		testutil.KubectlHelper(base, "wait", "pod", tID, "--for=condition=ready", "--timeout=1m").
			AssertOK()

		testutil.KubectlHelper(base, "exec", tID, "--", "mkdir", "-p", "/tmp/whatever").
			AssertOK()

		cmd := testutil.KubectlHelper(base, "get", "pods", tID, "-o", "jsonpath={ .status.containerStatuses[0].containerID }")
		cmd.Run()
		containerID = strings.TrimPrefix(cmd.Out(), "containerd://")
	}

	tearDown := func() {
		testutil.KubectlHelper(base, "delete", "pod", "-f", tID).Run()
	}

	tearDown()
	t.Cleanup(tearDown)
	setup()

	t.Run("test commit / push on Kube (https://github.com/containerd/nerdctl/issues/827)", func(t *testing.T) {
		t.Log("This test is meant to verify that we can commit / push an image from a pod." +
			"Currently, this is broken, hence the test assumes it will fail. Once the problem is fixed, we should just" +
			"change the expectation to 'success'.")

		base.Cmd("commit", containerID, "registry.example.com/my-app:v1").AssertOK()
		// See note above.
		base.Cmd("push", "registry.example.com/my-app:v1").Assert(icmd.Expected{
			ExitCode: 1,
			Err:      "failed to create a tmp single-platform image",
		})
	})
}
