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

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/composer/serviceparser"
	"go.farcloser.world/lepton/pkg/labels"
	"go.farcloser.world/lepton/pkg/strutil"
)

// StopOptions stores all option input from `compose stop`
type StopOptions struct {
	Timeout *uint
}

// Stop stops containers in `services` without removing them. It calls
// `stop CONTAINER_ID` to do the actual job.
func (c *Composer) Stop(ctx context.Context, opt StopOptions, services []string) error {
	serviceNames, err := c.ServiceNames(services...)
	if err != nil {
		return err
	}
	// reverse dependency order
	for _, svc := range strutil.ReverseStrSlice(serviceNames) {
		containers, err := c.Containers(ctx, svc)
		if err != nil {
			return err
		}
		if err := c.stopContainers(ctx, containers, opt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Composer) stopContainers(ctx context.Context, containers []containerd.Container, opt StopOptions) error {
	var timeoutArg string
	if opt.Timeout != nil {
		// `stop` uses `--time` instead of `--timeout`
		timeoutArg = fmt.Sprintf("--time=%d", *opt.Timeout)
	}

	var rmWG sync.WaitGroup
	for _, container := range containers {
		rmWG.Add(1)
		go func() {
			defer rmWG.Done()
			info, _ := container.Info(ctx, containerd.WithoutRefreshedMetadata)
			log.G(ctx).Infof("Stopping container %s", info.Labels[labels.Name])
			args := []string{"stop"}
			if opt.Timeout != nil {
				args = append(args, timeoutArg)
			}
			args = append(args, container.ID())
			if err := c.runCliCmd(ctx, args...); err != nil {
				log.G(ctx).Warn(err)
			}
		}()
	}
	rmWG.Wait()

	return nil
}

func (c *Composer) stopContainersFromParsedServices(
	ctx context.Context,
	containers map[string]serviceparser.Container,
) {
	var rmWG sync.WaitGroup
	for id, container := range containers {
		rmWG.Add(1)
		go func() {
			defer rmWG.Done()
			log.G(ctx).Infof("Stopping container %s", container.Name)
			if err := c.runCliCmd(ctx, "stop", id); err != nil {
				log.G(ctx).Warn(err)
			}
		}()
	}
	rmWG.Wait()
}
