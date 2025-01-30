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

// VolumeCreate specifies options for `volume create`.
type VolumeCreate struct {
	Stdout   io.Writer
	GOptions Global
	// Labels are the volume labels
	Labels []string
}

// VolumeInspect specifies options for `volume inspect`.
type VolumeInspect struct {
	Stdout   io.Writer
	GOptions Global
	// Format the output using the given go template
	Format string
	// Display the disk usage of volumes. Can be slow with volumes having loads of directories.
	Size bool
}

// VolumeList specifies options for `volume ls`.
type VolumeList struct {
	Stdout   io.Writer
	GOptions Global
	// Only display volume names
	Quiet bool
	// Format the output using the given go template
	Format string
	// Display the disk usage of volumes. Can be slow with volumes having loads of directories.
	Size bool
	// Filter matches volumes based on given conditions
	Filters []string
}

// VolumePrune specifies options for `volume prune`.
type VolumePrune struct {
	Stdout   io.Writer
	GOptions Global
	// Remove all unused volumes, not just anonymous ones
	All bool
	// Do not prompt for confirmation
	Force bool
}

// VolumeRemove specifies options for `volume rm`.
type VolumeRemove struct {
	Stdout   io.Writer
	GOptions Global
	// Force the removal of one or more volumes
	Force bool
}
