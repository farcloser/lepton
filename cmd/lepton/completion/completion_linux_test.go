package completion

import (
	"testing"

	"github.com/farcloser/lepton/pkg/testutil"
)

func TestCompletion(t *testing.T) {
	testutil.DockerIncompatible(t)

	base := testutil.NewBase(t)

	const gsc = "__complete"

	base.Cmd(gsc, "--cgroup-manager", "").AssertOutContains("cgroupfs\n")
	base.Cmd(gsc, "--snapshotter", "").AssertOutContains("native\n")
	base.Cmd(gsc, "").AssertOutContains("run\t")
	base.Cmd(gsc, "run", "-").AssertOutContains("--network\t")
	base.Cmd(gsc, "run", "--n").AssertOutContains("--network\t")
	base.Cmd(gsc, "run", "--ne").AssertOutContains("--network\t")
	base.Cmd(gsc, "run", "--net", "").AssertOutContains("host\n")
	base.Cmd(gsc, "run", "-it", "--net", "").AssertOutContains("host\n")
	base.Cmd(gsc, "run", "-it", "--rm", "--net", "").AssertOutContains("host\n")
	base.Cmd(gsc, "run", "--restart", "").AssertOutContains("always\n")
	base.Cmd(gsc, "network", "rm", "").AssertOutNotContains("host\n") // host is unremovable
	base.Cmd(gsc, "run", "--cap-add", "").AssertOutContains("sys_admin\n")
	base.Cmd(gsc, "run", "--cap-add", "").AssertOutNotContains("CAP_SYS_ADMIN\n") // invalid form

	// Tests with an image
	base.Cmd("pull", testutil.AlpineImage).AssertOK()
	base.Cmd(gsc, "run", "-i", "").AssertOutContains(testutil.AlpineImage)
	base.Cmd(gsc, "run", "-it", "").AssertOutContains(testutil.AlpineImage)
	base.Cmd(gsc, "run", "-it", "--rm", "").AssertOutContains(testutil.AlpineImage)

	// Tests with a network
	testNetworkName := "nerdctl-test-completion"
	defer base.Cmd("network", "rm", testNetworkName).Run()
	base.Cmd("network", "create", testNetworkName).AssertOK()
	base.Cmd(gsc, "network", "rm", "").AssertOutContains(testNetworkName)
	base.Cmd(gsc, "run", "--net", "").AssertOutContains(testNetworkName)

	// Tests with a volume
	testVolumekName := "nerdctl-test-completion"
	defer base.Cmd("volume", "rm", testVolumekName).Run()
	base.Cmd("volume", "create", testVolumekName).AssertOK()
	base.Cmd(gsc, "volume", "inspect", "").AssertOutContains(testVolumekName)
	base.Cmd(gsc, "volume", "rm", "").AssertOutContains(testVolumekName)

	// Tests with raw base (without Args={"--namespace=nerdctl-test"})
	rawBase := testutil.NewBase(t)
	rawBase.Args = nil // unset "--namespace=nerdctl-test"
	rawBase.Cmd(gsc, "--cgroup-manager", "").AssertOutContains("cgroupfs\n")
	rawBase.Cmd(gsc, "").AssertOutContains("run\t")
	// mind {"--namespace=nerdctl-test"} vs {"--namespace", "nerdctl-test"}
	rawBase.Cmd(gsc, "--namespace", testutil.Namespace, "").AssertOutContains("run\t")
	rawBase.Cmd(gsc, "--namespace", testutil.Namespace, "run", "-i", "").AssertOutContains(testutil.AlpineImage)
}
