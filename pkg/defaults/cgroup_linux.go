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

package defaults

import (
	"os"

	"go.farcloser.world/containers/cgroups"
)

func IsSystemdAvailable() bool {
	fi, err := os.Lstat("/run/systemd/system")
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// CgroupManager defaults to "systemd"  on v2 (rootful & rootless), "none" otherwise
func CgroupManager() string {
	if cgroups.Version() == cgroups.Version2 && IsSystemdAvailable() {
		return "systemd"
	}
	return "none"
}

func CgroupnsMode() string {
	if cgroups.Version() == cgroups.Version2 {
		return "private"
	}
	return "host"
}
