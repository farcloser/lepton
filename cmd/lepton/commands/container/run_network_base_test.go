//go:build linux || windows

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
	"strconv"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nettestutil"
)

// Tests various port mapping argument combinations by starting a nginx container and
// verifying its connectivity and that its serves its index.html from the external
// host IP as well as through the loopback interface.
// `loopbackIsolationEnabled` indicates whether the test should expect connections between
// the loopback interface and external host interface to succeed or not.
func baseTestRunPort(
	t *testing.T,
	nginxImage string,
	nginxIndexHTMLSnippet string,
	loopbackIsolationEnabled bool,
) {
	expectedIsolationErr := ""
	if loopbackIsolationEnabled {
		expectedIsolationErr = testutil.ExpectedConnectionRefusedError
	}

	hostIP, err := nettestutil.NonLoopbackIPv4()
	assert.NilError(t, err)
	type testCase struct {
		listenIP         net.IP
		connectIP        net.IP
		hostPort         string
		containerPort    string
		connectURLPort   int
		runShouldSuccess bool
		err              string
	}
	lo := net.ParseIP("127.0.0.1")
	zeroIP := net.ParseIP("0.0.0.0")
	testCases := []testCase{
		{
			listenIP:         lo,
			connectIP:        lo,
			hostPort:         "9091",
			containerPort:    "80",
			connectURLPort:   9091,
			runShouldSuccess: true,
		},
		{
			// for https://github.com/containerd/nerdctl/issues/88
			listenIP:         hostIP,
			connectIP:        hostIP,
			hostPort:         "9092",
			containerPort:    "80",
			connectURLPort:   9092,
			runShouldSuccess: true,
		},
		{
			listenIP:         hostIP,
			connectIP:        lo,
			hostPort:         "9093",
			containerPort:    "80",
			connectURLPort:   9093,
			err:              expectedIsolationErr,
			runShouldSuccess: true,
		},
		{
			listenIP:         lo,
			connectIP:        hostIP,
			hostPort:         "9094",
			containerPort:    "80",
			connectURLPort:   9094,
			err:              expectedIsolationErr,
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        lo,
			hostPort:         "9095",
			containerPort:    "80",
			connectURLPort:   9095,
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        hostIP,
			hostPort:         "9096",
			containerPort:    "80",
			connectURLPort:   9096,
			runShouldSuccess: true,
		},
		{
			listenIP:         lo,
			connectIP:        lo,
			hostPort:         "7000-7005",
			containerPort:    "79-84",
			connectURLPort:   7001,
			runShouldSuccess: true,
		},
		{
			listenIP:         hostIP,
			connectIP:        hostIP,
			hostPort:         "7010-7015",
			containerPort:    "79-84",
			connectURLPort:   7011,
			runShouldSuccess: true,
		},
		{
			listenIP:         hostIP,
			connectIP:        lo,
			hostPort:         "7020-7025",
			containerPort:    "79-84",
			connectURLPort:   7021,
			err:              expectedIsolationErr,
			runShouldSuccess: true,
		},
		{
			listenIP:         lo,
			connectIP:        hostIP,
			hostPort:         "7030-7035",
			containerPort:    "79-84",
			connectURLPort:   7031,
			err:              expectedIsolationErr,
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        hostIP,
			hostPort:         "7040-7045",
			containerPort:    "79-84",
			connectURLPort:   7041,
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        lo,
			hostPort:         "7050-7055",
			containerPort:    "80-85",
			connectURLPort:   7051,
			err:              "error after 30 attempts",
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        lo,
			hostPort:         "7060-7065",
			containerPort:    "80",
			connectURLPort:   7060,
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        lo,
			hostPort:         "7070-7075",
			containerPort:    "80",
			connectURLPort:   7075,
			err:              testutil.ExpectedConnectionRefusedError,
			runShouldSuccess: true,
		},
		{
			listenIP:         zeroIP,
			connectIP:        lo,
			hostPort:         "7080-7085",
			containerPort:    "79-85",
			connectURLPort:   7085,
			err:              "invalid ranges specified for container and host Ports",
			runShouldSuccess: false,
		},
	}

	tID := testutil.Identifier(t)
	for i, tc := range testCases {
		tcName := fmt.Sprintf("%+v", tc)
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()

			testContainerName := fmt.Sprintf("%s-%d", tID, i)
			base := testutil.NewBase(t)
			defer base.Cmd("rm", "-f", testContainerName).Run()
			pFlag := fmt.Sprintf("%s:%s:%s", tc.listenIP.String(), tc.hostPort, tc.containerPort)
			connectURL := "http://" + net.JoinHostPort(
				tc.connectIP.String(),
				strconv.Itoa(tc.connectURLPort),
			)
			t.Logf("pFlag=%q, connectURL=%q", pFlag, connectURL)
			cmd := base.Cmd("run", "-d",
				"--name", testContainerName,
				"-p", pFlag,
				nginxImage)
			if tc.runShouldSuccess {
				cmd.AssertOK()
			} else {
				cmd.AssertFail()
				return
			}

			resp, err := nettestutil.HTTPGet(connectURL, 30, false)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
				return
			}
			defer resp.Body.Close()
			assert.NilError(t, err)
			respBody, err := io.ReadAll(resp.Body)
			assert.NilError(t, err)
			assert.Assert(t, strings.Contains(string(respBody), nginxIndexHTMLSnippet))
		})
	}
}
