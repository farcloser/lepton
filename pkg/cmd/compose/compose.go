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

package compose

import (
	"context"
	"errors"
	"io"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"

	"go.farcloser.world/containers/reference"
	"go.farcloser.world/containers/specs"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/cmd/volume"
	"go.farcloser.world/lepton/pkg/composer"
	"go.farcloser.world/lepton/pkg/composer/serviceparser"
	"go.farcloser.world/lepton/pkg/imgutil"
	"go.farcloser.world/lepton/pkg/netutil"
	"go.farcloser.world/lepton/pkg/signutil"
	"go.farcloser.world/lepton/pkg/strutil"
)

// New returns a new *composer.Composer.
func New(client *containerd.Client, globalOptions *options.Global, opts *composer.Options, stdout, stderr io.Writer) (*composer.Composer, error) {
	if err := composer.Lock(globalOptions.DataRoot, globalOptions.Address); err != nil {
		return nil, err
	}

	cniEnv, err := netutil.NewCNIEnv(globalOptions.CNIPath, globalOptions.CNINetConfPath, netutil.WithNamespace(globalOptions.Namespace), netutil.WithDefaultNetwork(globalOptions.BridgeIP))
	if err != nil {
		return nil, err
	}
	networkConfigs, err := cniEnv.NetworkList()
	if err != nil {
		return nil, err
	}
	opts.NetworkExists = func(netName string) (bool, error) {
		for _, f := range networkConfigs {
			if f.Name == netName {
				return true, nil
			}
		}
		return false, nil
	}

	opts.NetworkInUse = func(ctx context.Context, netName string) (bool, error) {
		networkUsedByNsMap, err := netutil.UsedNetworks(ctx, client)
		if err != nil {
			return false, err
		}
		for _, v := range networkUsedByNsMap {
			if strutil.InStringSlice(v, netName) {
				return true, nil
			}
		}
		return false, nil
	}

	volStore, err := volume.Store(globalOptions.Namespace, globalOptions.DataRoot, globalOptions.Address)
	if err != nil {
		return nil, err
	}
	// FIXME: this is racy. See note in up_volume.go
	opts.VolumeExists = volStore.Exists

	opts.ImageExists = func(ctx context.Context, rawRef string) (bool, error) {
		parsedReference, err := reference.Parse(rawRef)
		if err != nil {
			return false, err
		}
		ref := parsedReference.String()
		if _, err := client.ImageService().Get(ctx, ref); err != nil {
			if errors.Is(err, errdefs.ErrNotFound) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}

	opts.EnsureImage = func(ctx context.Context, imageName, pullMode, platform string, ps *serviceparser.Service, quiet bool) error {
		ocispecPlatforms := []specs.Platform{platforms.DefaultSpec()}
		if platform != "" {
			parsed, err := platforms.Parse(platform)
			if err != nil {
				return err
			}
			ocispecPlatforms = []specs.Platform{parsed} // no append
		}

		imgPullOpts := options.ImagePull{
			GOptions:        globalOptions,
			OCISpecPlatform: ocispecPlatforms,
			Unpack:          nil,
			Mode:            pullMode,
			Quiet:           quiet,
			RFlags:          options.RemoteSnapshotterFlags{},
			Stdout:          stdout,
			Stderr:          stderr,
		}

		imageVerifyOptions := imageVerifyOptionsFromCompose(ps)
		ref, err := signutil.Verify(ctx, imageName, globalOptions.HostsDir, globalOptions.Experimental, imageVerifyOptions)
		if err != nil {
			return err
		}

		_, err = imgutil.EnsureImage(ctx, client, ref, imgPullOpts)
		return err
	}

	return composer.New(opts, client)
}

func imageVerifyOptionsFromCompose(ps *serviceparser.Service) options.ImageVerify {
	var opt options.ImageVerify
	if verifier, ok := ps.Unparsed.Extensions[serviceparser.ComposeVerify]; ok {
		opt.Provider = verifier.(string)
	} else {
		opt.Provider = "none"
	}

	// for cosign, if key is given, use key mode, otherwise use keyless mode.
	if keyVal, ok := ps.Unparsed.Extensions[serviceparser.ComposeCosignPublicKey]; ok {
		opt.CosignKey = keyVal.(string)
	}
	if ciVal, ok := ps.Unparsed.Extensions[serviceparser.ComposeCosignCertificateIdentity]; ok {
		opt.CosignCertificateIdentity = ciVal.(string)
	}
	if cirVal, ok := ps.Unparsed.Extensions[serviceparser.ComposeCosignCertificateIdentityRegexp]; ok {
		opt.CosignCertificateIdentityRegexp = cirVal.(string)
	}
	if coiVal, ok := ps.Unparsed.Extensions[serviceparser.ComposeCosignCertificateOidcIssuer]; ok {
		opt.CosignCertificateOidcIssuer = coiVal.(string)
	}
	if coirVal, ok := ps.Unparsed.Extensions[serviceparser.ComposeCosignCertificateOidcIssuerRegexp]; ok {
		opt.CosignCertificateOidcIssuerRegexp = coirVal.(string)
	}
	return opt
}
