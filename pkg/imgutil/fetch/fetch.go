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

package fetch

import (
	"context"
	"io"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/remotes"
	"github.com/containerd/log"

	"go.farcloser.world/containers/specs"

	"go.farcloser.world/lepton/pkg/imgutil/jobs"
	"go.farcloser.world/lepton/pkg/platformutil"
)

// Config for content fetch
type Config struct {
	// Resolver
	Resolver remotes.Resolver
	// ProgressOutput to display progress
	ProgressOutput io.Writer
	// RemoteOpts, e.g. containerd.WithPullUnpack.
	//
	// Regardless to RemoteOpts, the following opts are always set:
	// WithResolver, WithImageHandler
	//
	// RemoteOpts related to unpacking can be set only when len(Platforms) is 1.
	RemoteOpts []containerd.RemoteOpt
	Platforms  []specs.Platform // empty for all-platforms
}

func Fetch(ctx context.Context, client *containerd.Client, ref string, config *Config) error {
	ongoing := jobs.New(ref)

	pctx, stopProgress := context.WithCancel(ctx)
	progress := make(chan struct{})

	go func() {
		if config.ProgressOutput != nil {
			// no progress bar, because it hides some debug logs
			jobs.ShowProgress(pctx, ongoing, client.ContentStore(), config.ProgressOutput)
		}
		close(progress)
	}()

	h := images.HandlerFunc(func(ctx context.Context, desc specs.Descriptor) ([]specs.Descriptor, error) {
		ongoing.Add(desc)
		return nil, nil
	})

	log.G(pctx).WithField("image", ref).Debug("fetching")
	platformMC := platformutil.NewMatchComparerFromOCISpecPlatformSlice(config.Platforms)
	opts := []containerd.RemoteOpt{
		containerd.WithResolver(config.Resolver),
		containerd.WithImageHandler(h),
		containerd.WithPlatformMatcher(platformMC),
	}
	opts = append(opts, config.RemoteOpts...)

	// Note that client.Fetch does not unpack
	_, err := client.Fetch(pctx, ref, opts...)

	stopProgress()
	if err != nil {
		return err
	}

	<-progress
	return nil
}
