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
	"strings"

	containerd "github.com/containerd/containerd/v2/client"

	"go.farcloser.world/lepton/leptonic/container"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
)

// Top performs the equivalent of running `top` inside of container(s)
func Top(ctx context.Context, client *containerd.Client, containers []string, opt options.ContainerTop) error {
	walker := &containerwalker.ContainerWalker{
		Client: client,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			if found.MatchCount > 1 {
				return fmt.Errorf("multiple IDs found with provided prefix: %s", found.Req)
			}
			return container.Top(ctx, opt.Stdout, client, found.Container.ID(), strings.Join(containers[1:], " "))
		},
	}

	n, err := walker.Walk(ctx, containers[0])
	if err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("no such container %s", containers[0])
	}
	return nil
}
