/*
   Copyright The containerd Authors.

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

package platformutil

import (
	"context"

	"go.farcloser.world/containers/specs"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/platforms"
)

func LayerDescs(ctx context.Context, provider content.Provider, imageTarget specs.Descriptor, platform platforms.MatchComparer) ([]specs.Descriptor, error) {
	var descs []specs.Descriptor
	err := images.Walk(ctx, images.Handlers(images.HandlerFunc(func(ctx context.Context, desc specs.Descriptor) ([]specs.Descriptor, error) {
		if images.IsLayerType(desc.MediaType) {
			descs = append(descs, desc)
		}
		return nil, nil
	}), images.FilterPlatforms(images.ChildrenHandler(provider), platform)), imageTarget)
	return descs, err
}
