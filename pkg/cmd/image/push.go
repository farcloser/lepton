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
	"net/http"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/converter"
	"github.com/containerd/containerd/v2/core/remotes"
	"github.com/containerd/containerd/v2/core/remotes/docker"
	dockerconfig "github.com/containerd/containerd/v2/core/remotes/docker/config"
	"github.com/containerd/log"

	"go.farcloser.world/containers/reference"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/errutil"
	converterutil "go.farcloser.world/lepton/pkg/imgutil/converter"
	"go.farcloser.world/lepton/pkg/imgutil/dockerconfigresolver"
	"go.farcloser.world/lepton/pkg/imgutil/push"
	"go.farcloser.world/lepton/pkg/platformutil"
	"go.farcloser.world/lepton/pkg/signutil"
	"go.farcloser.world/lepton/pkg/snapshotterutil"
)

// Push pushes an image specified by `rawRef`.
func Push(ctx context.Context, client *containerd.Client, rawRef string, options options.ImagePush) error {
	parsedReference, err := reference.Parse(rawRef)
	if err != nil {
		return err
	}
	ref := parsedReference.String()
	refDomain := parsedReference.Domain

	platMC, err := platformutil.NewMatchComparer(options.AllPlatforms, options.Platforms)
	if err != nil {
		return err
	}
	pushRef := ref
	if !options.AllPlatforms {
		pushRef = ref + "-tmp-reduced-platform"
		// Push fails with "400 Bad Request" when the manifest is multi-platform, but we do not locally have
		// multi-platform blobs.
		// So we create a tmp reduced-platform image to avoid the error.
		err = EnsureAllContent(ctx, client, ref, platMC, options.GOptions)
		if err != nil {
			return err
		}
		platImg, err := converterutil.Convert(ctx, client, pushRef, ref, converter.WithPlatform(platMC))
		if err != nil {
			if len(options.Platforms) == 0 {
				return fmt.Errorf("failed to create a tmp single-platform image %q: %w", pushRef, err)
			}
			return fmt.Errorf(
				"failed to create a tmp reduced-platform image %q (platform=%v): %w",
				pushRef,
				options.Platforms,
				err,
			)
		}

		err = EnsureAllContent(ctx, client, ref, platMC, options.GOptions)
		if err != nil {
			return err
		}

		defer client.ImageService().Delete(ctx, platImg.Name, images.SynchronousDelete())
		log.G(ctx).
			Infof("pushing as a reduced-platform image (%s, %s)", platImg.Target.MediaType, platImg.Target.Digest)
	}

	// In order to push images where most layers are the same but the
	// repository name is different, it is necessary to refresh the
	// PushTracker. Otherwise, the MANIFEST_BLOB_UNKNOWN error will occur due
	// to the registry not creating the corresponding layer link file,
	// resulting in the failure of the entire image push.
	pushTracker := docker.NewInMemoryTracker()

	pushFunc := func(r remotes.Resolver) error {
		return push.Push(
			ctx,
			client,
			r,
			pushTracker,
			options.Stdout,
			pushRef,
			ref,
			platMC,
			options.AllowNondistributableArtifacts,
			options.Quiet,
		)
	}

	var dOpts []dockerconfigresolver.Opt
	if options.GOptions.InsecureRegistry {
		log.G(ctx).Warnf("skipping verifying HTTPS certs for %q", refDomain)
		dOpts = append(dOpts, dockerconfigresolver.WithSkipVerifyCerts(true))
	}
	dOpts = append(dOpts, dockerconfigresolver.WithHostsDirs(options.GOptions.HostsDir))

	ho, err := dockerconfigresolver.NewHostOptions(ctx, refDomain, dOpts...)
	if err != nil {
		return err
	}

	resolverOpts := docker.ResolverOptions{
		Tracker: pushTracker,
		Hosts:   dockerconfig.ConfigureHosts(ctx, *ho),
	}

	resolver := docker.NewResolver(resolverOpts)
	if err = pushFunc(resolver); err != nil {
		// In some circumstance (e.g. people just use 80 port to support pure http), the error will contain message like
		// "dial tcp <port>: connection refused"
		if !errors.Is(err, http.ErrSchemeMismatch) && !errutil.IsErrConnectionRefused(err) {
			return err
		}
		if options.GOptions.InsecureRegistry {
			log.G(ctx).
				WithError(err).
				Warnf("server %q does not seem to support HTTPS, falling back to plain HTTP", refDomain)
			dOpts = append(dOpts, dockerconfigresolver.WithPlainHTTP(true))
			resolver, err = dockerconfigresolver.New(ctx, refDomain, dOpts...)
			if err != nil {
				return err
			}
			return pushFunc(resolver)
		}
		log.G(ctx).WithError(err).Errorf("server %q does not seem to support HTTPS", refDomain)
		log.G(ctx).
			Info("Hint: you may want to try --insecure-registry to allow plain HTTP (if you are in a trusted network)")
		return err
	}

	img, err := client.ImageService().Get(ctx, pushRef)
	if err != nil {
		return err
	}
	refSpec, err := reference.Parse(pushRef)
	if err != nil {
		return err
	}
	signRef := fmt.Sprintf("%s@%s", refSpec.String(), img.Target.Digest.String())
	if err = signutil.Sign(signRef,
		options.GOptions.Experimental,
		options.SignOptions); err != nil {
		return err
	}
	if options.GOptions.Snapshotter == "soci" {
		if err = snapshotterutil.CreateSoci(ref, options.GOptions, options.AllPlatforms, options.Platforms, &options.SociOptions); err != nil {
			return err
		}
		if err = snapshotterutil.PushSoci(ref, options.GOptions, options.AllPlatforms, options.Platforms); err != nil {
			return err
		}
	}
	if options.Quiet {
		fmt.Fprintln(options.Stdout, ref)
	}
	return nil
}
