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

// Package labels defines labels that are set to containerd containers as labels.
// The labels defined in this package are also passed to OCI containers as annotations.
package labels

import "go.farcloser.world/lepton/pkg/version"

const (
	// ComposeProject Name
	ComposeProject = "com.docker.compose.project"

	// ComposeService Name
	ComposeService = "com.docker.compose.service"

	// ComposeNetwork Name
	ComposeNetwork = "com.docker.compose.network"

	// ComposeVolume Name
	ComposeVolume = "com.docker.compose.volume"
)

var (
	// Prefix is the common prefix for labels
	Prefix = version.RootName + "/"

	// Namespace is the containerd namespace such as "default", "k8s.io"
	Namespace = Prefix + "namespace"

	// Name is a human-friendly name.
	// WARNING: multiple containers may have same the name label
	Name = Prefix + "name"

	Hostname = Prefix + "hostname"

	// Domainname
	Domainname = Prefix + "domainname"

	// ExtraHosts are HostIPs to appended to /etc/hosts
	ExtraHosts = Prefix + "extraHosts"

	// StateDir is "/var/lib/<ROOT_NAME>/<ADDRHASH>/containers/<NAMESPACE>/<ID>"
	StateDir = Prefix + "state-dir"

	// Networks is a JSON-marshalled string of []string, e.g. []string{"bridge"}.
	// Currently, the length of the slice must be 1.
	Networks = Prefix + "networks"

	// Ports is a JSON-marshalled string of []cni.PortMapping .
	Ports = Prefix + "ports"

	// IPAddress is the static IP address of the container assigned by the user
	IPAddress = Prefix + "ip"

	// IP6Address is the static IP6 address of the container assigned by the user
	IP6Address = Prefix + "ip6"

	// LogURI is the log URI
	LogURI = Prefix + "log-uri"

	// PIDFile is the `run --pidfile`
	// (CLI flag is "pidfile", not "pid-file", for Podman compatibility)
	PIDFile = Prefix + "pid-file"

	// AnonymousVolumes is a JSON-marshalled string of []string
	AnonymousVolumes = Prefix + "anonymous-volumes"

	// Platform is the normalized platform string like "linux/ppc64le".
	Platform = Prefix + "platform"

	// Mounts is the mount points for the container.
	Mounts = Prefix + "mounts"

	// StopTimeout is seconds to wait for stop a container.
	StopTimeout = Prefix + "stop-timeout"

	MACAddress = Prefix + "mac-address"

	// PIDContainer is the `run --pid` for restarting
	PIDContainer = Prefix + "pid-container"

	// IPC is the `nerectl run --ipc` for restrating
	// IPC indicates ipc victim container.
	IPC = Prefix + "ipc"

	// Error encapsulates a container human-readable string
	// that describes container error.
	Error = Prefix + "error"

	// DefaultNetwork indicates whether a network is the default network
	// created and owned by the cli.
	// Boolean value which can be parsed with strconv.ParseBool() is required.
	// (like "prefix/default-network=true" or "prefix/default-network=false")
	DefaultNetwork = Prefix + "default-network"

	// ContainerAutoRemove is to check whether the --rm option is specified.
	ContainerAutoRemove = Prefix + "auto-remove"
)
