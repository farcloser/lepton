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
	"fmt"
	"io"

	"go.farcloser.world/core/units"

	"go.farcloser.world/lepton/leptonic/services/builder"
	"go.farcloser.world/lepton/pkg/api/options"
)

// Prune will prune all build cache.
func Prune(ctx context.Context, output io.Writer, _ *options.Global, opts *options.BuilderPrune) error {
	result, err := builder.Prune(ctx, opts.Stderr, opts.BuildKitHost, opts.All)
	if err != nil {
		return err
	}

	var totalReclaimedSpace int64
	if len(result) > 0 {
		_, err = fmt.Fprintln(output, "Deleted build cache objects:")
		if err != nil {
			return err
		}

		for _, item := range result {
			_, err = fmt.Fprintln(output, item.ID)
			if err != nil {
				return err
			}
			totalReclaimedSpace += item.Size
		}
	}

	_, err = fmt.Fprintf(output, "Total:  %s\n", units.BytesSize(float64(totalReclaimedSpace)))

	return err
}
