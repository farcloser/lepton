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

package volume_test

import (
	"strings"
	"testing"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestVolumePrune(t *testing.T) {
	setup := func(data test.Data, helpers test.Helpers) {
		anonIDBusy := strings.TrimSpace(helpers.Capture("volume", "create"))
		anonIDDangling := strings.TrimSpace(helpers.Capture("volume", "create"))

		namedBusy := data.Identifier("busy")
		namedDangling := data.Identifier("free")

		helpers.Ensure("volume", "create", namedBusy)
		helpers.Ensure("volume", "create", namedDangling)
		helpers.Ensure("run", "--quiet", "--name", data.Identifier(),
			"-v", namedBusy+":/namedbusyvolume",
			"-v", anonIDBusy+":/anonbusyvolume", testutil.CommonImage)

		data.Set("anonIDBusy", anonIDBusy)
		data.Set("anonIDDangling", anonIDDangling)
		data.Set("namedBusy", namedBusy)
		data.Set("namedDangling", namedDangling)
	}

	cleanup := func(data test.Data, helpers test.Helpers) {
		helpers.Anyhow("rm", "-f", data.Identifier())
		helpers.Anyhow("volume", "rm", "-f", data.Get("anonIDBusy"))
		helpers.Anyhow("volume", "rm", "-f", data.Get("anonIDDangling"))
		helpers.Anyhow("volume", "rm", "-f", data.Get("namedBusy"))
		helpers.Anyhow("volume", "rm", "-f", data.Get("namedDangling"))
	}

	testCase := nerdtest.Setup()
	// This set must be marked as private, since we cannot prune without interacting with other tests.
	testCase.Require = nerdtest.Private
	// Furthermore, these two subtests cannot be run in parallel
	testCase.SubTests = []*test.Case{
		{
			Description: "prune anonymous only",
			NoParallel:  true,
			Setup:       setup,
			Cleanup:     cleanup,
			Command:     test.Command("volume", "prune", "-f"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.All(
						expect.DoesNotContain(data.Get("anonIDBusy")),
						expect.Contains(data.Get("anonIDDangling")),
						expect.DoesNotContain(data.Get("namedBusy")),
						expect.DoesNotContain(data.Get("namedDangling")),
						func(stdout, info string, t *testing.T) {
							helpers.Ensure("volume", "inspect", data.Get("anonIDBusy"))
							helpers.Fail("volume", "inspect", data.Get("anonIDDangling"))
							helpers.Ensure("volume", "inspect", data.Get("namedBusy"))
							helpers.Ensure("volume", "inspect", data.Get("namedDangling"))
						},
					),
				}
			},
		},
		{
			Description: "prune all",
			NoParallel:  true,
			Setup:       setup,
			Cleanup:     cleanup,
			Command:     test.Command("volume", "prune", "-f", "--all"),
			Expected: func(data test.Data, helpers test.Helpers) *test.Expected {
				return &test.Expected{
					Output: expect.All(
						expect.DoesNotContain(data.Get("anonIDBusy")),
						expect.Contains(data.Get("anonIDDangling")),
						expect.DoesNotContain(data.Get("namedBusy")),
						expect.Contains(data.Get("namedDangling")),
						func(stdout, info string, t *testing.T) {
							helpers.Ensure("volume", "inspect", data.Get("anonIDBusy"))
							helpers.Fail("volume", "inspect", data.Get("anonIDDangling"))
							helpers.Ensure("volume", "inspect", data.Get("namedBusy"))
							helpers.Fail("volume", "inspect", data.Get("namedDangling"))
						},
					),
				}
			},
		},
	}

	testCase.Run(t)
}
