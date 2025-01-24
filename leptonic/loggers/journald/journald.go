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

package journald

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/coreos/go-systemd/v22/journal"

	"github.com/containerd/nerdctl/v2/leptonic/errs"
	timetypes "github.com/containerd/nerdctl/v2/leptonic/time"
)

const (
	journalctlBinary = "journalctl"
)

var (
	ErrNotAvailable         = errors.New("the local systemd journal is not available for logging")
	ErrCannotFindJournalctl = errors.New("could not find `journalctl` executable in PATH")
	ErrFailedToStart        = errors.New("failed to start journalctl command with args")

	clientPath     string
	errClientPath  error
	clientPathOnce sync.Once
)

type LogViewOptions struct {
	// Identifier to use with journalctl
	ID string

	// Whether to follow the output of the container logs.
	Follow bool

	// Start/end timestamps to filter logs by.
	Since string
	Until string

	NoTail bool
}

func Init() error {
	if !journal.Enabled() {
		return ErrNotAvailable
	}

	return nil
}

func Destroy() error {
	return nil
}

func WriteLogs(metadata map[string]string, stdout <-chan string, stderr <-chan string) error {
	var wg sync.WaitGroup
	wg.Add(2)

	fun := func(wg *sync.WaitGroup, dataChan <-chan string, pri journal.Priority, vars map[string]string) {
		for log := range dataChan {
			_ = journal.Send(log, pri, vars)
		}
		wg.Done()
	}

	go fun(&wg, stdout, journal.PriInfo, metadata)
	go fun(&wg, stderr, journal.PriErr, metadata)

	wg.Wait()

	return nil
}

// ReadLogs formats command line arguments for `journalctl` with the provided log viewing options and
// exec's and redirects `journalctl`s outputs to the provided io.Writers.
func ReadLogs(lvopts LogViewOptions, stdout, stderr io.Writer, stopChannel chan os.Signal) error {
	clientPathOnce.Do(func() {
		clientPath, errClientPath = exec.LookPath(journalctlBinary)
	})

	if errClientPath != nil {
		return errors.Join(ErrCannotFindJournalctl, errClientPath)
	}

	shortID := lvopts.ID[:12]
	var journalctlArgs = []string{"SYSLOG_IDENTIFIER=" + shortID, "--output=cat"}
	if lvopts.Follow {
		journalctlArgs = append(journalctlArgs, "-f")
	}

	if lvopts.Since != "" {
		// using GetTimestamp from moby to keep time format consistency
		ts, err := timetypes.GetTimestamp(lvopts.Since, time.Now())
		if err != nil {
			return errors.Join(fmt.Errorf("%w for \"since\"", errs.ErrInvalidArgument), err)
		}
		date, err := prepareJournalCtlDate(ts)
		if err != nil {
			return err
		}
		journalctlArgs = append(journalctlArgs, "--since", date)
	}

	if lvopts.Until != "" {
		// using GetTimestamp from moby to keep time format consistency
		ts, err := timetypes.GetTimestamp(lvopts.Until, time.Now())
		if err != nil {
			return errors.Join(fmt.Errorf("%w for \"until\"", errs.ErrInvalidArgument), err)
		}

		date, err := prepareJournalCtlDate(ts)
		if err != nil {
			return err
		}
		journalctlArgs = append(journalctlArgs, "--until", date)
	}

	cmd := exec.Command(clientPath, journalctlArgs...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return errors.Join(fmt.Errorf("%w: %#v", ErrFailedToStart, journalctlArgs), err)
	}

	// Setup killing goroutine:
	killed := false
	go func() {
		<-stopChannel
		killed = true
		_ = cmd.Process.Kill()
	}()

	err := cmd.Wait()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		if !killed && exitError.ExitCode() != 0 {
			return errors.New("journalctl command exited with non-zero exit code")
		}
	}

	return nil
}

func prepareJournalCtlDate(t string) (string, error) {
	i, err := strconv.ParseInt(t, 10, 64)
	if err != nil {
		return "", err
	}
	tm := time.Unix(i, 0)
	s := tm.Format("2006-01-02 15:04:05")
	return s, nil
}
