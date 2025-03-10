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

package nettype_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/lepton/pkg/netutil/nettype"
)

func TestDetect(t *testing.T) {
	type testCase struct {
		names    []string
		expected nettype.Type
		err      string
	}
	testCases := []testCase{
		{
			names:    nil,
			expected: nettype.CNI,
		},
		{
			names:    []string{"none"},
			expected: nettype.None,
		},
		{
			names:    []string{"host"},
			expected: nettype.Host,
		},
		{
			names:    []string{"bridge"},
			expected: nettype.CNI,
		},
		{
			names:    []string{"foo", "bar"},
			expected: nettype.CNI,
		},
		{
			names:    []string{"foo", "bar", "bridge"},
			expected: nettype.CNI,
		},
		{
			names: []string{"none", "host"},
			err:   "mixed network types",
		},
		{
			names: []string{"none", "bridge"},
			err:   "mixed network types",
		},
		{
			names: []string{"host", "foo"},
			err:   "mixed network types",
		},
	}

	for _, tc := range testCases {
		got, err := nettype.Detect(tc.names)
		if tc.err == "" {
			assert.NilError(t, err)
			assert.Equal(t, tc.expected, got)
		} else {
			assert.ErrorContains(t, err, tc.err)
		}
	}
}
