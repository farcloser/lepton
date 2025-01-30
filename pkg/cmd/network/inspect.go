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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/formatter"
	"github.com/containerd/nerdctl/v2/pkg/inspecttypes/dockercompat"
	"github.com/containerd/nerdctl/v2/pkg/inspecttypes/native"
	"github.com/containerd/nerdctl/v2/pkg/netutil"
)

func Inspect(ctx context.Context, output io.Writer, globalOptions *options.Global, opts *options.NetworkInspect) error {
	if opts.Mode != "native" && opts.Mode != "dockercompat" {
		return fmt.Errorf("unknown mode %q", opts.Mode)
	}

	cniEnv, err := netutil.NewCNIEnv(globalOptions.CNIPath, globalOptions.CNINetConfPath, netutil.WithNamespace(globalOptions.Namespace))
	if err != nil {
		return err
	}

	var result []interface{}
	netLists, errs := cniEnv.ListNetworksMatch(opts.Networks, true)

	for req, netList := range netLists {
		if len(netList) > 1 {
			errs = append(errs, fmt.Errorf("multiple IDs found with provided prefix: %s", req))
			continue
		}
		if len(netList) == 0 {
			errs = append(errs, fmt.Errorf("no network found matching: %s", req))
			continue
		}
		network := netList[0]
		r := &native.Network{
			CNI:       json.RawMessage(network.Bytes),
			CliID:     network.CliID,
			CliLabels: network.CliLabels,
			File:      network.File,
		}
		switch opts.Mode {
		case "native":
			result = append(result, r)
		case "dockercompat":
			compat, err := dockercompat.NetworkFromNative(r)
			if err != nil {
				return err
			}
			result = append(result, compat)
		}
	}

	if len(result) > 0 {
		if formatErr := formatter.FormatSlice(opts.Format, output, result); formatErr != nil {
			log.G(ctx).Error(formatErr)
		}
		err = nil
	} else {
		err = errors.New("unable to find any network matching the provided request")
	}

	for _, unErr := range errs {
		log.G(ctx).Error(unErr)
	}

	return err
}
