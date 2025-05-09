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
	"errors"
	"fmt"
	"io"
	"strings"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/log"

	"go.farcloser.world/lepton/pkg/api/options"
)

// Prune remove all stopped containers
func Prune(
	ctx context.Context,
	client *containerd.Client,
	output io.Writer,
	globalOptions *options.Global,
	options *options.ContainerPrune,
) error {
	containers, err := client.Containers(ctx)
	if err != nil {
		return err
	}

	var deleted []string
	for _, c := range containers {
		if err = RemoveContainer(ctx, c, globalOptions, false, true, client); err == nil {
			deleted = append(deleted, c.ID())
			continue
		}
		if errors.As(err, &StatusError{}) {
			continue
		}
		log.G(ctx).WithError(err).Warnf("failed to remove container %s", c.ID())
	}

	if len(deleted) > 0 {
		fmt.Fprintln(output, "Deleted Containers:")
		fmt.Fprintln(output, strings.Join(deleted, "\n"))
	}

	return nil
}
