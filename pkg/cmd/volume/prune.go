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
	"context"
	"fmt"
	"io"
	"strings"

	containerd "github.com/containerd/containerd/v2/client"

	"go.farcloser.world/lepton/leptonic/api"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/labels"
)

func Prune(ctx context.Context, client *containerd.Client, output io.Writer, globalOptions *options.Global, opts *options.VolumePrune) error {
	// Get the volume store and lock it until we are done.
	// This will prevent racing new containers from being created or removed until we are done with the cleanup of volumes
	volStore, err := Store(globalOptions.Namespace, globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return err
	}

	var toRemove []string

	err = volStore.Prune(func(volumes []*api.Volume) ([]string, error) {
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
			if !opts.All {
				if volume.Labels == nil {
					continue
				}
				val, ok := volume.Labels[labels.AnonymousVolumes]
				// skip the named volume and only remove the anonymous volume
				if !ok || val != "" {
					continue
				}
			}
			toRemove = append(toRemove, volume.Name)
		}

		return toRemove, nil
	})

	if err != nil {
		return err
	}

	if len(toRemove) > 0 {
		fmt.Fprintln(output, "Deleted Volumes:")
		fmt.Fprintln(output, strings.Join(toRemove, "\n"))
		fmt.Fprintln(output, "")
	}

	return nil
}
