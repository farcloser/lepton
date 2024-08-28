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
	"errors"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"

	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/clientutil"
	"github.com/farcloser/lepton/pkg/errs"
	"github.com/farcloser/lepton/pkg/mountutil/volumestore"
)

func Remove(ctx context.Context, client *containerd.Client, volumes []string, options *types.VolumeRemoveOptions) (removed []string, cannotRemove []string, errList []error, err error) {
	dataStore, err := clientutil.DataStore(options.GOptions.DataRoot, options.GOptions.Address)
	if err != nil {
		return nil, volumes, nil, err
	}

	volStore, err := volumestore.New(dataStore, options.GOptions.Namespace)
	if err != nil {
		return nil, volumes, nil, err
	}

	removed, notRemoved, warns, err := volStore.Remove(func() ([]string, error) {
		containers, err := client.Containers(ctx)
		if err != nil {
			return nil, err
		}

		usedVolumesList, err := usedVolumes(ctx, containers)
		if err != nil {
			return nil, err
		}

		toRemove := []string{}
		for _, name := range volumes {
			if cid, ok := usedVolumesList[name]; ok {
				cannotRemove = append(cannotRemove, name)
				errList = append(errList, errors.Join(errs.ErrFailedPrecondition, fmt.Errorf("volume %q is in use by container %q", name, cid)))
				continue
			}
			toRemove = append(toRemove, name)
		}

		return toRemove, nil
	})

	cannotRemove = append(cannotRemove, notRemoved...)

	return removed, cannotRemove, append(errList, warns...), err
}
