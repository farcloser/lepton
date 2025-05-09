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
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/log"

	"go.farcloser.world/containers/reference"

	"go.farcloser.world/lepton/leptonic/api"
	"go.farcloser.world/lepton/leptonic/services/image"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/containerdutil"
	"go.farcloser.world/lepton/pkg/formatter"
	"go.farcloser.world/lepton/pkg/imageinspector"
	"go.farcloser.world/lepton/pkg/inspecttypes/dockercompat"
)

func inspectIdentifier(
	ctx context.Context,
	client *containerd.Client,
	identifier string,
) ([]*api.Image, string, string, error) {
	return image.Inspect(ctx, client, identifier)
}

// Inspect prints detailed information of each image in `images`.
func Inspect(ctx context.Context, client *containerd.Client, identifiers []string, options options.ImageInspect) error {
	// Verify we have a valid mode
	// TODO: move this out of here, to Cobra command line arg validation
	if options.Mode != "native" && options.Mode != "dockercompat" {
		return fmt.Errorf("unknown mode %q", options.Mode)
	}

	// Set a timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Will hold the final answers
	var errs []error
	var entries []interface{}

	snapshotter := containerdutil.SnapshotService(client, options.GOptions.Snapshotter)
	// We have to query per provided identifier, as we need to post-process results for the case name + digest
	for _, identifier := range identifiers {
		candidateImageList, requestedName, requestedTag, err := inspectIdentifier(ctx, client, identifier)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %s", err, identifier))
			continue
		}

		var validatedImage *dockercompat.Image
		var repoTags []string
		var repoDigests []string

		// Go through the candidates
		for _, candidateImage := range candidateImageList {
			// Inspect the image
			candidateNativeImage, err := imageinspector.Inspect(ctx, client,
				images.Image{
					Name:      candidateImage.Name,
					Labels:    candidateImage.Labels,
					Target:    candidateImage.Target,
					CreatedAt: candidateImage.CreatedAt,
					UpdatedAt: candidateImage.UpdatedAt,
				}, snapshotter)
			if err != nil {
				log.G(ctx).WithError(err).WithField("name", candidateImage.Name).Error("failure inspecting image")
				continue
			}

			// If native, we just add everything in there and that's it
			if options.Mode == "native" {
				entries = append(entries, candidateNativeImage)
				continue
			}

			// If dockercompat: does the candidate have a name? Get it if so
			parsedReference, err := reference.Parse(candidateNativeImage.Image.Name)
			if err != nil {
				log.G(ctx).
					WithError(err).
					WithField("name", candidateNativeImage.Image.Name).
					Error("the found image has an unparsable name")
				continue
			}

			// If we were ALSO asked for a specific name on top of the digest, we need to make sure we keep only the
			// image with that name
			if requestedName != "" {
				// If the candidate did not have a name, then we should ignore this one and continue
				if parsedReference.Name() == "" {
					continue
				}

				// Otherwise, the candidate has a name. If it is the one we want, store it and continue, otherwise, fall
				// through
				candidateTag := parsedReference.Tag
				// If the name had a digest, an empty tag is not normalized to latest, so, account for that here
				if requestedTag == "" {
					requestedTag = "latest"
				}
				if parsedReference.Name() == requestedName && candidateTag == requestedTag {
					validatedImage, err = dockercompat.ImageFromNative(candidateNativeImage)
					if err != nil {
						log.G(ctx).
							WithError(err).
							WithField("name", candidateNativeImage.Image.Name).
							Error("could not get a docker compat version of the native image")
					}
					continue
				}
			} else if validatedImage == nil {
				// Alternatively, we got a request by digest only, so, if we do not know about it already, store it and
				// continue
				validatedImage, err = dockercompat.ImageFromNative(candidateNativeImage)
				if err != nil {
					log.G(ctx).WithError(err).WithField("name", candidateNativeImage.Image.Name).Error("could not get a docker compat version of the native image")
				}
				continue
			}

			// Fallthrough cases:
			// - we got a request by digest, but we already had the image stored
			// - we got a request by name, and the name of the candidate did not match the requested name
			// Now, check if the candidate has a name - if it does, populate repoTags and repoDigests
			if parsedReference.Name() != "" {
				tag := parsedReference.Tag
				if tag == "" {
					tag = "latest"
				}
				repoTags = append(repoTags, fmt.Sprintf("%s:%s", parsedReference.FamiliarName(), tag))
				repoDigests = append(
					repoDigests,
					fmt.Sprintf("%s@%s", parsedReference.FamiliarName(), candidateImage.Target.Digest.String()),
				)
			}
		}

		// Done iterating through candidates. Did we find anything that matches?
		if options.Mode == "dockercompat" {
			if validatedImage == nil {
				errs = append(errs, fmt.Errorf("no such image: %s", identifier))
				continue
			}
			// Then slap in the repoTags and repoDigests we found from the other candidates
			validatedImage.RepoTags = append(validatedImage.RepoTags, repoTags...)
			validatedImage.RepoDigests = append(validatedImage.RepoDigests, repoDigests...)
			// Store our image
			// foundImages[validatedDigest] = validatedImage
			entries = append(entries, validatedImage)
		}
	}

	// Display
	if len(entries) > 0 {
		if formatErr := formatter.FormatSlice(options.Format, options.Stdout, entries); formatErr != nil {
			log.G(ctx).Error(formatErr)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d errors:\n%w", len(errs), errors.Join(errs...))
	}

	return nil
}
