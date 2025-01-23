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

package apparmor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/moby/sys/userns"

	"github.com/containerd/containerd/v2/contrib/apparmor"
	"github.com/containerd/containerd/v2/pkg/oci"
)

var (
	appArmorSupported bool
	checkAppArmor     sync.Once

	paramEnabled     bool
	paramEnabledOnce sync.Once
)

func LoadDefaultProfile(name string) error {
	return apparmor.LoadDefaultProfile(name)
}

func DumpDefaultProfile(name string) (string, error) {
	return apparmor.DumpDefaultProfile(name)
}

func WithProfile(name string) oci.SpecOpts {
	return apparmor.WithProfile(name)
}

// CanApplySpecificExistingProfile attempts to run `aa-exec -p <NAME> -- true` to check whether
// the profile can be applied.
//
// CanApplySpecificExistingProfile does NOT depend on /sys/kernel/security/apparmor/profiles ,
// which might not be accessible from user namespaces (because securityfs cannot be mounted in a user namespace)
func CanApplySpecificExistingProfile(profileName string) bool {
	if !CanApplyExistingProfile() {
		return false
	}
	cmd := exec.Command("aa-exec", "-p", profileName, "--", "true")
	_, err := cmd.CombinedOutput()
	return err == nil
}

// CanLoadNewProfile returns whether the current process can load a new AppArmor profile.
//
// CanLoadNewProfile needs root.
//
// CanLoadNewProfile checks both /sys/module/apparmor/parameters/enabled and /sys/kernel/security.
//
// Related: https://gitlab.com/apparmor/apparmor/-/blob/v3.0.3/libraries/libapparmor/src/kernel.c#L311
func CanLoadNewProfile() bool {
	return !userns.RunningInUserNS() && os.Geteuid() == 0 && hostSupports()
}

// CanApplyExistingProfile returns whether the current process can apply an existing AppArmor profile
// to processes.
//
// CanApplyExistingProfile does NOT need root.
//
// CanApplyExistingProfile checks /sys/module/apparmor/parameters/enabled ,but does NOT check /sys/kernel/security/apparmor ,
// which might not be accessible from user namespaces (because securityfs cannot be mounted in a user namespace)
//
// Related: https://gitlab.com/apparmor/apparmor/-/blob/v3.0.3/libraries/libapparmor/src/kernel.c#L311
func CanApplyExistingProfile() bool {
	paramEnabledOnce.Do(func() {
		buf, err := os.ReadFile("/sys/module/apparmor/parameters/enabled")
		paramEnabled = err == nil && len(buf) > 1 && buf[0] == 'Y'
	})
	return paramEnabled
}

// Unload unloads a profile. Needs access to /sys/kernel/security/apparmor/.remove .
func Unload(target string) error {
	// FIXME: not safe
	remover, err := os.OpenFile("/sys/kernel/security/apparmor/.remove", os.O_RDWR|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := remover.WriteString(target); err != nil {
		remover.Close()
		return err
	}
	return remover.Close()
}

type Profile struct {
	Name string `json:"Name"`           // e.g., "prefix-default"
	Mode string `json:"Mode,omitempty"` // e.g., "enforce"
}

// Profiles return profiles.
//
// Profiles does not need the root but needs access to /sys/kernel/security/apparmor/policy/profiles,
// which might not be accessible from user namespaces (because securityfs cannot be mounted in a user namespace)
//
// So, Profiles cannot be called from rootless child.
func Profiles() ([]Profile, error) {
	// FIXME: not safe
	const profilesPath = "/sys/kernel/security/apparmor/policy/profiles"
	ents, err := os.ReadDir(profilesPath)
	if err != nil {
		return nil, err
	}
	res := make([]Profile, len(ents))
	for i, ent := range ents {
		namePath := filepath.Join(profilesPath, ent.Name(), "name")
		b, err := os.ReadFile(namePath)
		if err != nil {
			// log.L.WithError(err).Warnf("failed to read %q", namePath)
			continue
		}
		profile := Profile{
			Name: strings.TrimSpace(string(b)),
		}
		modePath := filepath.Join(profilesPath, ent.Name(), "mode")
		b, err = os.ReadFile(modePath)
		if err == nil {
			profile.Mode = strings.TrimSpace(string(b))
		}
		res[i] = profile
	}
	return res, nil
}

// hostSupports returns true if apparmor is enabled for the host, if
// apparmor_parser is enabled, and if we are not running docker-in-docker.
//
// This is derived from libcontainer/apparmor.IsEnabled(), with the addition
// of checks for apparmor_parser to be present and docker-in-docker.
func hostSupports() bool {
	checkAppArmor.Do(func() {
		// see https://github.com/opencontainers/runc/blob/0d49470392206f40eaab3b2190a57fe7bb3df458/libcontainer/apparmor/apparmor_linux.go
		if _, err := os.Stat("/sys/kernel/security/apparmor"); err == nil && os.Getenv("container") == "" {
			if _, err = os.Stat("/sbin/apparmor_parser"); err == nil {
				buf, err := os.ReadFile("/sys/module/apparmor/parameters/enabled")
				appArmorSupported = err == nil && len(buf) > 1 && buf[0] == 'Y'
			}
		}
	})
	return appArmorSupported
}
