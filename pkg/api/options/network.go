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

package options

import (
	"io"
)

// NetworkCreate specifies options for `network create`.
type NetworkCreate struct {
	// GOptions is the global options
	GOptions *Global

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

// NetworkInspect specifies options for `network inspect`.
type NetworkInspect struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions *Global
	// Inspect mode, "dockercompat" for Docker-compatible output, "native" for containerd-native output
	Mode string
	// Format the output using the given Go template, e.g, '{{json .}}'
	Format string
	// Networks are the networks to be inspected
	Networks []string
}

// NetworkList specifies options for `network ls`.
type NetworkList struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions *Global
	// Quiet only show numeric IDs
	Quiet bool
	// Format the output using the given Go template, e.g, '{{json .}}', 'wide'
	Format string
	// Filter matches network based on given conditions
	Filters []string
}

// NetworkPrune specifies options for `network prune`.
type NetworkPrune struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions *Global
	// Network drivers to keep while pruning
	NetworkDriversToKeep []string
}

// NetworkRemove specifies options for `network rm`.
type NetworkRemove struct {
	Stdout io.Writer
	// GOptions is the global options
	GOptions *Global
	// Networks are the networks to be removed
	Networks []string
}
