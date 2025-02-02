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

package emulation

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/containerd/platforms"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"go.farcloser.world/lepton/leptonic/errs"
)

var (
	ErrUnknownOCIArchitecture = errors.New("unknown OCI architecture")
)

func CanExecProbably(pp ...specs.Platform) (bool, error) {
	for _, p := range pp {
		ok, err := canExecProbably(p)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func canExecProbably(p specs.Platform) (bool, error) {
	if platforms.Default().Match(p) {
		return true, nil
	}

	if runtime.GOOS != "linux" {
		return false, nil
	}

	qemuArch, err := ociArch2qemuArch(p.Architecture)
	if err != nil {
		return false, errors.Join(errs.ErrInvalidArgument, err)
	}

	candidates := []string{
		"/proc/sys/fs/binfmt_misc/qemu-" + qemuArch,
		"/proc/sys/fs/binfmt_misc/buildkit-qemu-" + qemuArch,
	}

	// Rosetta 2 for Linux on ARM Mac
	// https://developer.apple.com/documentation/virtualization/running_intel_binaries_in_linux_vms_with_rosetta
	if runtime.GOARCH == "arm64" && p.Architecture == "amd64" {
		candidates = append(candidates, "/proc/sys/fs/binfmt_misc/rosetta")
	}

	for _, cand := range candidates {
		if _, err = os.Stat(cand); err == nil {
			return true, nil
		}
	}

	return false, nil
}

func ociArch2qemuArch(ociArch string) (string, error) {
	switch ociArch {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	case "386":
		return "i386", nil
	case "arm", "s390x", "ppc64le", "riscv64", "mips64":
		return ociArch, nil
	case "mips64le":
		return "mips64el", nil // NOT typo
	case "loong64":
		return "loongarch64", nil // NOT typo
	}

	return "", fmt.Errorf("%w: %q", ErrUnknownOCIArchitecture, ociArch)
}
