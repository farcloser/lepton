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
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/pkg/cap"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/log"

	"go.farcloser.world/containers/security/seccomp"
	"go.farcloser.world/containers/specs"

	"go.farcloser.world/lepton/leptonic/services/apparmor"
	"go.farcloser.world/lepton/pkg/defaults"
	"go.farcloser.world/lepton/pkg/maputil"
	"go.farcloser.world/lepton/pkg/strutil"
)

var privilegedOpts = []oci.SpecOpts{
	oci.WithPrivileged,
	oci.WithAllDevicesAllowed,
	oci.WithHostDevices,
	oci.WithNewPrivileges,
}

var privilegedWithoutDevicesOpts = []oci.SpecOpts{
	oci.WithPrivileged,
	oci.WithNewPrivileges,
}

const (
	systemPathsUnconfined = "unconfined"
)

func seccompWithDefaultProfile() oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		seccomp.LoadDefaultProfile(s)
		return nil
	}
}

func seccompWithProfile(profile string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) (err error) {
		return seccomp.LoadProfile(s, profile)
	}
}

func generateSecurityOpts(privileged bool, securityOptsMap map[string]string) ([]oci.SpecOpts, error) {
	for k := range securityOptsMap {
		switch k {
		case "seccomp", "apparmor", "no-new-privileges", "systempaths", "privileged-without-host-devices":
		default:
			log.L.Warnf("unknown security-opt: %q", k)
		}
	}
	var opts []oci.SpecOpts
	if seccompProfile, ok := securityOptsMap["seccomp"]; ok && seccompProfile != defaults.SeccompProfileName {
		if seccompProfile == "" {
			return nil, errors.New("invalid security-opt \"seccomp\"")
		}

		if seccompProfile != "unconfined" {
			opts = append(opts, seccompWithProfile(seccompProfile))
		}
	} else {
		opts = append(opts, seccompWithDefaultProfile())
	}

	profile := defaults.AppArmorProfileName
	explicitProfile := false
	if aProfile, ok := securityOptsMap["apparmor"]; ok {
		profile = aProfile
		explicitProfile = true
	}

	appArmorSpecs, err := apparmor.GetSpecOptions(profile)
	// If we failed with an explicit --security-opt, hard error
	if errors.Is(err, apparmor.ErrUnsupported) || errors.Is(err, apparmor.ErrCannotApply) {
		if explicitProfile {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if appArmorSpecs != nil {
		opts = append(opts, appArmorSpecs)
	}

	nnp, err := maputil.MapBoolValueAsOpt(securityOptsMap, "no-new-privileges")
	if err != nil {
		return nil, err
	}

	if !nnp {
		opts = append(opts, oci.WithNewPrivileges)
	}

	if value, ok := securityOptsMap["systempaths"]; ok && value == systemPathsUnconfined {
		opts = append(opts, oci.WithMaskedPaths(nil))
		opts = append(opts, oci.WithReadonlyPaths(nil))
	} else if ok && value != systemPathsUnconfined {
		return nil, errors.New(`invalid security-opt "systempaths=unconfined"`)
	}

	privilegedWithoutHostDevices, err := maputil.MapBoolValueAsOpt(securityOptsMap, "privileged-without-host-devices")
	if err != nil {
		return nil, err
	}

	if privilegedWithoutHostDevices && !privileged {
		return nil, errors.New(
			"flag `--security-opt privileged-without-host-devices` can't be used without `--privileged` enabled",
		)
	}

	if privileged {
		if privilegedWithoutHostDevices {
			opts = append(opts, privilegedWithoutDevicesOpts...)
		} else {
			opts = append(opts, privilegedOpts...)
		}
	}

	return opts, nil
}

func canonicalizeCapName(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToUpper(s)
	if !strings.HasPrefix(s, "CAP_") {
		s = "CAP_" + s
	}
	if !isKnownCapName(s) {
		log.L.Warnf("unknown capability name %q", s)
		// Not a fatal error, because runtime might be aware of this cap
	}
	return s
}

var (
	knownCapNames     map[string]struct{}
	knownCapNamesOnce sync.Once
)

func isKnownCapName(s string) bool {
	knownCapNamesOnce.Do(func() {
		known := cap.Known()
		knownCapNames = make(map[string]struct{}, len(known))
		for _, f := range known {
			knownCapNames[f] = struct{}{}
		}
	})
	_, ok := knownCapNames[s]
	return ok
}

func generateCapOpts(capAdd, capDrop []string) []oci.SpecOpts {
	if len(capAdd) == 0 && len(capDrop) == 0 {
		return nil
	}

	var opts []oci.SpecOpts
	if strutil.InStringSlice(capDrop, "ALL") {
		opts = append(opts, oci.WithCapabilities(nil))
	}

	if strutil.InStringSlice(capAdd, "ALL") {
		opts = append(opts, oci.WithAllCurrentCapabilities)
	} else {
		var capsAdd []string
		for _, c := range capAdd {
			capsAdd = append(capsAdd, canonicalizeCapName(c))
		}
		opts = append(opts, oci.WithAddedCapabilities(capsAdd))
	}

	if !strutil.InStringSlice(capDrop, "ALL") {
		var capsDrop []string
		for _, c := range capDrop {
			capsDrop = append(capsDrop, canonicalizeCapName(c))
		}
		opts = append(opts, oci.WithDroppedCapabilities(capsDrop))
	}
	return opts
}
