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
	"runtime"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/dnsutil/hostsstore"
	"go.farcloser.world/lepton/pkg/idutil/containerwalker"
	"go.farcloser.world/lepton/pkg/labels"
	"go.farcloser.world/lepton/pkg/namestore"
)

// Rename change container name to a new name
// containerID is container name, short ID, or long ID
func Rename(ctx context.Context, client *containerd.Client, containerID, newContainerName string,
	options options.ContainerRename,
) error {
	dataStore, err := clientutil.DataStore(options.GOptions.DataRoot, options.GOptions.Address)
	if err != nil {
		return err
	}
	namest, err := namestore.New(dataStore, options.GOptions.Namespace)
	if err != nil {
		return err
	}
	hostst, err := hostsstore.New(dataStore, options.GOptions.Namespace)
	if err != nil {
		return err
	}
	walker := &containerwalker.ContainerWalker{
		Client: client,
		OnFound: func(ctx context.Context, found containerwalker.Found) error {
			if found.MatchCount > 1 {
				return fmt.Errorf("multiple IDs found with provided prefix: %s", found.Req)
			}
			return renameContainer(ctx, found.Container, newContainerName, namest, hostst)
		},
	}

	if n, err := walker.Walk(ctx, containerID); err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("no such container %s", containerID)
	}
	return nil
}

func renameContainer(ctx context.Context, container containerd.Container, newName string,
	namst namestore.NameStore, hostst hostsstore.Store,
) (err error) {
	l, err := container.Labels(ctx)
	if err != nil {
		return err
	}
	name := l[labels.Name]

	id := container.ID()

	defer func() {
		// If we errored, rollback whatever we can
		if err != nil {
			lbls := map[string]string{
				labels.Name: name,
			}
			namst.Rename(newName, id, name)
			hostst.Update(id, name)
			container.SetLabels(ctx, lbls)
		}
	}()

	if err = namst.Rename(name, id, newName); err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		if err = hostst.Update(id, newName); err != nil {
			log.G(ctx).WithError(err).Warn("failed to update host networking definitions " +
				"- if your container is using network 'none', this is expected - otherwise, please report this as a bug")
		}
	}
	lbls := map[string]string{
		labels.Name: newName,
	}
	if _, err = container.SetLabels(ctx, lbls); err != nil {
		return err
	}
	return nil
}
