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

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRootJoin(t *testing.T) {
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, "somedir"), 0755)
	os.WriteFile(filepath.Join(base, "somefile"), []byte(""), 0644)
	os.Symlink("nonexistent", filepath.Join(base, "link-rel-nonexist-local"))
	os.Symlink("../nonexistent", filepath.Join(base, "link-rel-nonexist-non-local"))

	res, err := RootJoin(base, "somedir")
	assert.NilError(t, err)
	assert.Equal(t, res, filepath.Join(base, "somedir"))

	res, err = RootJoin(base, "..", filepath.Base(base), "somedir")
	assert.NilError(t, err)
	assert.Equal(t, res, filepath.Join(base, "somedir"))
}
