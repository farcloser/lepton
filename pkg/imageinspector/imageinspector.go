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

package imageinspector

import (
	"context"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/snapshots"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/imgutil"
	"go.farcloser.world/lepton/pkg/inspecttypes/native"
)

// Inspect inspects the image, for the platform specified in image.platform.
func Inspect(
	ctx context.Context,
	client *containerd.Client,
	image images.Image,
	snapshotter snapshots.Snapshotter,
) (*native.Image, error) {
	n := &native.Image{}

	img := containerd.NewImage(client, image)
	idx, idxDesc, err := imgutil.ReadIndex(ctx, img)
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", image.Name).Warnf("failed to inspect index")
	} else {
		n.IndexDesc = idxDesc
		n.Index = idx
	}

	mani, maniDesc, err := imgutil.ReadManifest(ctx, img)
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", image.Name).Warnf("failed to inspect manifest")
	} else {
		n.ManifestDesc = maniDesc
		n.Manifest = mani
	}

	imageConfig, imageConfigDesc, err := imgutil.ReadImageConfig(ctx, img)
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", image.Name).Warnf("failed to inspect image config")
	} else {
		n.ImageConfigDesc = imageConfigDesc
		n.ImageConfig = imageConfig
	}
	n.Size, err = imgutil.UnpackedImageSize(ctx, snapshotter, img)
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", image.Name).Warnf("failed to inspect calculate size")
	}
	n.Image = image

	return n, nil
}
