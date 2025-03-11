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

package builder

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"

	"go.farcloser.world/lepton/leptonic/buildkit"
)

var (
	ErrServiceBuilder = errors.New("builder error")

	ErrStdoutPiper  = errors.New("stdout piper error")
	ErrStartFailed  = errors.New("failed to start builder")
	ErrWaitFailed   = errors.New("failed to wait for builder")
	ErrDecodeFailed = errors.New("failed to decode result")
)

// Prune will prune all build cache.
func Prune(ctx context.Context, errout io.Writer, buildkitHost string, all bool) ([]*buildkit.UsageInfo, error) {
	buildctlBinary, err := buildkit.BuildctlBinary()
	if err != nil {
		return nil, errors.Join(ErrServiceBuilder, err)
	}

	buildctlArgs := buildkit.BuildctlBaseArgs(buildkitHost)
	buildctlArgs = append(buildctlArgs, "prune", "--format={{json .}}")

	if all {
		buildctlArgs = append(buildctlArgs, "--all")
	}

	buildctlCmd := exec.Command(buildctlBinary, buildctlArgs...)
	buildctlCmd.Stderr = errout

	stdout, err := buildctlCmd.StdoutPipe()
	if err != nil {
		return nil, errors.Join(ErrServiceBuilder, ErrStdoutPiper, err)
	}

	defer stdout.Close()

	if err = buildctlCmd.Start(); err != nil {
		return nil, errors.Join(ErrServiceBuilder, ErrStartFailed, err)
	}

	dec := json.NewDecoder(stdout)
	result := make([]*buildkit.UsageInfo, 0)

	for {
		v := &buildkit.UsageInfo{}
		if err := dec.Decode(v); err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Join(ErrServiceBuilder, ErrDecodeFailed, err)
		}
		result = append(result, v)
	}

	if err = buildctlCmd.Wait(); err != nil {
		return nil, errors.Join(ErrServiceBuilder, ErrWaitFailed, err)
	}

	return result, err
}
