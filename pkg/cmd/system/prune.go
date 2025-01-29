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

package system

import (
	"context"
	"fmt"
	"io"

	containerd "github.com/containerd/containerd/v2/client"

	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/cmd/builder"
	"github.com/containerd/nerdctl/v2/pkg/cmd/container"
	"github.com/containerd/nerdctl/v2/pkg/cmd/image"
	"github.com/containerd/nerdctl/v2/pkg/cmd/network"
	"github.com/containerd/nerdctl/v2/pkg/cmd/volume"
)

// Prune will remove all unused containers, networks,
// images (dangling only or both dangling and unreferenced), and optionally, volumes.
func Prune(ctx context.Context, client *containerd.Client, out io.Writer, globalOptions *options.Global, opts *options.SystemPrune) error {
	if err := container.Prune(ctx, client, types.ContainerPruneOptions{
		GOptions: *globalOptions,
		Stdout:   out,
	}); err != nil {
		return err
	}
	if err := network.Prune(ctx, client, out, globalOptions, &options.NetworkPrune{
		NetworkDriversToKeep: opts.NetworkDriversToKeep,
	}); err != nil {
		return err
	}
	if opts.Volumes {
		if err := volume.Prune(ctx, client, out, globalOptions, &options.VolumePrune{
			All:   false,
			Force: true,
		}); err != nil {
			return err
		}
	}
	if err := image.Prune(ctx, client, types.ImagePruneOptions{
		Stdout:   out,
		GOptions: *globalOptions,
		All:      opts.All,
	}); err != nil {
		// ?
		return nil //nolint:nilerr
	}

	if opts.BuildKitHost != "" {
		prunedObjects, err := builder.Prune(ctx, &options.BuilderPrune{
			Stderr:       opts.Stderr,
			All:          opts.All,
			BuildKitHost: opts.BuildKitHost,
		})
		if err != nil {
			return err
		}

		if len(prunedObjects) > 0 {
			_, err = fmt.Fprintln(out, "Deleted build cache objects:")
			for _, item := range prunedObjects {
				_, err = fmt.Fprintln(out, item.ID)
			}
			if err != nil {
				return err
			}
		}
	}

	// TODO: print total reclaimed space

	return nil
}
