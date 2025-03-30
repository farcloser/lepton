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
	"errors"
	"fmt"
	"io"

	"github.com/containerd/errdefs"

	"go.farcloser.world/lepton/leptonic/identifiers"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/netutil"
)

func Create(output io.Writer, globalOption *options.Global, options *options.NetworkCreate) error {
	if err := identifiers.Validate(options.Name); err != nil {
		return fmt.Errorf("invalid network name: %w", err)
	}

	if len(options.Subnets) == 0 {
		if options.Gateway != "" || options.IPRange != "" {
			return errors.New("cannot set gateway or ip-range without subnet, specify --subnet manually")
		}
		options.Subnets = []string{""}
	}

	e, err := netutil.NewCNIEnv(
		globalOption.CNIPath,
		globalOption.CNINetConfPath,
		netutil.WithNamespace(globalOption.Namespace),
	)
	if err != nil {
		return err
	}
	net, err := e.CreateNetwork(options)
	if err != nil {
		if errdefs.IsAlreadyExists(err) {
			return fmt.Errorf("network with name %s already exists", options.Name)
		}
		return err
	}
	_, err = fmt.Fprintln(output, *net.CliID)
	return err
}
