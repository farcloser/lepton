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

package image

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.farcloser.world/containers/reference"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"

	"github.com/containerd/nerdctl/v2/leptonic/api"
	"github.com/containerd/nerdctl/v2/leptonic/services/helpers"
)

var (
	ErrServiceImage = errors.New("image error")
)

/*
	Get(ctx context.Context, name string) (Image, error)
	Create(ctx context.Context, image Image) (Image, error)

	// Update will replace the data in the store with the provided image. If
	// one or more fieldpaths are provided, only those fields will be updated.
	Update(ctx context.Context, image Image, fieldpaths ...string) (Image, error)

	Delete(ctx context.Context, name string, opts ...DeleteOpt) error


type Image struct {
	// Name of the image.
	//
	// To be pulled, it must be a reference compatible with resolvers.
	//
	// This field is required.
	Name string

	// Labels provide runtime decoration for the image record.
	//
	// There is no default behavior for how these labels are propagated. They
	// only decorate the static metadata object.
	//
	// This field is optional.
	Labels map[string]string

	// Target describes the root content for this image. Typically, this is
	// a manifest, index or manifest list.
	Target ocispec.Descriptor

	CreatedAt, UpdatedAt time.Time
}*/

func ListNames(ctx context.Context, cli *client.Client, filters ...string) ([]string, error) {
	service := cli.ImageService()

	list, err := service.List(ctx, filters...)
	if err != nil {
		return nil, errWrap(helpers.ErrConvert(err))
	}

	imgs := []string{}
	for _, img := range list {
		imgs = append(imgs, img.Name)
	}

	return imgs, nil
}

func List(ctx context.Context, cli *client.Client, filters ...string) ([]*api.Image, error) {
	service := cli.ImageService()

	list, err := service.List(ctx, filters...)
	if err != nil {
		return nil, errWrap(helpers.ErrConvert(err))
	}

	imgs := []*api.Image{}
	for _, img := range list {
		imgs = append(imgs, &api.Image{
			Name:      img.Name,
			Labels:    img.Labels,
			Target:    img.Target,
			CreatedAt: img.CreatedAt,
			UpdatedAt: img.UpdatedAt,
		})
	}

	return imgs, nil
}

func Inspect(ctx context.Context, cli *client.Client, identifier string) ([]*api.Image, string, string, error) {
	// Figure out what we have here - digest, tag, name
	parsedReference, err := reference.Parse(identifier)
	if err != nil {
		return nil, "", "", err
	}

	digest := ""
	if parsedReference.Digest != "" {
		digest = parsedReference.Digest.String()
	}

	name := parsedReference.Name()
	tag := parsedReference.Tag

	// Initialize filters
	var filters []string
	// This will hold the final image list, if any
	var imageList []images.Image

	// No digest in the request? Then assume it is a name
	if digest == "" {
		filters = []string{fmt.Sprintf("name==%s:%s", name, tag)}
		// Query it
		imageList, err = cli.ImageService().List(ctx, filters...)
		if err != nil {
			return nil, "", "", errWrap(helpers.ErrConvert(err))
		}

		// Nothing? Then it could be a short id (aka truncated digest) - we are going to use this
		if len(imageList) == 0 {
			digest = fmt.Sprintf("sha256:%s.*", regexp.QuoteMeta(strings.TrimPrefix(identifier, "sha256:")))
			name = ""
			tag = ""
		} else {
			// Otherwise, we found one by name. Get the digest from it.
			digest = imageList[0].Target.Digest.String()
		}
	}

	// At this point, we DO have a digest (or short id), so, that is what we are retrieving
	list, err := cli.ImageService().List(ctx, fmt.Sprintf("target.digest~=^%s$", digest))
	if err != nil {
		return nil, "", "", errWrap(helpers.ErrConvert(err))
	}

	// TODO: docker does allow retrieving images by Id, so implement as a last ditch effort (probably look-up the store)

	imgs := []*api.Image{}
	for _, img := range list {
		imgs = append(imgs, &api.Image{
			Name:      img.Name,
			Labels:    img.Labels,
			Target:    img.Target,
			CreatedAt: img.CreatedAt,
			UpdatedAt: img.UpdatedAt,
		})
	}

	// Return the list we found, along with normalized name and tag
	return imgs, name, tag, nil
}

func errWrap(err error) error {
	return errors.Join(ErrServiceImage, err)
}
