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
	"github.com/containerd/containerd/v2/pkg/cap"

	"go.farcloser.world/containers/specs"
)

func setExecCapabilities(pspec *specs.Process) error {
	if pspec.Capabilities == nil {
		pspec.Capabilities = &specs.LinuxCapabilities{}
	}
	allCaps, err := cap.Current()
	if err != nil {
		return err
	}
	pspec.Capabilities.Bounding = allCaps
	pspec.Capabilities.Permitted = pspec.Capabilities.Bounding
	pspec.Capabilities.Inheritable = pspec.Capabilities.Bounding
	pspec.Capabilities.Effective = pspec.Capabilities.Bounding

	// https://github.com/moby/moby/pull/36466/files
	// > `docker exec --privileged` does not currently disable AppArmor
	// > profiles. Privileged configuration of the container is inherited
	return nil
}
