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
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/typeurl/v2"

	"go.farcloser.world/containers/stats"
)

func setContainerStatsAndRenderStatsEntry(ctx context.Context, container client.Container, previousStats *stats.ContainerStats) (statsEntry stats.Entry, err error) {
	task, err := container.Task(ctx, nil)
	if err != nil {
		return statsEntry, err
	}

	pid := int(task.Pid())

	metric, err := task.Metrics(ctx)
	if err != nil {
		return statsEntry, err
	}
	anydata, err := typeurl.UnmarshalAny(metric.Data)
	if err != nil {
		return statsEntry, err
	}

	statsEntry, err = stats.SetCgroup2StatsFields(previousStats, anydata, pid)

	previousStats.Time = time.Now()
	statsEntry.ID = container.ID()

	return statsEntry, err
}
