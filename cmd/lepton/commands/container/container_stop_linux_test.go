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

package container

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/go-iptables/iptables"
	"gotest.tools/v3/assert"

	"go.farcloser.world/lepton/pkg/rootlessutil"
	"go.farcloser.world/lepton/pkg/testutil"
	iptablesutil "go.farcloser.world/lepton/pkg/testutil/iptables"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
)

func TestStopStart(t *testing.T) {
	const (
		hostPort = 8080
	)
	testContainerName := testutil.Identifier(t)
	base := testutil.NewBase(t)
	defer base.Cmd("rm", "-f", testContainerName).Run()

	base.Cmd("run", "-d",
		"--restart=no",
		"--name", testContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:80", hostPort),
		testutil.NginxAlpineImage).AssertOK()

	check := func(httpGetRetry int) error {
		resp, err := nettestutil.HTTPGet("http://"+net.JoinHostPort("127.0.0.1",
			strconv.Itoa(hostPort)), httpGetRetry, false)
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
	base.Cmd("stop", testContainerName).AssertOK()
	base.Cmd("exec", testContainerName, "ps").AssertFail()
	if check(1) == nil {
		t.Fatal("expected to get an error")
	}
	base.Cmd("start", testContainerName).AssertOK()
	assert.NilError(t, check(30))
}

func TestStopWithStopSignal(t *testing.T) {
	t.Parallel()
	// There may be issues with logs in Docker.
	// This test is flaky with Docker. Might be related to https://github.com/containerd/nerdctl/pull/3557
	base := testutil.NewBase(t)
	testContainerName := testutil.Identifier(t)
	defer base.Cmd("rm", "-f", testContainerName).Run()

	base.Cmd("run", "-d", "--stop-signal", "SIGQUIT", "--name", testContainerName, testutil.CommonImage, "sh", "-euxc", `#!/bin/sh
set -eu
trap 'quit=1' QUIT
quit=0
while [ $quit -ne 1 ]; do
    printf 'wait quit'
    sleep 1
done
echo "signal quit"`).AssertOK()
	base.Cmd("stop", testContainerName).AssertOK()
	base.Cmd("logs", "-f", testContainerName).AssertOutContains("signal quit")
}

func TestStopCleanupForwards(t *testing.T) {
	const (
		hostPort          = 9999
		testContainerName = "ngx"
	)
	base := testutil.NewBase(t)
	defer func() {
		base.Cmd("rm", "-f", testContainerName).Run()
	}()

	// skip if rootless
	if rootlessutil.IsRootless() {
		t.Skip("pkg/testutil/iptables does not support rootless")
	}

	ipt, err := iptables.New()
	assert.NilError(t, err)

	containerID := base.Cmd("run", "-d",
		"--restart=no",
		"--name", testContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:80", hostPort),
		testutil.NginxAlpineImage).Run().Stdout()
	containerID = strings.TrimSuffix(containerID, "\n")

	containerIP := base.Cmd("inspect",
		"-f",
		"'{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}'",
		testContainerName).Run().Stdout()
	containerIP = strings.ReplaceAll(containerIP, "'", "")
	containerIP = strings.TrimSuffix(containerIP, "\n")

	// define iptables chain name depending on the target (docker/nerdctl)
	var chain string
	if nerdtest.IsDocker() {
		chain = "DOCKER"
	} else {
		redirectChain := "CNI-HOSTPORT-DNAT"
		chain = iptablesutil.GetRedirectedChain(t, ipt, redirectChain, testutil.Namespace, containerID)
	}
	assert.Equal(t, iptablesutil.ForwardExists(t, ipt, chain, containerIP, hostPort), true)

	base.Cmd("stop", testContainerName).AssertOK()
	assert.Equal(t, iptablesutil.ForwardExists(t, ipt, chain, containerIP, hostPort), false)
}

// Regression test for https://github.com/containerd/nerdctl/issues/3353
func TestStopCreated(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	tID := testutil.Identifier(t)

	tearDown := func() {
		base.Cmd("rm", "-f", tID).Run()
	}

	setup := func() {
		base.Cmd("create", "--name", tID, testutil.CommonImage).AssertOK()
	}

	t.Cleanup(tearDown)
	tearDown()
	setup()

	base.Cmd("stop", tID).AssertOK()
}
