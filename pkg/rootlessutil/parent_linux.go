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

package rootlessutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/containerd/log"

	"go.farcloser.world/lepton/leptonic/rootlesskit"
	"go.farcloser.world/lepton/pkg/version"
)

func IsRootlessParent() bool {
	return os.Geteuid() != 0
}

func ParentMain(hostGatewayIP string) error {
	if !IsRootlessParent() {
		return errors.New("should not be called when !IsRootlessParent()")
	}
	stateDir, err := rootlesskit.StateDir()
	log.L.Debugf("stateDir: %s", stateDir)
	if err != nil {
		return fmt.Errorf("rootless containerd not running? (hint: use `containerd-rootless-setuptool.sh install` to start rootless containerd): %w", err)
	}
	childPid, err := rootlesskit.ChildPid(stateDir)
	if err != nil {
		return err
	}

	detachedNetNSPath, err := getDetachedNetNSPath(stateDir)
	if err != nil {
		return err
	}
	detachNetNSMode := detachedNetNSPath != ""
	log.L.Debugf("RootlessKit detach-netns mode: %v", detachNetNSMode)

	// FIXME: remove dependency on `nsenter` binary
	arg0, err := exec.LookPath("nsenter")
	if err != nil {
		return err
	}
	// args are compatible with both util-linux nsenter and busybox nsenter
	args := []string{
		"-r/", // root dir (busybox nsenter wants this to be explicitly specified),
	}

	// Only append wd if we do have a working dir
	// - https://github.com/rootless-containers/usernetes/pull/327
	// - https://github.com/containerd/nerdctl/issues/3328
	wd, err := os.Getwd()
	if err != nil {
		log.L.WithError(err).Warn("unable to determine working directory")
	} else {
		args = append(args, "-w"+wd)
		os.Setenv("PWD", wd)
	}

	args = append(args, "--preserve-credentials",
		"-m", "-U",
		"-t", strconv.Itoa(childPid),
		"-F", // no fork
	)
	if !detachNetNSMode {
		args = append(args, "-n")
	}
	args = append(args, os.Args...)
	log.L.Debugf("rootless parent main: executing %q with %v", arg0, args)

	// Env vars corresponds to RootlessKit spec:
	// https://github.com/rootless-containers/rootlesskit/tree/v0.13.1#environment-variables
	os.Setenv("ROOTLESSKIT_STATE_DIR", stateDir)
	os.Setenv("ROOTLESSKIT_PARENT_EUID", strconv.Itoa(os.Geteuid()))
	os.Setenv("ROOTLESSKIT_PARENT_EGID", strconv.Itoa(os.Getegid()))
	os.Setenv(version.EnvPrefix+"_HOST_GATEWAY_IP", hostGatewayIP)
	return syscall.Exec(arg0, args, os.Environ())
}
