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

package volume

import (
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/mountutil/volumestore"
)

// Store returns a volume store
// that corresponds to a directory like `/var/lib/<ROOT_NAME>/1935db59/volumes/default`
func Store(ns, dataRoot, address string) (volumestore.VolumeService, error) {
	dataStore, err := clientutil.DataStore(dataRoot, address)
	if err != nil {
		return nil, err
	}
	return volumestore.New(dataStore, ns)
}
