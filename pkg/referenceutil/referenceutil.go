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

package referenceutil

import (
	"fmt"
	"path"

	"github.com/distribution/reference"
)

// Reference is a reference to an image.
type Reference interface {

	// String returns the full reference which can be understood by containerd.
	String() string
}

// ParseAnyReference parses the passed reference as CID, or a classic reference.
// Unlike ParseAny, it is not limited to the DockerRef limitations (being either tagged or digested)
// and should be used instead.
func ParseAnyReference(rawRef string) (Reference, error) {
	return reference.ParseAnyReference(rawRef)
}

// ParseAny parses the passed reference with allowing it to be non-docker reference.
// Otherwise, it's parsed as a docker reference.
func ParseAny(rawRef string) (Reference, error) {
	return ParseDockerRef(rawRef)
}

// ParseDockerRef parses the passed reference with assuming it's a docker reference.
func ParseDockerRef(rawRef string) (reference.Named, error) {
	return reference.ParseDockerRef(rawRef)
}

type stringRef struct {
	scheme string
	s      string
}

func (s stringRef) String() string {
	return s.s
}

// SuggestContainerName generates a container name from name.
// The result MUST NOT be parsed.
func SuggestContainerName(rawRef, containerID string) string {
	const shortIDLength = 5
	if len(containerID) < shortIDLength {
		panic(fmt.Errorf("got too short (< %d) container ID: %q", shortIDLength, containerID))
	}
	name := "untitled-" + containerID[:shortIDLength]
	if rawRef != "" {
		r, err := ParseAny(rawRef)
		if err == nil {
			switch rr := r.(type) {
			case reference.Named:
				if rrName := rr.Name(); rrName != "" {
					imageNameBased := path.Base(rrName)
					if imageNameBased != "" {
						name = imageNameBased + "-" + containerID[:shortIDLength]
					}
				}
			case stringRef:
				name = rr.scheme + "-" + rr.s[:shortIDLength] + "-" + containerID[:shortIDLength]
			}
		}
	}
	return name
}
