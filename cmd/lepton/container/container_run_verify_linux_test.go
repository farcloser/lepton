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
	"testing"

	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest/registry"
	"go.farcloser.world/lepton/pkg/testutil/various"
)

func TestRunVerifyCosign(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = require.All(
		require.Not(nerdtest.Docker),
		nerdtest.Build,
		nerdtest.Registry,
		require.Binary("cosign"),
	)

	testCase.Env = map[string]string{
		"COSIGN_PASSWORD": "1",
	}

	var reg *registry.Server
	var keyPair *various.CosignKeyPair

	testCase.Setup = func(data test.Data, helpers test.Helpers) {
		reg = nerdtest.RegistryWithNoAuth(data, helpers, 0, false)
		reg.Setup(data, helpers)
		keyPair = various.NewCosignKeyPair(t, "cosign-key-pair", "1")
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		if reg != nil {
			reg.Cleanup(data, helpers)
			keyPair.Cleanup()
		}
	}

	testCase.Command = func(data test.Data, helpers test.Helpers) test.TestableCommand {
		tID := data.Identifier()
		testImageRef := fmt.Sprintf("127.0.0.1:%d/%s", reg.Port, tID)
		dockerfile := fmt.Sprintf(`FROM %s
CMD ["echo", "build-test-string"]
	`, testutil.CommonImage)
		buildCtx := various.CreateBuildContext(t, dockerfile)

		helpers.Ensure("build", "-t", testImageRef, buildCtx)
		helpers.Ensure("push", testImageRef, "--sign=cosign", "--cosign-key="+keyPair.PrivateKey)
		helpers.Ensure("run", "--rm", "--verify=cosign", "--cosign-key="+keyPair.PublicKey, testImageRef)

		return helpers.Command("run", "--rm", "--verify=cosign", "--cosign-key=dummy", testImageRef)
	}

	testCase.Expected = test.Expects(1, nil, nil)

	testCase.Run(t)
}
