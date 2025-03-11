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

package rootlessutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rootless-containers/rootlesskit/v2/pkg/api/client"

	"go.farcloser.world/lepton/leptonic/rootlesskit"
)

func IsRootless() bool {
	return IsRootlessParent() || IsRootlessChild()
}

func ParentEUID() int {
	if !IsRootlessChild() {
		return os.Geteuid()
	}

	i, err := strconv.Atoi(os.Getenv("ROOTLESSKIT_PARENT_EUID"))
	if err != nil {
		panic(fmt.Errorf("failed to parse ROOTLESSKIT_PARENT_EUID: %w", err))
	}

	return i
}

func NewRootlessKitClient() (client.Client, error) {
	stateDir, err := rootlesskit.StateDir()
	if err != nil {
		return nil, err
	}
	apiSock := filepath.Join(stateDir, "api.sock")

	return client.New(apiSock)
}

// RootlessContainredSockAddress returns sock address of rootless containerd based on https://github.com/farcloser/lepton/blob/main/docs/faq.md#containerd-socket-address
func RootlessContainredSockAddress() (string, error) {
	stateDir, err := rootlesskit.StateDir()
	if err != nil {
		return "", err
	}
	childPid, err := rootlesskit.ChildPid(stateDir)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/proc/%d/root/run/containerd/containerd.sock", childPid), nil
}

// DetachedNetNS returns non-empty netns path if RootlessKit is running with --detach-netns mode.
// Otherwise, returns "" without an error.
func DetachedNetNS() (string, error) {
	if !IsRootless() {
		return "", nil
	}

	stateDir, err := rootlesskit.StateDir()
	if err != nil {
		return "", err
	}

	return getDetachedNetNSPath(stateDir)
}

// WithDetachedNetNSIfAny executes fn in [DetachedNetNS] if RootlessKit is running with --detach-netns mode.
// Otherwise, it just executes fn in the current netns.
func WithDetachedNetNSIfAny(fn func() error) error {
	netns, err := DetachedNetNS()
	if err != nil {
		return err
	}

	if netns == "" {
		return fn()
	}

	return ns.WithNetNSPath(netns, func(_ ns.NetNS) error { return fn() })
}

func getDetachedNetNSPath(stateDir string) (string, error) {
	p := filepath.Join(stateDir, "netns")
	if _, err := os.Stat(p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	return p, nil
}
