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

package container

import (
	"net"

	"github.com/containerd/go-cni"
	"github.com/spf13/cobra"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/portutil"
	"go.farcloser.world/lepton/pkg/strutil"
)

func loadNetworkFlags(cmd *cobra.Command) (options.ContainerNetwork, error) {
	netOpts := options.ContainerNetwork{}

	// --net/--network=<net name> ...
	netSlice := []string{}
	networkSet := false
	if cmd.Flags().Lookup("network").Changed {
		network, err := cmd.Flags().GetStringSlice("network")
		if err != nil {
			return netOpts, err
		}
		netSlice = append(netSlice, network...)
		networkSet = true
	}
	if cmd.Flags().Lookup("net").Changed {
		netFlag, err := cmd.Flags().GetStringSlice("net")
		if err != nil {
			return netOpts, err
		}
		netSlice = append(netSlice, netFlag...)
		networkSet = true
	}

	if !networkSet {
		network, err := cmd.Flags().GetStringSlice("network")
		if err != nil {
			return netOpts, err
		}
		netSlice = append(netSlice, network...)
	}
	netOpts.NetworkSlice = strutil.DedupeStrSlice(netSlice)

	// --mac-address=<MAC>
	macAddress, err := cmd.Flags().GetString("mac-address")
	if err != nil {
		return netOpts, err
	}
	if macAddress != "" {
		if _, err := net.ParseMAC(macAddress); err != nil {
			return netOpts, err
		}
	}
	netOpts.MACAddress = macAddress

	// --ip=<container static IP>
	ipAddress, err := cmd.Flags().GetString("ip")
	if err != nil {
		return netOpts, err
	}
	netOpts.IPAddress = ipAddress

	// --ip6=<container static IP6>
	ip6Address, err := cmd.Flags().GetString("ip6")
	if err != nil {
		return netOpts, err
	}
	netOpts.IP6Address = ip6Address

	// -h/--hostname=<container hostname>
	hostName, err := cmd.Flags().GetString("hostname")
	if err != nil {
		return netOpts, err
	}
	netOpts.Hostname = hostName

	// --domainname=<container domainname>
	domainname, err := cmd.Flags().GetString("domainname")
	if err != nil {
		return netOpts, err
	}
	netOpts.Domainname = domainname

	// --dns=<DNS host> ...
	dnsSlice, err := cmd.Flags().GetStringSlice("dns")
	if err != nil {
		return netOpts, err
	}
	netOpts.DNSServers = strutil.DedupeStrSlice(dnsSlice)

	// --dns-search=<domain name> ...
	dnsSearchSlice, err := cmd.Flags().GetStringSlice("dns-search")
	if err != nil {
		return netOpts, err
	}
	netOpts.DNSSearchDomains = strutil.DedupeStrSlice(dnsSearchSlice)

	// --dns-opt/--dns-option=<resolv.conf line> ...
	dnsOptions := []string{}

	dnsOptFlags, err := cmd.Flags().GetStringSlice("dns-opt")
	if err != nil {
		return netOpts, err
	}
	dnsOptions = append(dnsOptions, dnsOptFlags...)

	dnsOptionFlags, err := cmd.Flags().GetStringSlice("dns-option")
	if err != nil {
		return netOpts, err
	}
	dnsOptions = append(dnsOptions, dnsOptionFlags...)

	netOpts.DNSResolvConfOptions = strutil.DedupeStrSlice(dnsOptions)

	// --add-host=<host:IP> ...
	addHostFlags, err := cmd.Flags().GetStringSlice("add-host")
	if err != nil {
		return netOpts, err
	}
	netOpts.AddHost = addHostFlags

	// --uts=<Unix Time Sharing namespace>
	utsNamespace, err := cmd.Flags().GetString("uts")
	if err != nil {
		return netOpts, err
	}
	netOpts.UTSNamespace = utsNamespace

	// -p/--publish=127.0.0.1:80:8080/tcp ...
	portSlice, err := cmd.Flags().GetStringSlice("publish")
	if err != nil {
		return netOpts, err
	}
	portSlice = strutil.DedupeStrSlice(portSlice)
	portMappings := []cni.PortMapping{}
	for _, p := range portSlice {
		pm, err := portutil.ParseFlagP(p)
		if err != nil {
			return netOpts, err
		}
		portMappings = append(portMappings, pm...)
	}
	netOpts.PortMappings = portMappings

	return netOpts, nil
}
