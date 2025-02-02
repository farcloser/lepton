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
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/netutil"
	"go.farcloser.world/lepton/pkg/strutil"
)

func Prune(ctx context.Context, client *containerd.Client, globalOptions *options.Global, opts *options.NetworkPrune) error {
	e, err := netutil.NewCNIEnv(globalOptions.CNIPath, globalOptions.CNINetConfPath, netutil.WithNamespace(globalOptions.Namespace))
	if err != nil {
		return err
	}

	usedNetworks, err := netutil.UsedNetworks(ctx, client)
	if err != nil {
		return err
	}

	networkConfigs, err := e.NetworkList()
	if err != nil {
		return err
	}

	var removedNetworks []string //nolint:prealloc
	for _, net := range networkConfigs {
		if strutil.InStringSlice(opts.NetworkDriversToKeep, net.Name) {
			continue
		}
		if net.CliID == nil || net.File == "" {
			continue
		}
		if _, ok := usedNetworks[net.Name]; ok {
			continue
		}
		if err := e.RemoveNetwork(net); err != nil {
			log.G(ctx).WithError(err).Errorf("failed to remove network %s", net.Name)
			continue
		}
		removedNetworks = append(removedNetworks, net.Name)
	}

	if len(removedNetworks) > 0 {
		fmt.Fprintln(opts.Stdout, "Deleted Networks:")
		for _, name := range removedNetworks {
			fmt.Fprintln(opts.Stdout, name)
		}
		fmt.Fprintln(opts.Stdout, "")
	}
	return nil
}
