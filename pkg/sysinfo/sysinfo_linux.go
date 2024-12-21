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

/*
   Portions from https://github.com/moby/moby/blob/cff4f20c44a3a7c882ed73934dec6a77246c6323/pkg/sysinfo/sysinfo_linux.go
   Copyright (C) Docker/Moby authors.
   Licensed under the Apache License, Version 2.0
   NOTICE: https://github.com/moby/moby/blob/cff4f20c44a3a7c882ed73934dec6a77246c6323/NOTICE
*/

package sysinfo // import "github.com/docker/docker/pkg/sysinfo"

import (
	"os"
	"path"
	"strings"

	"go.farcloser.world/containers/cgroups"

	"github.com/containerd/containerd/v2/pkg/seccomp"
)

type infoCollector func(info *SysInfo)

// WithCgroup2GroupPath specifies the cgroup v2 group path to inspect availability
// of the controllers.
//
// WithCgroup2GroupPath is expected to be used for rootless mode with systemd driver.
//
// e.g. g = "/user.slice/user-1000.slice/user@1000.service"
func WithCgroup2GroupPath(g string) Opt {
	return func(o *SysInfo) {
		if p := path.Clean(g); p != "" {
			o.cg2GroupPath = p
		}
	}
}

// New returns a new SysInfo, using the filesystem to detect which features
// the kernel supports.
func New(options ...Opt) *SysInfo {
	if cgroups.Version() == cgroups.Version2 {
		return newV2(options...)
	}
	return nil
}

// applyNetworkingInfo adds networking information to the info.
func applyNetworkingInfo(info *SysInfo) {
	info.IPv4ForwardingDisabled = !readProcBool("/proc/sys/net/ipv4/ip_forward")
}

// applyAppArmorInfo adds whether AppArmor is enabled to the info.
func applyAppArmorInfo(info *SysInfo) {
	if _, err := os.Stat("/sys/kernel/security/apparmor"); !os.IsNotExist(err) {
		if _, err := os.ReadFile("/sys/kernel/security/apparmor/profiles"); err == nil {
			info.AppArmor = true
		}
	}
}

// applyCgroupNsInfo adds whether cgroupns is enabled to the info.
func applyCgroupNsInfo(info *SysInfo) {
	if _, err := os.Stat("/proc/self/ns/cgroup"); !os.IsNotExist(err) {
		info.CgroupNamespaces = true
	}
}

// applySeccompInfo checks if Seccomp is supported, via CONFIG_SECCOMP.
func applySeccompInfo(info *SysInfo) {
	info.Seccomp = seccomp.IsEnabled()
}

func cgroupEnabled(mountPoint, name string) bool {
	_, err := os.Stat(path.Join(mountPoint, name))
	return err == nil
}

func readProcBool(path string) bool {
	val, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(val)) == "1"
}
