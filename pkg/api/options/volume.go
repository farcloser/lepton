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

// VolumeCreate specifies options for `volume create`.
type VolumeCreate struct {
	Name   string
	Labels map[string]string
}

// VolumeInspect specifies options for `volume inspect`.
type VolumeInspect struct {
	NamesList []string
	// Format the output using the given go template
	Format string
	// Display the disk usage of volumes. Can be slow with volumes having loads of directories.
	Size bool
}

// VolumeList specifies options for `volume ls`.
type VolumeList struct {
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
	// Remove all unused volumes, not just anonymous ones
	All bool
}

// VolumeRemove specifies options for `volume rm`.
type VolumeRemove struct {
	NamesList []string
}
