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

package system_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/infoutil"
	"go.farcloser.world/lepton/pkg/inspecttypes/dockercompat"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func testInfoComparator(stdout, info string, t *testing.T) {
	var dinf dockercompat.Info
	err := json.Unmarshal([]byte(stdout), &dinf)
	assert.NilError(t, err, "failed to unmarshal stdout"+info)
	unameM := infoutil.UnameM()
	assert.Assert(
		t,
		dinf.Architecture == unameM,
		fmt.Sprintf("expected info.Architecture to be %q, got %q", unameM, dinf.Architecture)+info,
	)
}

func TestInfo(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.SubTests = []*test.Case{
		{
			Description: "info",
			Command:     test.Command("info", "--format", "{{json .}}"),
			Expected:    test.Expects(expect.ExitCodeSuccess, nil, testInfoComparator),
		},
		{
			Description: "info convenience form",
			Command:     test.Command("info", "--format", formatter.FormatJSON),
			Expected:    test.Expects(expect.ExitCodeSuccess, nil, testInfoComparator),
		},
		{
			Description: "info with namespace",
			Require:     require.Not(nerdtest.Docker),
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Custom(nerdtest.Binary(), "info")
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("Namespace:	default")),
		},
		{
			Description: "info with namespace env var",
			Env: map[string]string{
				"CONTAINERD_NAMESPACE": "test",
			},
			Require: require.Not(nerdtest.Docker),
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Custom(nerdtest.Binary(), "info")
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, expect.Contains("Namespace:	test")),
		},
	}

	testCase.Run(t)
}
