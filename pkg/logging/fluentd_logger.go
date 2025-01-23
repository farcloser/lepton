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

package logging

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/containerd/containerd/v2/core/runtime/v2/logging"
	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/leptonic/fluentd"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

const (
	fluentAddress                 = "fluentd-address"
	fluentdAsync                  = "fluentd-async"
	fluentdBufferLimit            = "fluentd-buffer-limit"
	fluentdRetryWait              = "fluentd-retry-wait"
	fluentdMaxRetries             = "fluentd-max-retries"
	fluentdSubSecondPrecision     = "fluentd-sub-second-precision"
	fluentdAsyncReconnectInterval = "fluentd-async-reconnect-interval"
	fluentRequestAck              = "fluentd-request-ack"
)

var FluentdLogOpts = []string{
	fluentAddress,
	fluentdAsync,
	fluentdBufferLimit,
	fluentdRetryWait,
	fluentdMaxRetries,
	fluentdSubSecondPrecision,
	fluentdAsyncReconnectInterval,
	fluentRequestAck,
	Tag,
}

type FluentdLogger struct {
	Opts         map[string]string
	fluentClient *fluentd.Logger
	id           string
	namespace    string
}

func (f *FluentdLogger) Init(dataStore, ns, id string) error {
	return nil
}

func (f *FluentdLogger) PreProcess(ctx context.Context, _ string, config *logging.Config) error {
	fluentConfig, err := parseFluentdConfig(f.Opts)
	if err != nil {
		return err
	}

	f.fluentClient = &fluentd.Logger{}
	if err = f.fluentClient.Init(ctx, fluentConfig); err != nil {
		return err
	}

	f.id = config.ID
	f.namespace = config.Namespace

	return nil
}

func (f *FluentdLogger) Process(stdout <-chan string, stderr <-chan string) error {
	metadata := map[string]string{
		"container_id": f.id,
		"namespace":    f.namespace,
	}

	return f.fluentClient.WriteLogs(f.Opts[Tag], metadata, stdout, stderr)
}

func (f *FluentdLogger) PostProcess() error {
	err := f.fluentClient.Destroy()
	f.fluentClient = nil
	return err
}

func FluentdLogOptsValidate(logOptMap map[string]string) error {
	for key := range logOptMap {
		if !strutil.InStringSlice(FluentdLogOpts, key) {
			log.L.Warnf("log-opt %s is ignored for fluentd log driver", key)
		}
	}
	if _, ok := logOptMap[fluentAddress]; !ok {
		log.L.Warnf("%s is missing for fluentd log driver, default values will be used", fluentAddress)
	}
	return nil
}

func parseFluentdConfig(config map[string]string) (*fluentd.Config, error) {
	var err error

	result := fluentd.NewConfig()
	if err := result.SetAddress(config[fluentAddress]); err != nil {
		return nil, err
	}

	if config[fluentdBufferLimit] != "" {
		result.BufferLimit, err = strconv.Atoi(config[fluentdBufferLimit])
		if err != nil {
			return nil, fmt.Errorf("error occurs %w, invalid buffer limit (%s)", err, config[fluentdBufferLimit])
		}
	}

	if config[fluentdRetryWait] != "" {
		temp, err := time.ParseDuration(config[fluentdRetryWait])
		if err != nil {
			return nil, fmt.Errorf("error occurs %w, invalid retry wait (%s)", err, config[fluentdRetryWait])
		}
		result.RetryWait = int(temp.Milliseconds())
	}

	if config[fluentdMaxRetries] != "" {
		result.MaxRetry, err = strconv.Atoi(config[fluentdMaxRetries])
		if err != nil {
			return nil, fmt.Errorf("error occurs %w, invalid max retries (%s)", err, config[fluentdMaxRetries])
		}
	}

	if config[fluentdAsync] != "" {
		result.Async, err = strconv.ParseBool(config[fluentdAsync])
		if err != nil {
			return result, fmt.Errorf("error occurs %w, invalid async (%s)", err, config[fluentdAsync])
		}
	}

	if config[fluentdAsyncReconnectInterval] != "" {
		tempDuration, err := time.ParseDuration(config[fluentdAsyncReconnectInterval])
		if err != nil {
			return nil, fmt.Errorf("error occurs %w, invalid async connect interval (%s)", err, config[fluentdAsyncReconnectInterval])
		}
		if err = result.SetAsyncReconnectInterval(int(tempDuration.Milliseconds())); err != nil {
			return nil, err
		}
	}

	if config[fluentdSubSecondPrecision] != "" {
		result.SubSecondPrecision, err = strconv.ParseBool(config[fluentdSubSecondPrecision])
		if err != nil {
			return nil, fmt.Errorf("error occurs %w, invalid sub second precision (%s)", err, config[fluentdSubSecondPrecision])
		}
	}

	if config[fluentRequestAck] != "" {
		result.RequestAck, err = strconv.ParseBool(config[fluentRequestAck])
		if err != nil {
			return nil, fmt.Errorf("error occurs %w, invalid request ack (%s)", err, config[fluentRequestAck])
		}
	}

	return result, nil
}
