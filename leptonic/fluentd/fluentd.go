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
	"context"
	"errors"
	"sync"

	"github.com/fluent/fluent-logger-golang/fluent"
)

var (
	ErrFailedCreatingClient   = errors.New("failed to create fluent client")
	ErrFailedDestroyingClient = errors.New("failed to destroy fluent client")
)

type Logger struct {
	client *fluent.Fluent
}

func (f *Logger) Init(_ context.Context, config *Config) error {
	var err error

	f.client, err = fluent.New(config.Config)
	if err != nil {
		return errors.Join(ErrFailedCreatingClient, err)
	}

	return nil
}

func (f *Logger) WriteLogs(tag string, metadata map[string]string, stdout <-chan string, stderr <-chan string) error {
	var wg sync.WaitGroup
	wg.Add(2)

	fun := func(wg *sync.WaitGroup, dataChan <-chan string, source string) {
		metadata["source"] = source
		for log := range dataChan {
			metadata["log"] = log
			_ = f.client.Post(tag, metadata)
		}

		wg.Done()
	}

	go fun(&wg, stdout, "stdout")
	go fun(&wg, stderr, "stderr")

	wg.Wait()

	return nil
}

func (f *Logger) Destroy() error {
	if err := f.client.Close(); err != nil {
		return errors.Join(ErrFailedDestroyingClient, err)
	}

	return nil
}
