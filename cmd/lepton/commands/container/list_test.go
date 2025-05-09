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
	"fmt"
	"testing"

	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

// https://github.com/containerd/nerdctl/issues/2598
func TestContainerListWithFormatLabel(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	tID := testutil.Identifier(t)
	cID := tID
	labelK := "label-key-" + tID
	labelV := "label-value-" + tID

	base.Cmd("run", "-d",
		"--name", cID,
		"--label", labelK+"="+labelV,
		testutil.CommonImage, "sleep", nerdtest.Infinity).AssertOK()
	defer base.Cmd("rm", "-f", cID).AssertOK()
	base.Cmd("ps", "-a",
		"--filter", "label="+labelK,
		"--format", fmt.Sprintf("{{.Label %q}}", labelK)).AssertOutExactly(labelV + "\n")
}

func TestContainerListWithJsonFormatLabel(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	tID := testutil.Identifier(t)
	cID := tID
	labelK := "label-key-" + tID
	labelV := "label-value-" + tID

	base.Cmd("run", "-d",
		"--name", cID,
		"--label", labelK+"="+labelV,
		testutil.CommonImage, "sleep", nerdtest.Infinity).AssertOK()
	defer base.Cmd("rm", "-f", cID).AssertOK()
	base.Cmd("ps", "-a",
		"--filter", "label="+labelK,
		"--format", formatter.FormatJSON).AssertOutContains(fmt.Sprintf("%s=%s", labelK, labelV))
}
