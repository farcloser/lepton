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
	"github.com/containerd/nerdctl/v2/pkg/netutil"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

func createCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:           "create [flags] NETWORK",
		Short:         "Create a network",
		Long:          `NOTE: To isolate CNI bridge, CNI plugin "firewall" (>= v1.6.0) is needed.`,
		Args:          helpers.IsExactArgs(1),
		RunE:          createAction,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringP(flagDriver, "d", netutil.DefaultNetworkName, "Driver to manage the Network")
	cmd.Flags().StringArrayP(flagOpt, "o", nil, "Set driver specific options")
	cmd.Flags().String(flagIpamDriver, "default", "IP Address helpers.Management Driver")
	cmd.Flags().StringArray(flagIpamOpt, nil, "Set IPAM driver specific options")
	cmd.Flags().StringArray(flagSubnet, nil, `Subnet in CIDR format that represents a network segment, e.g. "10.5.0.0/16"`)
	cmd.Flags().String(flagGateway, "", `Gateway for the master subnet`)
	cmd.Flags().String(flagIPRange, "", `Allocate container ip from a sub-range`)
	cmd.Flags().StringArray(flagLabel, nil, "Set metadata for a network")
	cmd.Flags().Bool(flagIPv6, false, "Enable IPv6 networking")

	_ = cmd.RegisterFlagCompletionFunc(flagDriver, completion.NetworkDrivers)
	_ = cmd.RegisterFlagCompletionFunc(flagIpamDriver, completion.IPAMDrivers)

	return cmd
}

func createOptions(cmd *cobra.Command, args []string) (*options.NetworkCreate, error) {
	name := args[0]

	driver, err := cmd.Flags().GetString(flagDriver)
	if err != nil {
		return nil, err
	}

	opts, err := cmd.Flags().GetStringArray(flagOpt)
	if err != nil {
		return nil, err
	}

	ipamDriver, err := cmd.Flags().GetString(flagIpamDriver)
	if err != nil {
		return nil, err
	}

	ipamOpts, err := cmd.Flags().GetStringArray(flagIpamOpt)
	if err != nil {
		return nil, err
	}

	subnets, err := cmd.Flags().GetStringArray(flagSubnet)
	if err != nil {
		return nil, err
	}

	gatewayStr, err := cmd.Flags().GetString(flagGateway)
	if err != nil {
		return nil, err
	}

	ipRangeStr, err := cmd.Flags().GetString(flagIPRange)
	if err != nil {
		return nil, err
	}

	labels, err := cmd.Flags().GetStringArray(flagLabel)
	if err != nil {
		return nil, err
	}

	labels = strutil.DedupeStrSlice(labels)
	ipv6, err := cmd.Flags().GetBool(flagIPv6)
	if err != nil {
		return nil, err
	}

	return &options.NetworkCreate{
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
	}, nil
}

func createAction(cmd *cobra.Command, args []string) error {
	globalOptions, err := helpers.ProcessRootCmdFlags(cmd)
	if err != nil {
		return err
	}

	opts, err := createOptions(cmd, args)
	if err != nil {
		return err
	}

	return network.Create(cmd.Context(), cmd.OutOrStdout(), globalOptions, opts)
}
