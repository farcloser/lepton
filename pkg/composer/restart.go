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

package composer

import (
	"context"
	"fmt"
	"sync"

	"github.com/compose-spec/compose-go/v2/types"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/labels"
)

// RestartOptions stores all option input from `compose restart`
type RestartOptions struct {
	Timeout *uint
}

// Restart restarts running/stopped containers in `services`. It calls
// `restart CONTAINER_ID` to do the actual job.
func (c *Composer) Restart(ctx context.Context, opt RestartOptions, services []string) error {
	// in dependency order
	return c.project.ForEachService(services, func(name string, svc *types.ServiceConfig) error {
		containers, err := c.Containers(ctx, svc.Name)
		if err != nil {
			return err
		}

		return c.restartContainers(ctx, containers, opt)
	})
}

func (c *Composer) restartContainers(ctx context.Context, containers []containerd.Container, opt RestartOptions) error {
	var timeoutArg string
	if opt.Timeout != nil {
		// `restart` uses `--time` instead of `--timeout`
		timeoutArg = fmt.Sprintf("--time=%d", *opt.Timeout)
	}

	var rsWG sync.WaitGroup
	for _, container := range containers {
		rsWG.Add(1)
		go func() {
			defer rsWG.Done()
			info, _ := container.Info(ctx, containerd.WithoutRefreshedMetadata)
			log.G(ctx).Infof("Restarting container %s", info.Labels[labels.Name])
			args := []string{"restart"}
			if opt.Timeout != nil {
				args = append(args, timeoutArg)
			}
			args = append(args, container.ID())
			if err := c.runCliCmd(ctx, args...); err != nil {
				log.G(ctx).Warn(err)
			}
		}()
	}
	rsWG.Wait()

	return nil
}
