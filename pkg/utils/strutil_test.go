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

package utils_test

import (
	"reflect"
	"testing"

	"go.farcloser.world/lepton/pkg/utils"
)

func TestConvertKVStringsToMap(t *testing.T) {
	t.Parallel()

	type args struct {
		values []string
	}

	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "normal",
			args: args{
				values: []string{"foo=bar", "baz=qux"},
			},
			want: map[string]string{
				"foo": "bar",
				"baz": "qux",
			},
		},
		{
			name: "normal-1",
			args: args{
				values: []string{"foo"},
			},
			want: map[string]string{
				"foo": "",
			},
		},
		{
			name: "normal-2",
			args: args{
				values: []string{"foo=bar=baz"},
			},
			want: map[string]string{
				"foo": "bar=baz",
			},
		},
		{
			name: "empty",
			args: args{
				values: []string{},
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := utils.KeyValueStringsToMap(tt.args.values); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertKVStringsToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
