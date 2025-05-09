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

package image_test

import (
	"testing"

	"go.farcloser.world/tigron/expect"
	"go.farcloser.world/tigron/require"
	"go.farcloser.world/tigron/test"

	"go.farcloser.world/lepton/pkg/testutil"
	"go.farcloser.world/lepton/pkg/testutil/nerdtest"
)

func TestImageConvert(t *testing.T) {
	nerdtest.Setup()

	testCase := &test.Case{
		Require: require.All(
			require.Not(nerdtest.Docker),
		),
		Setup: func(data test.Data, helpers test.Helpers) {
			helpers.Ensure("pull", "--quiet", testutil.CommonImage)
		},
		SubTests: []*test.Case{
			{
				Description: "zstd",
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rmi", "-f", data.Identifier("converted-image"))
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command(
						"image",
						"convert",
						"--oci",
						"--zstd",
						"--zstd-compression-level",
						"3",
						testutil.CommonImage,
						data.Identifier("converted-image"),
					)
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
			},
			{
				Description: "zstdchunked",
				Cleanup: func(data test.Data, helpers test.Helpers) {
					helpers.Anyhow("rmi", "-f", data.Identifier("converted-image"))
				},
				Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
					return helpers.Command(
						"image",
						"convert",
						"--oci",
						"--zstdchunked",
						"--zstdchunked-compression-level",
						"3",
						testutil.CommonImage,
						data.Identifier("converted-image"),
					)
				},
				Expected: test.Expects(expect.ExitCodeSuccess, nil, nil),
			},
		},
	}

	testCase.Run(t)
}
