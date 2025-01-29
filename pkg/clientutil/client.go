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

package clientutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.farcloser.world/containers/digest"

	ctdcli "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"
	"github.com/containerd/platforms"

	"github.com/containerd/nerdctl/v2/leptonic/emulation"
	"github.com/containerd/nerdctl/v2/leptonic/services/containerd"
)

func NewClient(ctx context.Context, namespace string, address string) (*ctdcli.Client, context.Context, context.CancelFunc, error) {
	return containerd.NewClient(ctx, namespace, address)
}

func NewClientWithOpt(ctx context.Context, namespace string, address string, clientOpt ctdcli.Opt) (*ctdcli.Client, context.Context, context.CancelFunc, error) {
	return containerd.NewClient(ctx, namespace, address, clientOpt)
}

func NewClientWithPlatform(ctx context.Context, namespace, address, platform string) (*ctdcli.Client, context.Context, context.CancelFunc, error) {
	clientOpts := []ctdcli.Opt{}
	if platform != "" {
		platformParsed, err := platforms.Parse(platform)
		if err != nil {
			return nil, nil, nil, err
		}

		if canExec, canExecErr := emulation.CanExecProbably(platformParsed); !canExec {
			warn := fmt.Sprintf("Platform %q seems incompatible with the host platform %q. If you see \"exec format error\", see https://github.com/containerd/nerdctl/blob/main/docs/multi-platform.md",
				platform, platforms.DefaultString())
			if canExecErr != nil {
				log.L.WithError(canExecErr).Warn(warn)
			} else {
				log.L.Warn(warn)
			}
		}

		clientOpts = append(clientOpts, ctdcli.WithDefaultPlatform(platforms.Only(platformParsed)))
	}

	return containerd.NewClient(ctx, namespace, address, clientOpts...)
}

// DataStore returns a string like "/var/lib/version.RootName/1935db59".
// "1935db9" is from `$(echo -n "/run/containerd/containerd.sock" | sha256sum | cut -c1-8)`
// on Windows it will return "%PROGRAMFILES%/version.RootName/1935db59"
// where `version.RootName` is defined at build time
func DataStore(dataRoot, address string) (string, error) {
	if err := os.MkdirAll(dataRoot, 0o700); err != nil {
		return "", err
	}
	addrHash, err := getAddrHash(address)
	if err != nil {
		return "", err
	}
	dataStore := filepath.Join(dataRoot, addrHash)
	if err := os.MkdirAll(dataStore, 0o700); err != nil {
		return "", err
	}
	return dataStore, nil
}

func getAddrHash(addr string) (string, error) {
	const addrHashLen = 8

	if runtime.GOOS != "windows" {
		addr = strings.TrimPrefix(addr, "unix://")

		var err error
		addr, err = filepath.EvalSymlinks(addr)
		if err != nil {
			return "", err
		}
	}

	d := digest.FromString(addr)
	h := d.Encoded()[0:addrHashLen]
	return h, nil
}
