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

package fluentd

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
)

const (
	defaultHost        = "127.0.0.1"
	defaultPort        = 24224
	defaultProtocol    = "tcp"
	defaultPath        = ""
	defaultBufferLimit = 1024 * 1024
	defaultMaxRetries  = math.MaxInt32
)

var (
	defaultRetryWait     = int((1000 * time.Millisecond).Milliseconds())
	minReconnectInterval = int((100 * time.Millisecond).Milliseconds())
	maxReconnectInterval = int((10 * time.Second).Milliseconds())

	ErrInvalidArgument            = errors.New("invalid argument")
	ErrUnixSocketPathMustExist    = errors.New("unix socket path must not be empty")
	ErrUnsupportedProtocol        = errors.New("unsupported protocol")
	ErrUnsupportedPathForProtocol = errors.New("path is not supported for this protocol")
)

type Config struct {
	fluent.Config
}

func NewConfig() *Config {
	return &Config{
		Config: fluent.Config{
			FluentPort:             defaultPort,
			FluentHost:             defaultHost,
			FluentNetwork:          defaultProtocol,
			FluentSocketPath:       defaultPath,
			BufferLimit:            defaultBufferLimit,
			RetryWait:              defaultRetryWait,
			MaxRetry:               defaultMaxRetries,
			Async:                  false,
			AsyncReconnectInterval: 0,
			SubSecondPrecision:     false,
			RequestAck:             false,
		},
	}
}

func (cfg *Config) SetAsyncReconnectInterval(asyncReconnectInterval int) error {
	// Enforce limits on reconnect interval
	if asyncReconnectInterval != 0 && (asyncReconnectInterval < minReconnectInterval || asyncReconnectInterval > maxReconnectInterval) {
		return fmt.Errorf("%w: asyncReconnectInterval (%d) must be between %d and %d milliseconds", ErrInvalidArgument, asyncReconnectInterval, minReconnectInterval, maxReconnectInterval)
	}

	cfg.AsyncReconnectInterval = asyncReconnectInterval
	return nil
}

func (cfg *Config) SetAddress(address string) error {
	if address == "" {
		return nil
	}

	if !strings.Contains(address, "://") {
		address = defaultProtocol + "://" + address
	}

	tempURL, err := url.Parse(address)
	if err != nil {
		return err
	}

	cfg.FluentNetwork = tempURL.Scheme

	switch tempURL.Scheme {
	case "unix":
		if strings.TrimLeft(tempURL.Path, string(os.PathSeparator)) == "" {
			return ErrUnixSocketPathMustExist
		}
		cfg.FluentHost = ""
		cfg.FluentPort = 0
		cfg.FluentSocketPath = tempURL.Path
		return nil
	case "tcp", "tls":
	// continue to process below
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedProtocol, tempURL.Scheme)
	}

	if tempURL.Path != "" {
		return fmt.Errorf("%w: %s", ErrUnsupportedPathForProtocol, tempURL.Path)
	}

	if h := tempURL.Hostname(); h != "" {
		cfg.FluentHost = h
	}

	if p := tempURL.Port(); p != "" {
		portNum, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return errors.Join(fmt.Errorf("%w: invalid port %q", ErrInvalidArgument, p), err)
		}
		cfg.FluentPort = int(portNum)
	}

	return nil
}
