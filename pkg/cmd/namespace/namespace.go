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

	"github.com/containerd/nerdctl/v2/leptonic/services/namespace"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
)

func Update(ctx context.Context, cli *client.Client, globalOptions *options.Global, opts *options.NamespaceUpdate) error {
	errs := namespace.Update(ctx, cli, opts.Name, opts.Labels)
	if len(errs) > 0 {
		for _, err := range errs {
			log.G(ctx).WithError(err).Error()
		}

		return errors.New("an error occurred")
	}

	return nil
}

func Create(ctx context.Context, cli *client.Client, globalOptions *options.Global, opts *options.NamespaceCreate) error {
	return namespace.Create(ctx, cli, opts.Name, opts.Labels)
}

func Remove(ctx context.Context, client *client.Client, globalOptions *options.Global, opts *options.NamespaceRemove) error {
	errs := namespace.Remove(ctx, client, opts.NamesList, opts.CGroup)

	if len(errs) > 0 {
		for _, err := range errs {
			log.G(ctx).WithError(err).Error()
		}

		return errors.New("failed to remove namespaces")
	}

	return nil
}
