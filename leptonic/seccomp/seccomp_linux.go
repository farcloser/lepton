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

package seccomp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"go.farcloser.world/containers/specs"

	"github.com/containerd/containerd/v2/contrib/seccomp"
)

var (
	ErrCannotLoadProfile   = errors.New("cannot load seccomp profile")
	ErrCannotDecodeProfile = errors.New("cannot decode seccomp profile")
)

func LoadProfile(s *specs.Spec, profile string) error {
	s.Linux.Seccomp = &specs.LinuxSeccomp{}

	f, err := os.ReadFile(profile)
	if err != nil {
		return errors.Join(fmt.Errorf("%w %q", ErrCannotLoadProfile, profile), err)
	}

	if err = json.Unmarshal(f, s.Linux.Seccomp); err != nil {
		return errors.Join(fmt.Errorf("%w %q", ErrCannotDecodeProfile, profile), err)
	}

	return nil
}

func LoadDefaultProfile(s *specs.Spec) {
	s.Linux.Seccomp = seccomp.DefaultProfile(s)
}
