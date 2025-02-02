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

package native

import (
	"github.com/containerd/containerd/v2/core/images"

	"go.farcloser.world/containers/specs"
)

// Image corresponds to a containerd-native image object.
// Not compatible with `docker image inspect`.
type Image struct {
	Image        images.Image      `json:"Image"`
	IndexDesc    *specs.Descriptor `json:"IndexDesc,omitempty"`
	Index        *specs.Index      `json:"Index,omitempty"`
	ManifestDesc *specs.Descriptor `json:"ManifestDesc,omitempty"`
	Manifest     *specs.Manifest   `json:"Manifest,omitempty"`
	// e.g., "application/vnd.docker.container.image.v1+json"
	ImageConfigDesc specs.Descriptor `json:"ImageConfigDesc"`
	ImageConfig     specs.Image      `json:"ImageConfig"`
	Size            int64            `json:"size"`
}
