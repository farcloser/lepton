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
	"io"

	containerd "github.com/containerd/containerd/v2/client"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/builder"
	"go.farcloser.world/lepton/pkg/cmd/container"
	"go.farcloser.world/lepton/pkg/cmd/image"
	"go.farcloser.world/lepton/pkg/cmd/network"
	"go.farcloser.world/lepton/pkg/cmd/volume"
)

// Prune will remove all unused containers, networks,
// images (dangling only or both dangling and unreferenced), and optionally, volumes.
func Prune(
	ctx context.Context,
	client *containerd.Client,
	output io.Writer,
	globalOptions *options.Global,
	opts *options.SystemPrune,
) error {
	if err := container.Prune(ctx, client, output, globalOptions, &options.ContainerPrune{}); err != nil {
		return err
	}
	if err := network.Prune(ctx, client, output, globalOptions, &options.NetworkPrune{
		NetworkDriversToKeep: opts.NetworkDriversToKeep,
	}); err != nil {
		return err
	}
	if opts.Volumes {
		if err := volume.Prune(ctx, client, output, globalOptions, &options.VolumePrune{
			All: false,
		}); err != nil {
			return err
		}
	}
	if err := image.Prune(ctx, client, globalOptions, options.ImagePrune{
		Stdout: output,
		All:    opts.All,
	}); err != nil {
		// ?
		return nil //nolint:nilerr
	}

	if opts.BuildKitHost != "" {
		err := builder.Prune(ctx, output, globalOptions, &options.BuilderPrune{
			Stderr:       opts.Stderr,
			All:          opts.All,
			BuildKitHost: opts.BuildKitHost,
		})
		if err != nil {
			return err
		}
	}

	// TODO: print total reclaimed space

	return nil
}
