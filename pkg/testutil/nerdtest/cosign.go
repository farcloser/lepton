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

package nerdtest

import (
	"os"
	"path/filepath"

	"gotest.tools/v3/assert"

	"go.farcloser.world/tigron/test"
)

type CosignKeyPair struct {
	PublicKey  string
	PrivateKey string
}

func NewCosignKeyPair(data test.Data, helpers test.Helpers) *CosignKeyPair {
	dir, err := os.MkdirTemp(data.TempDir(), "cosign")
	assert.NilError(helpers.T(), err)

	cmd := helpers.Custom("cosign", "generate-key-pair")
	cmd.WithCwd(dir)
	cmd.Run(&test.Expected{
		ExitCode: 0,
	})

	return &CosignKeyPair{
		PublicKey:  filepath.Join(dir, "cosign.pub"),
		PrivateKey: filepath.Join(dir, "cosign.key"),
	}
}
