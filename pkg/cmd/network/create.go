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

package network

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/containerd/errdefs"

	"github.com/containerd/nerdctl/v2/leptonic/identifiers"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/netutil"
)

func Create(ctx context.Context, stdout io.Writer, globalOptions *options.Global, opts *options.NetworkCreate) error {
	if err := identifiers.Validate(opts.Name); err != nil {
		return fmt.Errorf("invalid network name: %w", err)
	}

	if len(opts.Subnets) == 0 {
		if opts.Gateway != "" || opts.IPRange != "" {
			return errors.New("cannot set gateway or ip-range without subnet, specify --subnet manually")
		}

		opts.Subnets = []string{""}
	}

	e, err := netutil.NewCNIEnv(globalOptions.CNIPath, globalOptions.CNINetConfPath, netutil.WithNamespace(globalOptions.Namespace))
	if err != nil {
		return err
	}
	net, err := e.CreateNetwork(*opts)
	if err != nil {
		if errdefs.IsAlreadyExists(err) {
			return fmt.Errorf("network with name %s already exists", opts.Name)
		}
		return err
	}
	_, err = fmt.Fprintln(stdout, *net.CliID)
	return err
}
