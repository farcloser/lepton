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

package container_test

import (
	"testing"

	"go.farcloser.world/lepton/pkg/testutil"
)

func TestUpdateContainer(t *testing.T) {
	t.Parallel()

	testutil.DockerIncompatible(t)
	testContainerName := testutil.Identifier(t)
	base := testutil.NewBase(t)
	base.Cmd("run", "-d", "--name", testContainerName, testutil.CommonImage, "sleep", "infinity").
		AssertOK()
	defer base.Cmd("rm", "-f", testContainerName).Run()
	base.Cmd("update", "--memory", "999999999", "--restart", "123", testContainerName).AssertFail()
	base.Cmd("inspect", "--mode=native", testContainerName).
		AssertOutNotContains(`"limit": 999999999,`)
}
