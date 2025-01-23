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

package time // import "github.com/containerd/nerdctl/v2/leptonic/time"

import (
	"strconv"
	"testing"
	"time"
)

func TestGetTimestamp(t *testing.T) {
	now := time.Now().In(time.UTC)
	cases := []struct {
		in, expected string
		expectedErr  bool
	}{
		// Partial RFC3339 strings get parsed with second precision
		{"2006-01-02T15:04:05.999999999+07:00", "1136189045.999999999", false},
		{"2006-01-02T15:04:05.999999999Z", "1136214245.999999999", false},
		{"2006-01-02T15:04:05.999999999", "1136214245.999999999", false},
		{"2006-01-02T15:04:05Z", "1136214245.000000000", false},
		{"2006-01-02T15:04:05", "1136214245.000000000", false},
		{"2006-01-02T15:04:0Z", "", true},
		{"2006-01-02T15:04:0", "", true},
		{"2006-01-02T15:04Z", "1136214240.000000000", false},
		{"2006-01-02T15:04+00:00", "1136214240.000000000", false},
		{"2006-01-02T15:04-00:00", "1136214240.000000000", false},
		{"2006-01-02T15:04", "1136214240.000000000", false},
		{"2006-01-02T15:0Z", "", true},
		{"2006-01-02T15:0", "", true},
		{"2006-01-02T15Z", "1136214000.000000000", false},
		{"2006-01-02T15+00:00", "1136214000.000000000", false},
		{"2006-01-02T15-00:00", "1136214000.000000000", false},
		{"2006-01-02T15", "1136214000.000000000", false},
		{"2006-01-02T1Z", "1136163600.000000000", false},
		{"2006-01-02T1", "1136163600.000000000", false},
		{"2006-01-02TZ", "", true},
		{"2006-01-02T", "", true},
		{"2006-01-02+00:00", "1136160000.000000000", false},
		{"2006-01-02-00:00", "1136160000.000000000", false},
		{"2006-01-02-00:01", "1136160060.000000000", false},
		{"2006-01-02Z", "1136160000.000000000", false},
		{"2006-01-02", "1136160000.000000000", false},
		{"2015-05-13T20:39:09Z", "1431549549.000000000", false},

		// unix timestamps returned as is
		{"1136073600", "1136073600", false},
		{"1136073600.000000001", "1136073600.000000001", false},
		// Durations
		{"1m", strconv.FormatInt(now.Add(-1*time.Minute).Unix(), 10), false},
		{"1.5h", strconv.FormatInt(now.Add(-90*time.Minute).Unix(), 10), false},
		{"1h30m", strconv.FormatInt(now.Add(-90*time.Minute).Unix(), 10), false},

		{"invalid", "", true},
		{"", "", true},
	}

	for _, c := range cases {
		o, err := GetTimestamp(c.in, now)
		if o != c.expected ||
			(err == nil && c.expectedErr) ||
			(err != nil && !c.expectedErr) {
			t.Errorf("wrong value for '%s'. expected:'%s' got:'%s' with error: `%s`", c.in, c.expected, o, err)
			t.Fail()
		}
	}
}
