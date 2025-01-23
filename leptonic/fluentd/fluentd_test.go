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

package fluentd

import (
	"reflect"
	"testing"
)

type loc struct {
	scheme string
	host   string
	port   int
	path   string
}

func TestParseAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		want    *loc
		wantErr bool
	}{
		{name: "empty", args: args{address: ""}, want: &loc{scheme: "tcp", host: "127.0.0.1", port: 24224}, wantErr: false},
		{name: "unix", args: args{address: "unix:///var/run/fluentd/fluentd.sock"}, want: &loc{scheme: "unix", path: "/var/run/fluentd/fluentd.sock"}, wantErr: false},
		{name: "tcp", args: args{address: "tcp://127.0.0.1:24224"}, want: &loc{scheme: "tcp", host: "127.0.0.1", port: 24224}, wantErr: false},
		{name: "tcpWithPath", args: args{address: "tcp://127.0.0.1:24224/1234"}, want: nil, wantErr: true},
		{name: "unixWithEmpty", args: args{address: "unix://"}, want: nil, wantErr: true},
		{name: "invalidPath", args: args{address: "://asd123"}, want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			err := cfg.SetAddress(tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := &loc{
				scheme: cfg.FluentNetwork,
				host:   cfg.FluentHost,
				port:   cfg.FluentPort,
				path:   cfg.FluentSocketPath,
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAddress() got = %v, want %v", got, tt.want)
			}
		})
	}
}
