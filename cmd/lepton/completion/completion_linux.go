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

package completion

import (
	"github.com/spf13/cobra"
	"go.farcloser.world/containers/security/apparmor"
	"go.farcloser.world/containers/security/cgroups"
)

func ApparmorProfiles(cmd *cobra.Command) ([]string, cobra.ShellCompDirective) {
	profiles, err := apparmor.Profiles()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var names []string //nolint:prealloc
	for _, f := range profiles {
		names = append(names, f.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func CgroupManagerNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	availableManagers := cgroups.AvailableManagers()
	candidates := make([]string, len(availableManagers))
	for i, manager := range availableManagers {
		candidates[i] = string(manager)
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp
}
