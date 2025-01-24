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

package testtooling

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Microsoft/hcsshim"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	hypervContainer     bool
	hypervSupported     bool
	hypervSupportedOnce sync.Once
)

// HyperVSupported is a test helper to check if hyperv is enabled on
// the host. This can be used to skip tests that require virtualization.
func HyperVSupported() bool {
	if s := os.Getenv("NO_HYPERV"); s != "" {
		if b, err := strconv.ParseBool(s); err == nil && b {
			return false
		}
	}
	hypervSupportedOnce.Do(func() {
		// Hyper-V Virtual Machine Management service name
		const hypervServiceName = "vmms"

		m, err := mgr.Connect()
		if err != nil {
			return
		}
		defer m.Disconnect()

		s, err := m.OpenService(hypervServiceName)
		// hyperv service was present
		if err == nil {
			hypervSupported = true
			s.Close()
		}
	})
	return hypervSupported
}

// HyperVContainer is a test helper to check if the container is a
// hyperv type container, lists only running containers
func HyperVContainer(inspectID string) (bool, error) {
	query := hcsshim.ComputeSystemQuery{}
	containersList, err := hcsshim.GetContainers(query)
	if err != nil {
		hypervContainer = false
		return hypervContainer, err
	}

	for _, container := range containersList {
		// have to use IDs, not all containers have name set
		if strings.Contains(container.ID, inspectID) {
			if container.SystemType == "VirtualMachine" {
				hypervContainer = true
			}
		}
	}

	return hypervContainer, nil
}

// Checks whether an HNS endpoint with a name matching exists.
func ListHnsEndpointsRegex(hnsEndpointNameRegex string) ([]hcsshim.HNSEndpoint, error) {
	r, err := regexp.Compile(hnsEndpointNameRegex)
	if err != nil {
		return nil, err
	}
	hnsEndpoints, err := hcsshim.HNSListEndpointRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to list HNS endpoints for request: %w", err)
	}

	res := []hcsshim.HNSEndpoint{}
	for _, endp := range hnsEndpoints {
		if r.MatchString(endp.Name) {
			res = append(res, endp)
		}
	}
	return res, nil
}
