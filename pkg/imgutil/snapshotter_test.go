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

package imgutil

import (
	"context"
	"reflect"
	"testing"

	containerd "github.com/containerd/containerd/v2/client"
	ctdsnapshotters "github.com/containerd/containerd/v2/pkg/snapshotters"
	"gotest.tools/v3/assert"

	"go.farcloser.world/containers/digest"
	"go.farcloser.world/containers/specs"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/imgutil/pull"
)

const (
	testRef = "test:latest"
)

func TestGetSnapshotterOpts(t *testing.T) {
	type testCase struct {
		sns   []string
		check func(t *testing.T, o snapshotterOpts)
	}
	testCases := []testCase{
		{
			sns:   []string{"overlayfs"},
			check: sameOpts(&defaultSnapshotterOpts{snapshotter: "overlayfs"}),
		},
		{
			sns:   []string{"overlayfs2"},
			check: sameOpts(&defaultSnapshotterOpts{snapshotter: "overlayfs2"}),
		},
		{
			sns:   []string{"soci"},
			check: remoteSnOpts("soci", true),
		},
	}
	for _, tc := range testCases {
		for i := range tc.sns {
			got := getSnapshotterOpts(tc.sns[i])
			tc.check(t, got)
		}
	}
}

func remoteSnOpts(name string, withExtra bool) func(*testing.T, snapshotterOpts) {
	return func(t *testing.T, got snapshotterOpts) {
		opts, ok := got.(*remoteSnapshotterOpts)
		assert.Equal(t, ok, true)
		assert.Equal(t, opts.snapshotter, name)
		assert.Equal(t, opts.extraLabels != nil, withExtra)
	}
}

func sameOpts(want snapshotterOpts) func(*testing.T, snapshotterOpts) {
	return func(t *testing.T, got snapshotterOpts) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("getSnapshotterOpts() got = %v, want %v", got, want)
		}
	}
}

func getAndApplyRemoteOpts(t *testing.T, sn string) *containerd.RemoteContext {
	config := &pull.Config{}
	snOpts := getSnapshotterOpts(sn)
	rFlags := options.RemoteSnapshotterFlags{}
	snOpts.apply(config, testRef, rFlags)

	rc := &containerd.RemoteContext{}
	for _, o := range config.RemoteOpts {
		// here passing a nil client is safe
		// because the remote opts will not use client
		if err := o(nil, rc); err != nil {
			t.Errorf("failed to apply remote opts: %s", err)
		}
	}

	return rc
}

func TestDefaultSnapshotterOpts(t *testing.T) {
	rc := getAndApplyRemoteOpts(t, "overlayfs")
	assert.Equal(t, rc.Snapshotter, "overlayfs")
}

// dummyImageHandler will return a dummy layer
// see https://github.com/containerd/containerd/blob/77d53d2d230c3bcd3f02e6f493019a72905c875b/images/mediatypes.go#L115
type dummyImageHandler struct{}

func (dih *dummyImageHandler) Handle(
	_ctx context.Context,
	_desc specs.Descriptor,
) (subdescs []specs.Descriptor, err error) {
	return []specs.Descriptor{
		{
			MediaType: "application/vnd.oci.image.layer.dummy",
			Digest:    digest.FromString("dummy"),
		},
	}, nil
}

func TestRemoteSnapshotterOpts(t *testing.T) {
	tests := []struct {
		name  string
		check []func(t *testing.T, a map[string]string)
	}{
		{
			name: "soci",
			check: []func(t *testing.T, a map[string]string){
				checkRemoteSnapshotterAnnotataions, checkSociSnapshotterAnnotataions,
			},
		},
	}

	for _, tt := range tests {
		sn := tt.name
		t.Run(sn, func(t *testing.T) {
			rc := getAndApplyRemoteOpts(t, sn)
			assert.Equal(t, rc.Snapshotter, sn)

			desc := specs.Descriptor{
				MediaType: specs.MediaTypeImageManifest,
			}

			h := &dummyImageHandler{}
			got, err := rc.HandlerWrapper(h).Handle(context.Background(), desc)

			assert.NilError(t, err)
			assert.Check(t, len(got) == 1)
			for _, f := range tt.check {
				f(t, got[0].Annotations)
			}
		})
	}
}

func checkRemoteSnapshotterAnnotataions(t *testing.T, a map[string]string) {
	assert.Check(t, a != nil)
	assert.Equal(t, a[ctdsnapshotters.TargetRefLabel], testRef)
}

// using values from soci source to check for annotations (
// see
// https://github.com/awslabs/soci-snapshotter/blob/b05ba712d246ecc5146469f87e5e9305702fd72b/fs/source/source.go#L80C1-L80C6
func checkSociSnapshotterAnnotataions(t *testing.T, a map[string]string) {
	assert.Check(t, a != nil)
	_, ok := a["containerd.io/snapshot/remote/soci.size"]
	assert.Equal(t, ok, true)
	_, ok = a["containerd.io/snapshot/remote/image.layers.size"]
	assert.Equal(t, ok, true)
	_, ok = a["containerd.io/snapshot/remote/soci.index.digest"]
	assert.Equal(t, ok, true)
}
