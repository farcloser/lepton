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
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/poll"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
)

func TestRunRestart(t *testing.T) {
	// Because this will kill the daemon, parallel tests would fail
	// t.Parallel()

	const (
		hostPort = 8180
	)
	testContainerName := testutil.Identifier(t)
	if testing.Short() {
		t.Skipf("test is long")
	}
	base := testutil.NewBase(t)
	if !base.DaemonIsKillable {
		t.Skip("daemon is not killable (hint: set \"-test.allow-kill-daemon\")")
	}
	t.Log("NOTE: this test may take a while")

	tearDown := func() {
		base.Cmd("rm", "-f", testContainerName).Run()
	}

	tearDown()
	t.Cleanup(tearDown)

	base.Cmd("run", "-d",
		"--restart=always",
		"--name", testContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:80", hostPort),
		testutil.NginxAlpineImage).AssertOK()

	check := func(httpGetRetry int) error {
		resp, err := nettestutil.HTTPGet(
			"http://"+net.JoinHostPort("127.0.0.1", strconv.Itoa(hostPort)),
			httpGetRetry,
			false,
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(respBody), testutil.NginxAlpineIndexHTMLSnippet) {
			return fmt.Errorf("expected contain %q, got %q",
				testutil.NginxAlpineIndexHTMLSnippet, string(respBody))
		}
		return nil
	}
	assert.NilError(t, check(30))

	base.KillDaemon()
	base.EnsureDaemonActive()

	const (
		maxRetry = 30
		sleep    = 3 * time.Second
	)
	for i := range maxRetry {
		t.Logf("(retry %d) ps -a: %q", i, base.Cmd("ps", "-a").Run().Combined())
		err := check(1)
		if err == nil {
			t.Logf("test is passing, after %d retries", i)
			return
		}
		time.Sleep(sleep)
	}
	base.DumpDaemonLogs(10)
	t.Fatalf("the container does not seem to be restarted")
}

func TestRunRestartWithOnFailure(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	if nerdtest.IsNotDocker() {
		testutil.RequireContainerdPlugin(
			base,
			"io.containerd.internal.v1",
			"restart",
			[]string{"on-failure"},
		)
	}
	tID := testutil.Identifier(t)
	defer base.Cmd("rm", "-f", tID).Run()
	base.Cmd("run", "-d", "--restart=on-failure:2", "--name", tID, testutil.AlpineImage, "sh", "-c", "exit 1").
		AssertOK()

	check := func(log poll.LogT) poll.Result {
		inspect := base.InspectContainer(tID)
		if inspect.State != nil && inspect.State.Status == "exited" {
			return poll.Success()
		}
		return poll.Continue("container is not yet exited")
	}
	poll.WaitOn(t, check, poll.WithDelay(100*time.Microsecond), poll.WithTimeout(60*time.Second))
	inspect := base.InspectContainer(tID)
	assert.Equal(t, inspect.RestartCount, 2)
}

func TestRunRestartWithUnlessStopped(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	if nerdtest.IsNotDocker() {
		testutil.RequireContainerdPlugin(
			base,
			"io.containerd.internal.v1",
			"restart",
			[]string{"unless-stopped"},
		)
	}
	tID := testutil.Identifier(t)
	defer base.Cmd("rm", "-f", tID).Run()
	base.Cmd("run", "-d", "--restart=unless-stopped", "--name", tID, testutil.AlpineImage, "sh", "-c", "exit 1").
		AssertOK()

	check := func(log poll.LogT) poll.Result {
		inspect := base.InspectContainer(tID)
		if inspect.State != nil && inspect.State.Status == "exited" {
			return poll.Success()
		}
		if inspect.RestartCount == 2 {
			base.Cmd("stop", tID).AssertOK()
		}
		return poll.Continue("container is not yet exited")
	}
	poll.WaitOn(t, check, poll.WithDelay(100*time.Microsecond), poll.WithTimeout(60*time.Second))
	inspect := base.InspectContainer(tID)
	assert.Equal(t, inspect.RestartCount, 2)
}

func TestUpdateRestartPolicy(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	if nerdtest.IsNotDocker() {
		testutil.RequireContainerdPlugin(
			base,
			"io.containerd.internal.v1",
			"restart",
			[]string{"on-failure"},
		)
	}
	tID := testutil.Identifier(t)
	defer base.Cmd("rm", "-f", tID).Run()
	base.Cmd("run", "-d", "--restart=on-failure:1", "--name", tID, testutil.AlpineImage, "sh", "-c", "exit 1").
		AssertOK()
	base.Cmd("update", "--restart=on-failure:2", tID).AssertOK()
	check := func(log poll.LogT) poll.Result {
		inspect := base.InspectContainer(tID)
		if inspect.State != nil && inspect.State.Status == "exited" {
			return poll.Success()
		}
		return poll.Continue("container is not yet exited")
	}
	poll.WaitOn(t, check, poll.WithDelay(100*time.Microsecond), poll.WithTimeout(60*time.Second))
	inspect := base.InspectContainer(tID)
	assert.Equal(t, inspect.RestartCount, 2)
}

// The test is to add a restart policy to a container with no prior restart policy,
// and check it can work correctly.
func TestAddRestartPolicy(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	if nerdtest.IsNotDocker() {
		testutil.RequireContainerdPlugin(
			base,
			"io.containerd.internal.v1",
			"restart",
			[]string{"on-failure"},
		)
	}
	tID := testutil.Identifier(t)
	defer base.Cmd("rm", "-f", tID).Run()
	base.Cmd("run", "-d", "--name", tID, testutil.NginxAlpineImage).AssertOK()
	base.Cmd("update", "--restart=on-failure", tID).AssertOK()
	inspect := base.InspectContainer(tID)
	originalPid := inspect.State.Pid
	exec.Command("kill", "-9", strconv.Itoa(originalPid)).Run()

	check := func(log poll.LogT) poll.Result {
		inspect := base.InspectContainer(tID)
		if inspect.State != nil && inspect.State.Status == "running" &&
			inspect.State.Pid != originalPid {
			return poll.Success()
		}
		return poll.Continue("container is not yet running")
	}
	poll.WaitOn(t, check, poll.WithDelay(100*time.Microsecond), poll.WithTimeout(60*time.Second))
	inspect = base.InspectContainer(tID)
	assert.Equal(t, inspect.RestartCount, 1)
}
