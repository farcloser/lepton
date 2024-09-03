/*
   Copyright The containerd Authors.

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

package types

import (
	"io"
)

// NetworkCreateOptions specifies options for `network create`.
type NetworkCreateOptions struct {
	// GOptions is the global options
	GOptions GlobalCommandOptions

	Name        string
	Driver      string
	Options     map[string]string
	IPAMDriver  string
	IPAMOptions map[string]string
	Subnets     []string
	Gateway     string
	IPRange     string
	Labels      []string
	IPv6        bool
}

// NetworkInspectOptions specifies options for `network inspect`.
type NetworkInspectOptions struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions GlobalCommandOptions
	// Inspect mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output
	Mode string
	// Format the output using the given Go template, e.g, '{{json .}}'
	Format string
	// Networks are the networks to be inspected
	Networks []string
}

// NetworkListOptions specifies options for `network ls`.
type NetworkListOptions struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions GlobalCommandOptions
	// Quiet only show numeric IDs
	Quiet bool
	// Format the output using the given Go template, e.g, '{{json .}}', 'wide'
	Format string
	// Filter matches network based on given conditions
	Filters []string
}

// NetworkPruneOptions specifies options for `network prune`.
type NetworkPruneOptions struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions GlobalCommandOptions
	// Network drivers to keep while pruning
	NetworkDriversToKeep []string
}

// NetworkRemoveOptions specifies options for `network rm`.
type NetworkRemoveOptions struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions GlobalCommandOptions
	// Networks are the networks to be removed
	Networks []string
}
