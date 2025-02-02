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

package container

import (
	"fmt"
	"os/exec"
	"testing"

	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/testregistry"
	helpers2 "go.farcloser.world/lepton/pkg/testutil/various"
)

func TestRunVerifyCosign(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = test.Require(
		test.Not(nerdtest.Docker),
		nerdtest.Build,
		&test.Requirement{
			Check: func(data test.Data, helpers test.Helpers) (ret bool, mess string) {
				ret = true
				mess = "cosign is in the path"
				_, err := exec.LookPath("cosign")
				if err != nil {
					ret = false
					mess = fmt.Sprintf("cosign is not in the path: %+v", err)
					return ret, mess
				}
				return ret, mess
			},
		},
	)

	testCase.Env = map[string]string{
		"COSIGN_PASSWORD": "1",
	}

	var keyPair *helpers2.CosignKeyPair
	var reg *testregistry.RegistryServer

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		keyPair = helpers2.NewCosignKeyPair(t, "cosign-key-pair", "1")
		reg = testregistry.NewWithNoAuth(data, helpers, 0, false)
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		if keyPair != nil {
			keyPair.Cleanup()
			reg.Cleanup(nil)
		}
	}

	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		tID := data.Identifier()
		testImageRef := fmt.Sprintf("127.0.0.1:%d/%s", reg.Port, tID)
		dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "build-test-string"]
	`, testutil.CommonImage)

		buildCtx := helpers2.CreateBuildContext(t, dockerfile)

		helpers.Ensure("build", "-t", testImageRef, buildCtx)
		helpers.Ensure("push", testImageRef, "--sign=cosign", "--cosign-key="+keyPair.PrivateKey)
		helpers.Ensure("run", "--rm", "--verify=cosign", "--cosign-key="+keyPair.PublicKey, testImageRef)
		return helpers.Command("run", "--rm", "--verify=cosign", "--cosign-key=dummy", testImageRef)
	}

	testCase.Expected = test.Expects(1, nil, nil)

	testCase.Run(t)
}
