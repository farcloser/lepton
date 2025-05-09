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

package taskutil

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/containerd/console"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/log"
	"golang.org/x/term"

	"go.farcloser.world/core/version/semver"

	cio2 "go.farcloser.world/lepton/leptonic/cio"
	"go.farcloser.world/lepton/pkg/consoleutil"
	"go.farcloser.world/lepton/pkg/infoutil"
)

// NewTask is from https://github.com/containerd/containerd/blob/v1.4.3/cmd/ctr/commands/tasks/tasks_unix.go#L70-L108
func NewTask(
	ctx context.Context,
	client *containerd.Client,
	container containerd.Container,
	attachStreamOpt []string,
	flagI, flagT, flagD bool,
	con console.Console,
	logURI, detachKeys, namespace string,
	detachC chan<- struct{},
) (containerd.Task, error) {
	var t containerd.Task
	closer := func() {
		if detachC != nil {
			detachC <- struct{}{}
		}
		// t will be set by container.NewTask at the end of this function.
		//
		// We cannot use container.Task(ctx, cio.Load) to get the IO here
		// because the `cancel` field of the returned `*cio` is nil. [1]
		//
		// [1]
		// https://github.com/containerd/containerd/blob/8f756bc8c26465bd93e78d9cd42082b66f276e10/cio/io.go#L358-L359
		io := t.IO()
		if io == nil {
			log.G(ctx).Errorf("got a nil io")
			return
		}
		io.Cancel()
	}
	var ioCreator cio.Creator
	if len(attachStreamOpt) != 0 {
		log.G(ctx).Debug("attaching output instead of using the log-uri")
		if flagT {
			in, err := consoleutil.NewDetachableStdin(con, detachKeys, closer)
			if err != nil {
				return nil, err
			}
			ioCreator = cio.NewCreator(cio.WithStreams(in, con, nil), cio.WithTerminal)
		} else {
			streams := processAttachStreamsOpt(attachStreamOpt)
			ioCreator = cio.NewCreator(cio.WithStreams(streams.stdIn, streams.stdOut, streams.stdErr))
		}

	} else if flagT && flagD {
		u, err := url.Parse(logURI)
		if err != nil {
			return nil, err
		}

		var args []string
		for k, vs := range u.Query() {
			args = append(args, k)
			if len(vs) > 0 {
				args = append(args, vs[0])
			}
		}

		// args[0]: _PREFIX_INTERNAL_LOGGING
		// args[1]: /var/lib/<ROOT_NAME>/1935db59
		if len(args) != 2 {
			return nil, errors.New("parse logging path error")
		}
		parsedPath := u.Path
		// For Windows, remove the leading slash
		if (runtime.GOOS == "windows") && (strings.HasPrefix(parsedPath, "/")) {
			parsedPath = strings.TrimLeft(parsedPath, "/")
		}
		ioCreator = cio.TerminalBinaryIO(parsedPath, map[string]string{
			args[0]: args[1],
		})
	} else if flagT && !flagD {
		if con == nil {
			return nil, errors.New("got nil con with flagT=true")
		}
		var in io.Reader
		if flagI {
			// FIXME: check IsTerminal on Windows too
			if runtime.GOOS != "windows" && !term.IsTerminal(0) {
				return nil, errors.New("the input device is not a TTY")
			}
			var err error
			in, err = consoleutil.NewDetachableStdin(con, detachKeys, closer)
			if err != nil {
				return nil, err
			}
		}
		ioCreator = cio2.NewContainerIO(namespace, logURI, true, in, os.Stdout, os.Stderr)
	} else if flagD && logURI != "" {
		u, err := url.Parse(logURI)
		if err != nil {
			return nil, err
		}
		ioCreator = cio.LogURI(u)
	} else {
		var in io.Reader
		if flagI {
			v2, _ := semver.NewVersion("2.0.0-0")
			if sv, err := infoutil.ServerSemVer(ctx, client); err != nil {
				log.G(ctx).Warn(err)
			} else if sv.LessThan(v2) {
				log.G(ctx).Warnf("`(run|exec) -i` without `-t` expects containerd 2.0 or later, got containerd %v", sv)
			}
			var stdinC io.ReadCloser = &StdinCloser{
				Stdin: os.Stdin,
				Closer: func() {
					if t, err := container.Task(ctx, nil); err != nil {
						log.G(ctx).WithError(err).Debugf("failed to get task for StdinCloser")
					} else {
						t.CloseIO(ctx, containerd.WithStdinCloser)
					}
				},
			}
			in = stdinC
		}
		ioCreator = cio2.NewContainerIO(namespace, logURI, false, in, os.Stdout, os.Stderr)
	}
	t, err := container.NewTask(ctx, ioCreator)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// struct used to store streams specified with attachStreamOpt (-a, --attach)
type streams struct {
	stdIn  *os.File
	stdOut *os.File
	stdErr *os.File
}

func nullStream() *os.File {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil
	}
	defer devNull.Close()

	return devNull
}

func processAttachStreamsOpt(streamsArr []string) streams {
	stdIn := os.Stdin
	stdOut := os.Stdout
	stdErr := os.Stderr

	for i, str := range streamsArr {
		streamsArr[i] = strings.ToUpper(str)
	}

	if !slices.Contains(streamsArr, "STDIN") {
		stdIn = nullStream()
	}

	if !slices.Contains(streamsArr, "STDOUT") {
		stdOut = nullStream()
	}

	if !slices.Contains(streamsArr, "STDERR") {
		stdErr = nullStream()
	}

	return streams{
		stdIn:  stdIn,
		stdOut: stdOut,
		stdErr: stdErr,
	}
}

// StdinCloser is from https://github.com/containerd/containerd/blob/v1.4.3/cmd/ctr/commands/tasks/exec.go#L181-L194
type StdinCloser struct {
	mu     sync.Mutex
	Stdin  *os.File
	Closer func()
	closed bool
}

func (s *StdinCloser) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, syscall.EBADF
	}
	n, err := s.Stdin.Read(p)
	if err != nil {
		if s.Closer != nil {
			s.Closer()
			s.closed = true
		}
	}
	return n, err
}

// Close implements Closer
func (s *StdinCloser) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if s.Closer != nil {
		s.Closer()
	}
	s.closed = true
	return nil
}
