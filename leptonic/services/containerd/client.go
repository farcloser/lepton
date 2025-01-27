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

package containerd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/defaults"

	"github.com/containerd/nerdctl/v2/leptonic/errs"
	"github.com/containerd/nerdctl/v2/leptonic/rootlesskit"
	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/leptonic/socket"
	"github.com/containerd/nerdctl/v2/pkg/rootlessutil"
)

const (
	dockerContainerdaddress = "/var/run/docker/containerd/containerd.sock"
)

var (
	ErrServiceClient = errors.New("containerd client error")

	ErrSocketNotAccessible = errors.New("cannot access containerd socket")
)

// RootlessContainredSockAddress returns sock address of rootless containerd based on https://github.com/containerd/nerdctl/blob/main/docs/faq.md#containerd-socket-address
func rootlessContainerdSockAddress() (string, error) {
	stateDir, err := rootlesskit.StateDir()
	if err != nil {
		return "", err
	}
	childPid, err := rootlesskit.ChildPid(stateDir)
	if err != nil {
		return "", err
	}

	cpid := strconv.Itoa(childPid)

	return filepath.Join("/proc", cpid, "root", defaults.DefaultAddress), nil
}

func NewClient(ctx context.Context, ns string, address string, clientOpts ...containerd.Opt) (*containerd.Client, context.Context, context.CancelFunc, error) {
	tryAddress := address
	if address == "" {
		tryAddress = defaults.DefaultAddress
	}

	tryAddress = strings.TrimPrefix(tryAddress, "unix://")

	if address == "" && rootlessutil.IsRootless() {
		addr, err := rootlessContainerdSockAddress()
		if err != nil {
			// FIXME: say something? Return an err?
			fmt.Fprintln(os.Stdout, "FIXME")
		} else {
			tryAddress = addr
		}
	}

	if err := socket.IsSocketAccessible(tryAddress); err != nil {
		if address != "" || socket.IsSocketAccessible(dockerContainerdaddress) != nil {
			return nil, nil, nil, errors.Join(ErrServiceClient, errs.ErrSystemFailure, ErrSocketNotAccessible, err)
		}

		tryAddress = dockerContainerdaddress
	}

	client, err := containerd.New(tryAddress, clientOpts...)
	if err != nil {
		return nil, nil, nil, errors.Join(ErrServiceClient, errs.ErrSystemFailure, err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(namespace.NamespacedContext(ctx, ns))

	return client, ctx, cancel, nil
}
