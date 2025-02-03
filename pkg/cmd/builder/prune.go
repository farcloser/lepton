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
	"fmt"
	"io"
	"os/exec"

	"go.farcloser.world/core/units"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/buildkitutil"
)

// Prune will prune all build cache.
func Prune(ctx context.Context, output io.Writer, _ *options.Global, opts *options.BuilderPrune) error {
	buildctlBinary, err := buildkitutil.BuildctlBinary()
	if err != nil {
		return err
	}

	buildctlArgs := buildkitutil.BuildctlBaseArgs(opts.BuildKitHost)
	buildctlArgs = append(buildctlArgs, "prune", "--format={{json .}}")

	if opts.All {
		buildctlArgs = append(buildctlArgs, "--all")
	}

	buildctlCmd := exec.Command(buildctlBinary, buildctlArgs...)

	buildctlCmd.Stderr = opts.Stderr
	stdout, err := buildctlCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("faild to get stdout piper for %v: %w", buildctlCmd.Args, err)
	}

	defer stdout.Close()

	if err = buildctlCmd.Start(); err != nil {
		return fmt.Errorf("faild to start %v: %w", buildctlCmd.Args, err)
	}

	dec := json.NewDecoder(stdout)
	result := make([]buildkitutil.UsageInfo, 0)

	for {
		var v buildkitutil.UsageInfo
		if err := dec.Decode(&v); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("faild to decode output from %v: %w", buildctlCmd.Args, err)
		}
		result = append(result, v)
	}

	if err = buildctlCmd.Wait(); err != nil {
		return fmt.Errorf("faild to wait for %v to complete: %w", buildctlCmd.Args, err)
	}

	var totalReclaimedSpace int64

	if len(result) > 0 {
		fmt.Fprintln(output, "Deleted build cache objects:")
		for _, item := range result {
			fmt.Fprintln(output, item.ID)
			totalReclaimedSpace += item.Size
		}
	}

	_, err = fmt.Fprintf(output, "Total:  %s\n", units.BytesSize(float64(totalReclaimedSpace)))

	return err
}
