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

package container_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/containerd/continuity/testutil/loopback"
	"gotest.tools/v3/assert"

	"go.farcloser.world/containers/security/cgroups"
	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/cmd/container"
	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestRunCgroupV2(t *testing.T) {
	t.Parallel()

	if cgroups.Version() != cgroups.Version2 {
		t.Skip("test requires cgroup v2")
	}
	base := testutil.NewBase(t)
	info := base.Info()
	switch info.CgroupDriver {
	case cgroups.NoneManager, "":
		t.Skip("test requires cgroup driver")
	}

	if !info.MemoryLimit {
		t.Skip("test requires MemoryLimit")
	}
	if !info.SwapLimit {
		t.Skip("test requires SwapLimit")
	}
	if !info.CPUShares {
		t.Skip("test requires CPUShares")
	}
	if !info.CPUSet {
		t.Skip("test requires CPUSet")
	}
	if !info.PidsLimit {
		t.Skip("test requires PidsLimit")
	}
	const expected1 = `42000 100000
44040192
44040192
42
77
0-1
0
`
	const expected2 = `42000 100000
44040192
60817408
6291456
42
77
0-1
0
`

	// In CgroupV2 CPUWeight replace CPUShares => weight := 1 + ((shares-2)*9999)/262142
	base.Cmd("run", "--rm",
		"--cpus", "0.42", "--cpuset-mems", "0",
		"--memory", "42m",
		"--pids-limit", "42",
		"--cpu-shares", "2000", "--cpuset-cpus", "0-1",
		"-w", "/sys/fs/cgroup", testutil.AlpineImage,
		"cat", "cpu.max", "memory.max", "memory.swap.max",
		"pids.max", "cpu.weight", "cpuset.cpus", "cpuset.mems").AssertOutExactly(expected1)
	base.Cmd("run", "--rm",
		"--cpu-quota", "42000", "--cpuset-mems", "0",
		"--cpu-period", "100000", "--memory", "42m", "--memory-reservation", "6m", "--memory-swap", "100m",
		"--pids-limit", "42", "--cpu-shares", "2000", "--cpuset-cpus", "0-1",
		"-w", "/sys/fs/cgroup", testutil.AlpineImage,
		"cat", "cpu.max", "memory.max", "memory.swap.max", "memory.low", "pids.max",
		"cpu.weight", "cpuset.cpus", "cpuset.mems").AssertOutExactly(expected2)

	base.Cmd("run", "--name", testutil.Identifier(t)+"-testUpdate1", "-w", "/sys/fs/cgroup", "-d",
		testutil.AlpineImage, "sleep", nerdtest.Infinity).AssertOK()
	defer base.Cmd("rm", "-f", testutil.Identifier(t)+"-testUpdate1").Run()
	update := []string{"update", "--cpu-quota", "42000", "--cpuset-mems", "0", "--cpu-period", "100000",
		"--memory", "42m",
		"--pids-limit", "42", "--cpu-shares", "2000", "--cpuset-cpus", "0-1"}
	if nerdtest.IsDocker() && info.CgroupVersion == strconv.Itoa(int(cgroups.Version2)) && info.SwapLimit {
		// Workaround for Docker with cgroup v2:
		// > Error response from daemon: Cannot update container 67c13276a13dd6a091cdfdebb355aa4e1ecb15fbf39c2b5c9abee89053e88fce:
		// > Memory limit should be smaller than already set memoryswap limit, update the memoryswap at the same time
		update = append(update, "--memory-swap=84m")
	}
	update = append(update, testutil.Identifier(t)+"-testUpdate1")
	base.Cmd(update...).AssertOK()
	base.Cmd("exec", testutil.Identifier(t)+"-testUpdate1",
		"cat", "cpu.max", "memory.max", "memory.swap.max",
		"pids.max", "cpu.weight", "cpuset.cpus", "cpuset.mems").AssertOutExactly(expected1)

	defer base.Cmd("rm", "-f", testutil.Identifier(t)+"-testUpdate2").Run()
	base.Cmd("run", "--name", testutil.Identifier(t)+"-testUpdate2", "-w", "/sys/fs/cgroup", "-d",
		testutil.AlpineImage, "sleep", nerdtest.Infinity).AssertOK()
	base.EnsureContainerStarted(testutil.Identifier(t) + "-testUpdate2")

	base.Cmd("update", "--cpu-quota", "42000", "--cpuset-mems", "0", "--cpu-period", "100000",
		"--memory", "42m", "--memory-reservation", "6m", "--memory-swap", "100m",
		"--pids-limit", "42", "--cpu-shares", "2000", "--cpuset-cpus", "0-1",
		testutil.Identifier(t)+"-testUpdate2").AssertOK()
	base.Cmd("exec", testutil.Identifier(t)+"-testUpdate2",
		"cat", "cpu.max", "memory.max", "memory.swap.max", "memory.low",
		"pids.max", "cpu.weight", "cpuset.cpus", "cpuset.mems").AssertOutExactly(expected2)

}

func TestRunDevice(t *testing.T) {
	testCase := nerdtest.Setup()

	testCase.Require = nerdtest.Rootful

	const n = 3
	lo := make([]*loopback.Loopback, n)

	testCase.Setup = func(data test.Data, helpers test.Helpers) {

		for i := range n {
			var err error
			lo[i], err = loopback.New(4096)
			assert.NilError(t, err)
			t.Logf("lo[%d] = %+v", i, lo[i])
			loContent := fmt.Sprintf("lo%d-content", i)
			assert.NilError(t, os.WriteFile(lo[i].Device, []byte(loContent), 0o700))
			data.Set("loContent"+strconv.Itoa(i), loContent)
		}

		// lo0 is readable but not writable.
		// lo1 is readable and writable
		// lo2 is not accessible.
		helpers.Ensure("run",
			"-d",
			"--name", data.Identifier(),
			"--device", lo[0].Device+":r",
			"--device", lo[1].Device,
			testutil.AlpineImage, "sleep", nerdtest.Infinity)
		data.Set("id", data.Identifier())
	}

	testCase.Cleanup = func(data test.Data, helpers test.Helpers) {
		for i := range n {
			if lo[i] != nil {
				_ = lo[i].Close()
			}
		}
		helpers.Anyhow("rm", "-f", data.Identifier())
	}

	testCase.SubTests = []*test.Case{
		{
			Description: "can read lo0",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("exec", data.Get("id"), "cat", lo[0].Device)
			},
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Contains(data.Get("locontent0")),
				}
			},
		},
		{
			Description: "cannot write lo0",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("exec", data.Get("id"), "sh", "-ec", "echo -n \"overwritten-lo1-content\">"+lo[0].Device)
			},
			Expected: test.Expects(expect.ExitCodeGenericFail, nil, nil),
		},
		{
			Description: "cannot read lo2",
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("exec", data.Get("id"), "cat", lo[2].Device)
			},
			Expected: test.Expects(expect.ExitCodeGenericFail, nil, nil),
		},
		{
			Description: "can read lo1",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("exec", data.Get("id"), "cat", lo[1].Device)
			},
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.Contains(data.Get("locontent1")),
				}
			},
		},
		{
			Description: "can write lo1 and read back updated value",
			NoParallel:  true,
			Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
				return helpers.Command("exec", data.Get("id"), "sh", "-ec", "echo -n \"overwritten-lo1-content\">"+lo[1].Device)
			},
			Expected: test.Expects(expect.ExitCodeSuccess, nil, func(stdout string, info string, t *testing.T) {
				lo1Read, err := os.ReadFile(lo[1].Device)
				assert.NilError(t, err)
				assert.Equal(t, string(bytes.Trim(lo1Read, "\x00")), "overwritten-lo1-content")
			}),
		},
	}

	testCase.Run(t)
}

func TestParseDevice(t *testing.T) {
	t.Parallel()

	type testCase struct {
		s                     string
		expectedDevPath       string
		expectedContainerPath string
		expectedMode          string
		err                   string
	}
	testCases := []testCase{
		{
			s:                     "/dev/sda1",
			expectedDevPath:       "/dev/sda1",
			expectedContainerPath: "/dev/sda1",
			expectedMode:          "rwm",
		},
		{
			s:                     "/dev/sda2:r",
			expectedDevPath:       "/dev/sda2",
			expectedContainerPath: "/dev/sda2",
			expectedMode:          "r",
		},
		{
			s:                     "/dev/sda3:rw",
			expectedDevPath:       "/dev/sda3",
			expectedContainerPath: "/dev/sda3",
			expectedMode:          "rw",
		},
		{
			s:   "sda4",
			err: "not an absolute path",
		},
		{
			s:                     "/dev/sda5:/dev/sda5",
			expectedDevPath:       "/dev/sda5",
			expectedContainerPath: "/dev/sda5",
			expectedMode:          "rwm",
		},
		{
			s:                     "/dev/sda6:/dev/foo6",
			expectedDevPath:       "/dev/sda6",
			expectedContainerPath: "/dev/foo6",
			expectedMode:          "rwm",
		},
		{
			s:   "/dev/sda7:/dev/sda7:rwmx",
			err: "unexpected rune",
		},
	}

	for _, tc := range testCases {
		t.Log(tc.s)
		devPath, containerPath, mode, err := container.ParseDevice(tc.s)
		if tc.err == "" {
			assert.NilError(t, err)
			assert.Equal(t, tc.expectedDevPath, devPath)
			assert.Equal(t, tc.expectedContainerPath, containerPath)
			assert.Equal(t, tc.expectedMode, mode)
		} else {
			assert.ErrorContains(t, err, tc.err)
		}
	}
}

func TestRunCgroupConf(t *testing.T) {
	t.Parallel()

	if cgroups.Version() != cgroups.Version2 {
		t.Skip("test requires cgroup v2")
	}
	testutil.DockerIncompatible(t) // Docker lacks --cgroup-conf
	base := testutil.NewBase(t)
	info := base.Info()
	switch info.CgroupDriver {
	case cgroups.NoneManager, "":
		t.Skip("test requires cgroup driver")
	}
	if !info.MemoryLimit {
		t.Skip("test requires MemoryLimit")
	}
	base.Cmd("run", "--rm", "--cgroup-conf", "memory.high=33554432", "-w", "/sys/fs/cgroup", testutil.AlpineImage,
		"cat", "memory.high").AssertOutExactly("33554432\n")
}

func TestRunCgroupParent(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	info := base.Info()
	switch info.CgroupDriver {
	case cgroups.NoneManager, "":
		t.Skip("test requires cgroup driver")
	}

	containerName := testutil.Identifier(t)
	t.Logf("Using %q cgroup driver", info.CgroupDriver)

	parent := "/foobarbaz"
	if info.CgroupDriver == cgroups.SystemdManager {
		// Path separators aren't allowed in systemd path. runc
		// explicitly checks for this.
		// https://github.com/opencontainers/runc/blob/016a0d29d1750180b2a619fc70d6fe0d80111be0/libcontainer/cgroups/systemd/common.go#L65-L68
		parent = "foobarbaz.slice"
	}

	tearDown := func() {
		base.Cmd("rm", "-f", containerName).Run()
	}

	tearDown()
	t.Cleanup(tearDown)

	// cgroup2 without host cgroup ns will just output 0::/ which doesn't help much to verify
	// we got our expected path. This approach should work for both cgroup1 and 2, there will
	// just be many more entries for cgroup1 as there'll be an entry per controller.
	base.Cmd(
		"run",
		"-d",
		"--name",
		containerName,
		"--cgroupns=host",
		"--cgroup-parent", parent,
		testutil.AlpineImage,
		"sleep",
		"infinity",
	).AssertOK()

	id := base.InspectContainer(containerName).ID
	expected := filepath.Join(parent, id)
	if info.CgroupDriver == cgroups.SystemdManager {
		expected = filepath.Join(parent, testutil.GetTarget()+"-"+id)
		if nerdtest.IsDocker() {
			expected = filepath.Join(parent, "docker-"+id)
		}
	}
	base.Cmd("exec", containerName, "cat", "/proc/self/cgroup").AssertOutContains(expected)
}

func TestRunBlkioWeightCgroupV2(t *testing.T) {
	t.Parallel()

	if cgroups.Version() != cgroups.Version2 {
		t.Skip("test requires cgroup v2")
	}
	if _, err := os.Stat("/sys/module/bfq"); err != nil {
		t.Skipf("test requires \"bfq\" module to be loaded: %v", err)
	}
	base := testutil.NewBase(t)
	info := base.Info()
	switch info.CgroupDriver {
	case cgroups.NoneManager, "":
		t.Skip("test requires cgroup driver")
	}
	containerName := testutil.Identifier(t)
	defer base.Cmd("rm", "-f", containerName).AssertOK()
	// when bfq io scheduler is used, the io.weight knob is exposed as io.bfq.weight
	base.Cmd("run", "--name", containerName, "--blkio-weight", "300", "-w", "/sys/fs/cgroup", testutil.AlpineImage, "sleep", nerdtest.Infinity).AssertOK()
	base.Cmd("exec", containerName, "cat", "io.bfq.weight").AssertOutExactly("default 300\n")
	base.Cmd("update", containerName, "--blkio-weight", "400").AssertOK()
	base.Cmd("exec", containerName, "cat", "io.bfq.weight").AssertOutExactly("default 400\n")
}
