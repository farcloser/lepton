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

package namespace

import (
	"context"
	"errors"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/leptonic/services/namespace"
	"go.farcloser.world/lepton/pkg/api/options"
)

func Remove(ctx context.Context, client *client.Client, _ *options.Global, opts *options.NamespaceRemove) error {
	errs := namespace.Remove(ctx, client, opts.NamesList, opts.CGroup)
	if len(errs) > 0 {
		for _, err := range errs {
			log.G(ctx).WithError(err).Error()
		}

		return errors.New("error while removing namespaces")
	}

	return nil
}
