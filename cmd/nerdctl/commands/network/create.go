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
	"github.com/spf13/cobra"
	"go.farcloser.world/core/utils"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
	"github.com/containerd/nerdctl/v2/cmd/nerdctl/helpers"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/cmd/network"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

func createCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "create [flags] NETWORK",
		Short:         "Create a network",
		Long:          `NOTE: To isolate CNI bridge, CNI plugin "firewall" (>= v1.1.0) is needed.`,
		Args:          helpers.IsExactArgs(1),
		RunE:          createAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringP("driver", "d", DefaultNetworkDriver, "Driver to manage the Network")
	cmd.Flags().StringArrayP("opt", "o", nil, "Set driver specific options")
	cmd.Flags().String("ipam-driver", "default", "IP Address helpers.Management Driver")
	cmd.Flags().StringArray("ipam-opt", nil, "Set IPAM driver specific options")
	cmd.Flags().StringArray("subnet", nil, `Subnet in CIDR format that represents a network segment, e.g. "10.5.0.0/16"`)
	cmd.Flags().String("gateway", "", `Gateway for the master subnet`)
	cmd.Flags().String("ip-range", "", `Allocate container ip from a sub-range`)
	cmd.Flags().StringArray("label", nil, "Set metadata for a network")
	cmd.Flags().Bool("ipv6", false, "Enable IPv6 networking")

	_ = cmd.RegisterFlagCompletionFunc("driver", completion.NetworkDrivers)
	_ = cmd.RegisterFlagCompletionFunc("ipam-driver", completion.IPAMDrivers)

	return cmd
}

func createAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	driver, err := cmd.Flags().GetString("driver")
	if err != nil {
		return err
	}

	opts, err := cmd.Flags().GetStringArray("opt")
	if err != nil {
		return err
	}

	ipamDriver, err := cmd.Flags().GetString("ipam-driver")
	if err != nil {
		return err
	}

	ipamOpts, err := cmd.Flags().GetStringArray("ipam-opt")
	if err != nil {
		return err
	}

	subnets, err := cmd.Flags().GetStringArray("subnet")
	if err != nil {
		return err
	}

	gatewayStr, err := cmd.Flags().GetString("gateway")
	if err != nil {
		return err
	}

	ipRangeStr, err := cmd.Flags().GetString("ip-range")
	if err != nil {
		return err
	}

	labels, err := cmd.Flags().GetStringArray("label")
	if err != nil {
		return err
	}

	labels = strutil.DedupeStrSlice(labels)

	ipv6, err := cmd.Flags().GetBool("ipv6")
	if err != nil {
		return err
	}

	return network.Create(cmd.OutOrStdout(), globalOptions, &options.NetworkCreate{
		Name:        name,
		Driver:      driver,
		Options:     utils.KeyValueStringsToMap(opts),
		IPAMDriver:  ipamDriver,
		IPAMOptions: utils.KeyValueStringsToMap(ipamOpts),
		Subnets:     subnets,
		Gateway:     gatewayStr,
		IPRange:     ipRangeStr,
		Labels:      labels,
		IPv6:        ipv6,
	})
}
