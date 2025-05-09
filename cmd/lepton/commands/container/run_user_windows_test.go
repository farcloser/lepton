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
	"testing"

	"go.farcloser.world/lepton/pkg/testutil"
)

func TestRunUserName(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	testCases := map[string]string{
		"":                       "ContainerAdministrator",
		"ContainerAdministrator": "ContainerAdministrator",
		"ContainerUser":          "ContainerUser",
	}
	for userStr, expected := range testCases {
		t.Run(userStr, func(t *testing.T) {
			t.Parallel()

			cmd := []string{"run", "--rm"}
			if userStr != "" {
				cmd = append(cmd, "--user", userStr)
			}
			cmd = append(cmd, testutil.WindowsNano, "whoami")
			base.Cmd(cmd...).AssertOutContains(expected)
		})
	}
}
