/*
   Copyright The containerd Authors.

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

package main

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/containerd/containerd/v2/pkg/cap"

	"github.com/containerd/nerdctl/v2/cmd/nerdctl/completion"
)

func capShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	candidates := []string{}
	for _, c := range cap.Known() {
		// "CAP_SYS_ADMIN" -> "sys_admin"
		s := strings.ToLower(strings.TrimPrefix(c, "CAP_"))
		candidates = append(candidates, s)
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp
}

func runShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return completion.ImageNames(cmd)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}