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

package volume

import (
	"context"
	"errors"
	"io"

	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/formatter"
)

func Inspect(ctx context.Context, output io.Writer, globalOptions *options.Global, opts *options.VolumeInspect) error {
	volStore, err := Store(globalOptions.Namespace, globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return err
	}

	result := []interface{}{}
	warns := []error{}

	for _, name := range opts.NamesList {
		vol, err := volStore.Get(name, opts.Size)
		if err != nil {
			warns = append(warns, err)
			continue
		}

		result = append(result, vol)
	}

	err = formatter.FormatSlice(opts.Format, output, result)
	if err != nil {
		return err
	}

	for _, warn := range warns {
		log.G(ctx).Warn(warn)
	}

	if len(warns) != 0 {
		return errors.New("some volumes could not be inspected")
	}

	return nil
}
