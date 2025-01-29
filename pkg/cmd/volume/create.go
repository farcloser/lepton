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
	"fmt"
	"io"

	"github.com/docker/docker/pkg/stringid"

	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/labels"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

func Create(ctx context.Context, out io.Writer, globalOptions *options.Global, options *options.VolumeCreate) error {
	volStore, err := Store(globalOptions.Namespace, globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return err
	}

	name := options.Name
	if name == "" {
		name = stringid.GenerateRandomID()
		options.Labels = append(options.Labels, labels.AnonymousVolumes+"=")
	}

	_, err = volStore.Create(name, strutil.DedupeStrSlice(options.Labels))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(out, name)

	return err
}
