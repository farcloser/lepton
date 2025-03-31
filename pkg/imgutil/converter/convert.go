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

package converter

import (
	"context"

	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/converter"
)

// It looks like containerd converter.Convert is faulty.
// When dstRef != srcRef, convert will first forcefully delete dstRef,
// apparently *asynchronously*, then create the image.
// This is racy, and the deletion may kick in after the creation.
// This here is to workaround the bug, by manually creating the image first,
// then converting it in place (which avoid the problematic code-path).

func Convert(
	ctx context.Context,
	client converter.Client,
	dstRef, srcRef string,
	opts ...converter.Opt,
) (*images.Image, error) {
	imageService := client.ImageService()

	img, err := imageService.Get(ctx, srcRef)
	if err != nil {
		return nil, err
	}

	img.Name = dstRef

	_ = imageService.Delete(ctx, dstRef, images.SynchronousDelete())

	if _, err = imageService.Create(ctx, img); err != nil {
		return nil, err
	}

	return converter.Convert(ctx, client, dstRef, dstRef, opts...)
}
