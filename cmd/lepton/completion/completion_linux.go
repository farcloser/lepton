package completion

import (
	"github.com/spf13/cobra"

	"github.com/farcloser/lepton/pkg/apparmorutil"
	ncdefaults "github.com/farcloser/lepton/pkg/defaults"
	"github.com/farcloser/lepton/pkg/rootlessutil"
)

func ApparmorProfiles(cmd *cobra.Command) ([]string, cobra.ShellCompDirective) {
	profiles, err := apparmorutil.Profiles()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string // nolint: prealloc
	for _, f := range profiles {
		names = append(names, f.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

func CgroupManagerNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	candidates := []string{"cgroupfs"}
	if ncdefaults.IsSystemdAvailable() {
		candidates = append(candidates, "systemd")
	}
	if rootlessutil.IsRootless() {
		candidates = append(candidates, "none")
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp
}
