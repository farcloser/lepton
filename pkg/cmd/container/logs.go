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
	"os"
	"os/signal"
	"syscall"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/api/types/cri"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
	"go.farcloser.world/lepton/pkg/labels"
	"go.farcloser.world/lepton/pkg/labels/k8slabels"
	"go.farcloser.world/lepton/pkg/logging"
)

func Logs(ctx context.Context, client *containerd.Client, container string, options options.ContainerLogs) error {
	dataStore, err := clientutil.DataStore(options.GOptions.DataRoot, options.GOptions.Address)
	if err != nil {
		return err
	}

	if options.GOptions.Namespace == "moby" {
		log.G(ctx).Warn("Currently, `logs` only supports containers created with `run -d` or CRI")
	}

	stopChannel := make(chan os.Signal, 1)
	// catch OS signals:
	signal.Notify(stopChannel, syscall.SIGTERM, syscall.SIGINT)

	walker := &containerwalker.ContainerWalker{
		Client: client,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			if found.MatchCount > 1 {
				return fmt.Errorf("multiple IDs found with provided prefix: %s", found.Req)
			}
			l, err := found.Container.Labels(ctx)
			if err != nil {
				return err
			}

			logPath, err := getLogPath(ctx, found.Container)
			if err != nil {
				return err
			}

			follow := options.Follow
			if follow {
				task, err := found.Container.Task(ctx, nil)
				if err != nil {
					if !errdefs.IsNotFound(err) {
						return err
					}
					follow = false
				} else {
					status, err := task.Status(ctx)
					if err != nil {
						return err
					}
					if status.Status != containerd.Running {
						follow = false
					} else {
						waitCh, err := task.Wait(ctx)
						if err != nil {
							return fmt.Errorf("failed to get wait channel for task %#v: %w", task, err)
						}

						// Setup goroutine to send stop event if container task finishes:
						go func() {
							<-waitCh
							log.G(ctx).Debugf("container task has finished, sending kill signal to log viewer")
							stopChannel <- os.Interrupt
						}()
					}
				}
			}

			logViewOpts := logging.LogViewOptions{
				ContainerID:       found.Container.ID(),
				Namespace:         l[labels.Namespace],
				DatastoreRootPath: dataStore,
				LogPath:           logPath,
				Follow:            follow,
				Timestamps:        options.Timestamps,
				Tail:              options.Tail,
				Since:             options.Since,
				Until:             options.Until,
			}
			logViewer, err := logging.InitContainerLogViewer(l, logViewOpts, stopChannel, options.GOptions.Experimental)
			if err != nil {
				return err
			}

			return logViewer.PrintLogsTo(options.Stdout, options.Stderr)
		},
	}
	n, err := walker.Walk(ctx, container)
	if err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("no such container %s", container)
	}
	return nil
}

func getLogPath(ctx context.Context, container containerd.Container) (string, error) {
	extensions, err := container.Extensions(ctx)
	if err != nil {
		return "", fmt.Errorf("get extensions for container %s,failed: %w", container.ID(), err)
	}
	metaData := extensions[k8slabels.ContainerMetadataExtension]
	var meta cri.ContainerMetadata
	if metaData != nil {
		err = meta.UnmarshalJSON(metaData.GetValue())
		if err != nil {
			return "", fmt.Errorf("unmarshal extensions for container %s,failed: %w", container.ID(), err)
		}
	}

	return meta.LogPath, nil
}
