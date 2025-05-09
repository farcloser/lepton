//go:build unix

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

/*
   Portions from:
   - https://github.com/moby/moby/blob/v20.10.6/api/types/container/container_top.go
   - https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go
   Copyright (C) The Moby authors.
   Licensed under the Apache License, Version 2.0
   NOTICE: https://github.com/moby/moby/blob/v20.10.6/NOTICE
*/

package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	containerd "github.com/containerd/containerd/v2/client"
)

// containerTop was inspired from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L133-L189
//
// ContainerTop lists the processes running inside the given
// container by calling ps with the given args, or with the flags
// "-ef" if no args are given.  An error is returned if the container
// is not found, or is not running, or if there are any problems
// running ps, or parsing the output.
// procList *ContainerTopOKBody
func containerTop(ctx context.Context, stdio io.Writer, client *containerd.Client, id, psArgs string) error {
	if psArgs == "" {
		psArgs = "-ef"
	}

	if err := validatePSArgs(psArgs); err != nil {
		return err
	}

	container, err := client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return err
	}

	status, err := task.Status(ctx)
	if err != nil {
		return err
	}

	if status.Status != containerd.Running {
		return nil
	}

	// TODO handle restarting case: wait for container to restart and then launch top command

	procs, err := task.Pids(ctx)
	if err != nil {
		return err
	}

	psList := make([]uint32, 0, len(procs))
	for _, ps := range procs {
		psList = append(psList, ps.Pid)
	}

	args := strings.Split(psArgs, " ")
	pids := psPidsArg(psList)
	output, err := exec.Command("ps", append(args, pids)...).Output()
	if err != nil {
		// some ps options (such as f) can't be used together with q,
		// so retry without it
		output, err = exec.Command("ps", args...).Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok { //nolint:errorlint
				// first line of stderr shows why ps failed
				line := bytes.SplitN(ee.Stderr, []byte{'\n'}, 2)
				if len(line) > 0 && len(line[0]) > 0 {
					return errors.New(string(line[0]))
				}
			}
			return nil
		}
	}
	procList, err := parsePSOutput(output, psList)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdio, 20, 1, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(procList.Titles, "\t"))

	for _, proc := range procList.Processes {
		fmt.Fprintln(w, strings.Join(proc, "\t"))
	}

	return w.Flush()
}

// appendProcess2ProcList is from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L49-L55
func appendProcess2ProcList(procList *ContainerTopOKBody, fields []string) {
	// Make sure number of fields equals number of header titles
	// merging "overhanging" fields
	process := fields[:len(procList.Titles)-1]
	process = append(process, strings.Join(fields[len(procList.Titles)-1:], " "))
	procList.Processes = append(procList.Processes, process)
}

// psPidsArg is from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L119-L131
//
// psPidsArg converts a slice of PIDs to a string consisting
// of comma-separated list of PIDs prepended by "-q".
// For example, psPidsArg([]uint32{1,2,3}) returns "-q1,2,3".
func psPidsArg(pids []uint32) string {
	b := []byte{'-', 'q'}
	for i, p := range pids {
		b = strconv.AppendUint(b, uint64(p), 10)
		if i < len(pids)-1 {
			b = append(b, ',')
		}
	}
	return string(b)
}

// validatePSArgs is from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L19-L35
func validatePSArgs(psArgs string) error {
	// NOTE: \\s does not detect Unicode whitespaces.
	// So we use fieldsASCII instead of strings.Fields in parsePSOutput.
	// See https://github.com/docker/docker/pull/24358
	re := regexp.MustCompile(`\s+(\S*)=\s*(PID\S*)`)
	for _, group := range re.FindAllStringSubmatch(psArgs, -1) {
		if len(group) >= 3 {
			k := group[1]
			v := group[2]
			if k != "pid" {
				return fmt.Errorf("specifying \"%s=%s\" is not allowed", k, v)
			}
		}
	}
	return nil
}

// fieldsASCII is from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L37-L47
//
// fieldsASCII is similar to strings.Fields but only allows ASCII whitespaces
func fieldsASCII(s string) []string {
	fn := func(r rune) bool {
		switch r {
		case '\t', '\n', '\f', '\r', ' ':
			return true
		}
		return false
	}
	return strings.FieldsFunc(s, fn)
}

// hasPid is from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L57-L64
func hasPid(procs []uint32, pid int) bool {
	for _, p := range procs {
		if int(p) == pid {
			return true
		}
	}
	return false
}

// parsePSOutput is from https://github.com/moby/moby/blob/v20.10.6/daemon/top_unix.go#L66-L117
func parsePSOutput(output []byte, procs []uint32) (*ContainerTopOKBody, error) {
	procList := &ContainerTopOKBody{}

	lines := strings.Split(string(output), "\n")
	procList.Titles = fieldsASCII(lines[0])

	pidIndex := -1
	for i, name := range procList.Titles {
		if name == "PID" {
			pidIndex = i
			break
		}
	}
	if pidIndex == -1 {
		return nil, errors.New("couldn't find PID field in ps output")
	}

	// loop through the output and extract the PID from each line
	// fixing #30580, be able to display thread line also when "m" option used
	// in "docker top" client command
	preContainedPidFlag := false
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		fields := fieldsASCII(line)

		var (
			p   int
			err error
		)

		if fields[pidIndex] == "-" {
			if preContainedPidFlag {
				appendProcess2ProcList(procList, fields)
			}
			continue
		}
		p, err = strconv.Atoi(fields[pidIndex])
		if err != nil {
			return nil, fmt.Errorf("unexpected pid '%s': %w", fields[pidIndex], err)
		}

		if hasPid(procs, p) {
			preContainedPidFlag = true
			appendProcess2ProcList(procList, fields)
			continue
		}
		preContainedPidFlag = false
	}
	return procList, nil
}
