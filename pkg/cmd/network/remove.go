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

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/netutil"
)

func Remove(ctx context.Context, cli *client.Client, globalOptions *options.Global, opts *options.NetworkRemove) error {
	cniEnv, err := netutil.NewCNIEnv(globalOptions.CNIPath, globalOptions.CNINetConfPath, netutil.WithNamespace(globalOptions.Namespace))
	if err != nil {
		return err
	}

	usedNetworkInfo, err := netutil.UsedNetworks(ctx, cli)
	if err != nil {
		return err
	}

	var result []string
	netLists, errs := cniEnv.ListNetworksMatch(opts.Networks, false)

	for req, netList := range netLists {
		if len(netList) > 1 {
			errs = append(errs, fmt.Errorf("multiple IDs found with provided prefix: %s", req))
			continue
		}
		if len(netList) == 0 {
			errs = append(errs, fmt.Errorf("no network found matching: %s", req))
			continue
		}
		network := netList[0]
		if value, ok := usedNetworkInfo[network.Name]; ok {
			errs = append(errs, fmt.Errorf("network %q is in use by container %q", req, value))
			continue
		}
		if network.Name == "bridge" {
			errs = append(errs, errors.New("cannot remove pre-defined network bridge"))
			continue
		}
		if network.File == "" {
			errs = append(errs, fmt.Errorf("%s is a pre-defined network and cannot be removed", req))
			continue
		}
		if network.CliID == nil {
			errs = append(errs, fmt.Errorf("%s is managed outside nerdctl and cannot be removed", req))
			continue
		}
		if err := cniEnv.RemoveNetwork(network); err != nil {
			errs = append(errs, err)
		} else {
			result = append(result, req)
		}
	}
	for _, unErr := range errs {
		log.G(ctx).Error(unErr)
	}
	if len(result) > 0 {
		for _, id := range result {
			fmt.Fprintln(opts.Stdout, id)
		}
		err = nil
	} else {
		err = errors.New("no network could be removed")
	}

	return err
}
