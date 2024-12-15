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

package image

import (
	"context"

	containerd "github.com/containerd/containerd/v2/client"

	"github.com/containerd/nerdctl/v2/pkg/api/types"
	"github.com/containerd/nerdctl/v2/pkg/imgutil"
	"github.com/containerd/nerdctl/v2/pkg/signutil"
)

// Pull pulls an image specified by `rawRef`.
func Pull(ctx context.Context, client *containerd.Client, rawRef string, options types.ImagePullOptions) error {
	_, err := EnsureImage(ctx, client, rawRef, options)
	if err != nil {
		return err
	}

	return nil
}

// EnsureImage pulls an image from registry.
func EnsureImage(ctx context.Context, client *containerd.Client, rawRef string, options types.ImagePullOptions) (*imgutil.EnsuredImage, error) {
	var ensured *imgutil.EnsuredImage

	ref, err := signutil.Verify(ctx, rawRef, options.GOptions.HostsDir, options.GOptions.Experimental, options.VerifyOptions)
	if err != nil {
		return nil, err
	}

	ensured, err = imgutil.EnsureImage(ctx, client, ref, options)
	if err != nil {
		return nil, err
	}
	return ensured, err
}
