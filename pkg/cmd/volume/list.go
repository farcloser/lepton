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
	"strings"

	"github.com/containerd/log"

	"github.com/farcloser/lepton/pkg/api/types"
	"github.com/farcloser/lepton/pkg/inspecttypes/native"
)

func List(options *types.VolumeListOptions) (map[string]native.Volume, error) {
	if options.Quiet && options.Size {
		log.L.Warn("cannot use --size and --quiet together, ignoring --size")
		options.Size = false
	}
	sizeFilter := hasSizeFilter(options.Filters)
	if sizeFilter && options.Quiet {
		log.L.Warn("cannot use --filter=size and --quiet together, ignoring --filter=size")
		options.Filters = removeSizeFilters(options.Filters)
	}
	if sizeFilter && !options.Size {
		log.L.Warn("should use --filter=size and --size together")
		options.Size = true
	}

	return Volumes(
		options.GOptions.Namespace,
		options.GOptions.DataRoot,
		options.GOptions.Address,
		options.Size,
		options.Filters,
	)
}

func hasSizeFilter(filters []string) bool {
	for _, filter := range filters {
		if strings.HasPrefix(filter, "size") {
			return true
		}
	}
	return false
}

func removeSizeFilters(filters []string) []string {
	var res []string
	for _, filter := range filters {
		if !strings.HasPrefix(filter, "size") {
			res = append(res, filter)
		}
	}
	return res
}

// Volumes returns volumes that match the given filters.
//
// Supported filters:
//   - label=<key>=<value>: Match volumes by label on both key and value.
//     If value is left empty, match all volumes with key regardless of its value.
//   - name=<value>: Match all volumes with a name containing the value string.
//   - size=<value>: Match all volumes with a size meets the value.
//     Size operand can be >=, <=, >, <, = and value must be an integer.
//
// Unsupported filters:
//   - dangling=true: Filter volumes by dangling.
//   - driver=local: Filter volumes by driver.
func Volumes(ns string, dataRoot string, address string, volumeSize bool, filters []string) (map[string]native.Volume, error) {
	volumeStore, err := Store(ns, dataRoot, address)
	if err != nil {
		return nil, err
	}
	vols, err := volumeStore.List(volumeSize)
	if err != nil {
		return nil, err
	}

	return filter(vols, filters)
}
