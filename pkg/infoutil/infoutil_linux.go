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

package infoutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/meminfo"
	"go.farcloser.world/containers/security/apparmor"
	"go.farcloser.world/containers/security/cgroups"
	"go.farcloser.world/containers/sysinfo"

	"github.com/containerd/nerdctl/v2/pkg/defaults"
	"github.com/containerd/nerdctl/v2/pkg/inspecttypes/dockercompat"
	"github.com/containerd/nerdctl/v2/pkg/rootlessutil"
)

const UnameO = "GNU/Linux"

func fulfillSecurityOptions(info *dockercompat.Info) {
	if apparmor.CanApplyExistingProfile() {
		info.SecurityOptions = append(info.SecurityOptions, "name=apparmor")
		if rootlessutil.IsRootless() && !apparmor.CanApplySpecificExistingProfile(defaults.AppArmorProfileName) {
			info.Warnings = append(info.Warnings, fmt.Sprintf(strings.TrimSpace(`
WARNING: AppArmor profile %q is not loaded.
         Use 'sudo nerdctl apparmor load' if you prefer to use AppArmor with rootless mode.
         This warning is negligible if you do not intend to use AppArmor.`), defaults.AppArmorProfileName))
		}
	}
	info.SecurityOptions = append(info.SecurityOptions, "name=seccomp,profile="+defaults.SeccompProfileName)
	if cgroups.DefaultMode() == cgroups.PrivateNsMode {
		info.SecurityOptions = append(info.SecurityOptions, "name=cgroupns")
	}
	if rootlessutil.IsRootlessChild() {
		info.SecurityOptions = append(info.SecurityOptions, "name=rootless")
	}
}

// fulfillPlatformInfo fulfills cgroup and kernel info.
//
// fulfillPlatformInfo requires the following fields to be set:
// SecurityOptions, CgroupDriver, CgroupVersion
func fulfillPlatformInfo(info *dockercompat.Info) {
	fulfillSecurityOptions(info)
	mobySysInfo := mobySysInfo(info)

	if info.CgroupDriver == cgroups.NoneManager {
		if info.CgroupVersion == strconv.Itoa(int(cgroups.Version2)) {
			info.Warnings = append(info.Warnings, "WARNING: Running in rootless-mode without cgroups. Systemd is required to enable cgroups in rootless-mode.")
		} else {
			info.Warnings = append(info.Warnings, "WARNING: Running in rootless-mode without cgroups. To enable cgroups in rootless-mode, you need to boot the system in cgroup v2 mode.")
		}
	} else {
		info.MemoryLimit = mobySysInfo.MemoryLimit
		if !info.MemoryLimit {
			info.Warnings = append(info.Warnings, "WARNING: No memory limit support")
		}
		info.SwapLimit = mobySysInfo.SwapLimit
		if !info.SwapLimit {
			info.Warnings = append(info.Warnings, "WARNING: No swap limit support")
		}
		info.CPUCfsPeriod = mobySysInfo.CPUCfs
		if !info.CPUCfsPeriod {
			info.Warnings = append(info.Warnings, "WARNING: No cpu cfs period support")
		}
		info.CPUCfsQuota = mobySysInfo.CPUCfs
		if !info.CPUCfsQuota {
			info.Warnings = append(info.Warnings, "WARNING: No cpu cfs quota support")
		}
		info.CPUShares = mobySysInfo.CPUShares
		if !info.CPUShares {
			info.Warnings = append(info.Warnings, "WARNING: No cpu shares support")
		}
		info.CPUSet = mobySysInfo.Cpuset
		if !info.CPUSet {
			info.Warnings = append(info.Warnings, "WARNING: No cpuset support")
		}
		info.PidsLimit = mobySysInfo.PidsLimit
		if !info.PidsLimit {
			info.Warnings = append(info.Warnings, "WARNING: No pids limit support")
		}
		info.OomKillDisable = mobySysInfo.OomKillDisable
		if !info.OomKillDisable && info.CgroupVersion == strconv.Itoa(int(cgroups.Version1)) {
			// no warning for cgroup v2
			info.Warnings = append(info.Warnings, "WARNING: No oom kill disable support")
		}
	}
	info.IPv4Forwarding = !mobySysInfo.IPv4ForwardingDisabled
	if !info.IPv4Forwarding {
		info.Warnings = append(info.Warnings, "WARNING: IPv4 forwarding is disabled")
	}
	info.NCPU = sysinfo.NumCPU()
	memLimit, err := meminfo.Read()
	if err != nil {
		info.Warnings = append(info.Warnings, fmt.Sprintf("failed to read mem info: %v", err))
	} else {
		info.MemTotal = memLimit.MemTotal
	}
}

func mobySysInfo(info *dockercompat.Info) *sysinfo.SysInfo {
	g := "/"
	if info.CgroupDriver == cgroups.SystemdManager && info.CgroupVersion == strconv.Itoa(int(cgroups.Version2)) && rootlessutil.IsRootless() {
		g = fmt.Sprintf("/user.slice/user-%d.slice", rootlessutil.ParentEUID())
	}
	inf, _, _ := sysinfo.New(g)
	// FIXME: log warnings and errors
	return inf
}
