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

package volume

import (
	"github.com/docker/docker/pkg/stringid"

	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/inspecttypes/native"
	"github.com/farcloser/lepton/pkg/labels"
	"github.com/farcloser/lepton/pkg/mountutil/volumestore"
	"github.com/farcloser/lepton/pkg/strutil"
)

func Create(name string, options *types.VolumeCreateOptions) (*native.Volume, error) {
	dataStore, err := clientutil.DataStore(options.GOptions.DataRoot, options.GOptions.Address)
	if err != nil {
		return nil, err
	}

	volumeStore, err := volumestore.New(dataStore, options.GOptions.Namespace)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = stringid.GenerateRandomID()
		options.Labels = append(options.Labels, labels.AnonymousVolumes+"=")
	}

	vol, err := volumeStore.Create(name, strutil.DedupeStrSlice(options.Labels))
	if err != nil {
		return nil, err
	}

	return vol, nil
}
