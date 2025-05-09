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

package container

import (
	"context"
	"fmt"

	containerd "github.com/containerd/containerd/v2/client"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/containerutil"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
)

// Restart will restart one or more containers.
func Restart(
	ctx context.Context,
	client *containerd.Client,
	containers []string,
	options options.ContainerRestart,
) error {
	walker := &containerwalker.ContainerWalker{
		Client: client,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			if found.MatchCount > 1 {
				return fmt.Errorf("multiple IDs found with provided prefix: %s", found.Req)
			}
			if err := containerutil.Stop(ctx, found.Container, options.Timeout, options.Signal); err != nil {
				return err
			}
			if err := containerutil.Start(ctx, found.Container, false, client, ""); err != nil {
				return err
			}
			_, err := fmt.Fprintln(options.Stdout, found.Req)
			return err
		},
	}

	return walker.WalkAll(ctx, containers, true)
}
