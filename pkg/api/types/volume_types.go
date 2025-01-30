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

package types

import (
	"io"

	"github.com/containerd/nerdctl/v2/pkg/api/options"
)

// VolumeCreateOptions specifies options for `volume create`.
type VolumeCreateOptions struct {
	Stdout   io.Writer
	GOptions options.Global
	// Labels are the volume labels
	Labels []string
}

// VolumeInspectOptions specifies options for `volume inspect`.
type VolumeInspectOptions struct {
	Stdout   io.Writer
	GOptions options.Global
	// Format the output using the given go template
	Format string
	// Display the disk usage of volumes. Can be slow with volumes having loads of directories.
	Size bool
}

// VolumeListOptions specifies options for `volume ls`.
type VolumeListOptions struct {
	Stdout   io.Writer
	GOptions options.Global
	// Only display volume names
	Quiet bool
	// Format the output using the given go template
	Format string
	// Display the disk usage of volumes. Can be slow with volumes having loads of directories.
	Size bool
	// Filter matches volumes based on given conditions
	Filters []string
}

// VolumePruneOptions specifies options for `volume prune`.
type VolumePruneOptions struct {
	Stdout   io.Writer
	GOptions options.Global
	// Remove all unused volumes, not just anonymous ones
	All bool
	// Do not prompt for confirmation
	Force bool
}

// VolumeRemoveOptions specifies options for `volume rm`.
type VolumeRemoveOptions struct {
	Stdout   io.Writer
	GOptions options.Global
	// Force the removal of one or more volumes
	Force bool
}
