/*
   Copyright The containerd Authors.

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

package statsutil

import (
	"fmt"
	"strconv"
	"strings"

	"go.farcloser.world/containers/stats"
	"go.farcloser.world/core/units"
)

// Rendering a FormattedStatsEntry from StatsEntry
func RenderEntry(in *stats.Entry) FormattedStatsEntry {
	return FormattedStatsEntry{
		Entry: *in,
	}
}

// FormattedStatsEntry represents a formatted StatsEntry
type FormattedStatsEntry struct {
	stats.Entry
}

func (s *FormattedStatsEntry) Name(noTrunc bool) string {
	if len(s.Entry.Name) > 1 {
		if !noTrunc {
			var truncLen int
			if strings.HasPrefix(s.Entry.Name, "k8s://") {
				truncLen = 24
			} else {
				truncLen = 12
			}
			if len(s.Entry.Name) > truncLen {
				return s.Entry.Name[:truncLen]
			}
		}
		return s.Entry.Name
	}
	return "--"
}

func (s *FormattedStatsEntry) ID(noTrunc bool) string {
	if !noTrunc {
		if len(s.Entry.ID) > 12 {
			return s.Entry.ID[:12]
		}
	}
	return s.Entry.ID
}

func (s *FormattedStatsEntry) CPUPerc() string {
	if s.Entry.IsInvalid {
		return "--"
	}
	return fmt.Sprintf("%.2f%%", s.Entry.CPUPercentage)
}

func (s *FormattedStatsEntry) MemUsage() string {
	if s.Entry.IsInvalid {
		return "-- / --"
	}
	return fmt.Sprintf("%s / %s", units.BytesSize(s.Entry.Memory), units.BytesSize(s.Entry.MemoryLimit))
}

func (s *FormattedStatsEntry) MemPerc() string {
	if s.Entry.IsInvalid {
		return "--"
	}
	return fmt.Sprintf("%.2f%%", s.Entry.MemoryPercentage)
}

func (s *FormattedStatsEntry) NetIO() string {
	if s.Entry.IsInvalid {
		return "--"
	}
	return fmt.Sprintf("%s / %s", units.HumanSizeWithPrecision(s.Entry.NetworkRx, 3), units.HumanSizeWithPrecision(s.Entry.NetworkTx, 3))
}

func (s *FormattedStatsEntry) BlockIO() string {
	if s.Entry.IsInvalid {
		return "--"
	}
	return fmt.Sprintf("%s / %s", units.HumanSizeWithPrecision(s.BlockRead, 3), units.HumanSizeWithPrecision(s.Entry.BlockWrite, 3))
}

func (s *FormattedStatsEntry) PIDs() string {
	if s.Entry.IsInvalid {
		return "--"
	}
	return strconv.FormatUint(s.Entry.PidsCurrent, 10)
}
