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

package logging

import (
	"reflect"
	"testing"

	"github.com/containerd/nerdctl/v2/leptonic/fluentd"
)

func TestParseFluentdConfig(t *testing.T) {
	type args struct {
		config map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    *fluentd.Config
		wantErr bool
	}{
		{"DefaultLocation", args{
			config: map[string]string{}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				return cfg
			})(),
			false,
		},
		{"InputLocation", args{
			config: map[string]string{
				fluentAddress: "tcp://127.0.0.1:123",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.FluentPort = 123
				cfg.FluentHost = "127.0.0.1"
				return cfg
			})(), false},
		{"InvalidLocation", args{config: map[string]string{fluentAddress: "://asd123"}}, nil, true},
		{"InputAsyncOption", args{
			config: map[string]string{
				fluentdAsync: "true",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.Async = true
				return cfg
			})(), false},
		{"InputAsyncReconnectOption", args{
			config: map[string]string{
				fluentdAsyncReconnectInterval: "100ms",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.AsyncReconnectInterval = 100
				return cfg
			})(), false},
		{"InputBufferLimitOption", args{
			config: map[string]string{
				fluentdBufferLimit: "1000",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.BufferLimit = 1000
				return cfg
			})(), false},
		{"InputRetryWaitOption", args{
			config: map[string]string{
				fluentdRetryWait: "10s",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.RetryWait = 10000
				return cfg
			})(), false},
		{"InputMaxRetriesOption", args{
			config: map[string]string{
				fluentdMaxRetries: "100",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.MaxRetry = 100
				return cfg
			})(), false},
		{"InputSubSecondPrecision", args{
			config: map[string]string{
				fluentdSubSecondPrecision: "true",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.SubSecondPrecision = true
				return cfg
			})(), false},
		{"InputRequestAck", args{
			config: map[string]string{
				fluentRequestAck: "true",
			}},
			(func() *fluentd.Config {
				cfg := fluentd.NewConfig()
				cfg.RequestAck = true
				return cfg
			})(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFluentdConfig(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFluentdConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFluentdConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
