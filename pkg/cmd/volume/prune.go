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
	"context"

	containerd "github.com/containerd/containerd/v2/client"

	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/inspecttypes/native"
	"github.com/farcloser/lepton/pkg/labels"
	"github.com/farcloser/lepton/pkg/mountutil/volumestore"
)

func Prune(ctx context.Context, client *containerd.Client, options *types.VolumePruneOptions) (removed []string, cannotRemove []string, errList []error, err error) {
	dataStore, err := clientutil.DataStore(options.GOptions.DataRoot, options.GOptions.Address)
	if err != nil {
		return nil, nil, nil, err
	}

	volStore, err := volumestore.New(dataStore, options.GOptions.Namespace)
	if err != nil {
		return nil, nil, nil, err
	}

	var toRemove []string // nolint: prealloc

	removed, err = volStore.Prune(func(volumes []*native.Volume) ([]string, error) {
		// Get containers and see which volumes are used
		containers, err := client.Containers(ctx)
		if err != nil {
			return nil, err
		}

		usedVolumesList, err := usedVolumes(ctx, containers)
		if err != nil {
			return nil, err
		}

		for _, volume := range volumes {
			if _, ok := usedVolumesList[volume.Name]; ok {
				continue
			}

			if !options.All {
				if volume.Labels == nil {
					continue
				}
				val, ok := (*volume.Labels)[labels.AnonymousVolumes]
				// skip the named volume and only remove the anonymous volume
				if !ok || val != "" {
					continue
				}
			}

			toRemove = append(toRemove, volume.Name)
		}

		// FIXME: @apostasie: implement filters here
		// Needs significant refacto, as we need VolumeNatives instead of just names
		return toRemove, nil
	})

	return removed, nil, nil, err
}
