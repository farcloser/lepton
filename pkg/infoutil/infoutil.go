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

package infoutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/protobuf/types"
	"github.com/containerd/log"

	"go.farcloser.world/containers/security/cgroups"
	"go.farcloser.world/core/version/semver"

	"go.farcloser.world/lepton/leptonic/buildkit"
	"go.farcloser.world/lepton/pkg/inspecttypes/dockercompat"
	"go.farcloser.world/lepton/pkg/inspecttypes/native"
	"go.farcloser.world/lepton/pkg/version"
)

const (
	snapshotterPluginsPrefix = "io.containerd.snapshotter."
)

var (
	ErrUnableToRetrieveHostname = errors.New("unable to retrieve hostname")
	ErrContainerdFailure        = errors.New("unable to communicate with containerd")
)

func NativeDaemonInfo(ctx context.Context, client *containerd.Client) (*native.DaemonInfo, error) {
	introService := client.IntrospectionService()

	plugins, err := introService.Plugins(ctx)
	if err != nil {
		return nil, errors.Join(ErrContainerdFailure, err)
	}

	server, err := introService.Server(ctx)
	if err != nil {
		return nil, errors.Join(ErrContainerdFailure, err)
	}

	ver, err := client.VersionService().Version(ctx, &types.Empty{})
	if err != nil {
		return nil, errors.Join(ErrContainerdFailure, err)
	}

	return &native.DaemonInfo{
		Plugins: plugins,
		Server:  server,
		Version: ver,
	}, nil
}

func Info(ctx context.Context, client *containerd.Client, snapshotter string, cgroupManager cgroups.Manager) (*dockercompat.Info, error) {
	introService := client.IntrospectionService()

	plugins, err := introService.Plugins(ctx)
	if err != nil {
		return nil, err
	}

	server, err := introService.Server(ctx)
	if err != nil {
		return nil, err
	}

	daemonVersion, err := client.Version(ctx)
	if err != nil {
		return nil, err
	}

	var snapshotterPlugins []string
	for _, p := range plugins.Plugins {
		if strings.HasPrefix(p.Type, snapshotterPluginsPrefix) && p.InitErr == nil {
			snapshotterPlugins = append(snapshotterPlugins, p.ID)
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Join(ErrUnableToRetrieveHostname, err)
	}

	var info = &dockercompat.Info{
		ID:   server.UUID,
		Name: hostname,
		// Storage drivers and logging drivers are not really Server concept here, but mimics `docker info` output
		Driver: snapshotter,
		Plugins: dockercompat.PluginsInfo{
			Storage: snapshotterPlugins,
		},
		SystemTime:      time.Now().Format(time.RFC3339Nano),
		LoggingDriver:   "json-file", // hard-coded
		CgroupDriver:    cgroupManager,
		CgroupVersion:   strconv.Itoa(int(cgroups.Version())),
		KernelVersion:   UnameR(),
		OperatingSystem: DistroName(),
		OSType:          runtime.GOOS,
		Architecture:    UnameM(),
		ServerVersion:   daemonVersion.Version,
	}

	fulfillPlatformInfo(info)

	return info, nil
}

func GetSnapshotterNames(ctx context.Context, client *containerd.Client) ([]string, error) {
	var names []string

	plugins, err := client.IntrospectionService().Plugins(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range plugins.Plugins {
		if strings.HasPrefix(p.Type, snapshotterPluginsPrefix) && p.InitErr == nil {
			names = append(names, p.ID)
		}
	}

	return names, nil
}

func ClientVersion() dockercompat.ClientVersion {
	return dockercompat.ClientVersion{
		Version:   version.GetVersion(),
		GitCommit: version.GetRevision(),
		GoVersion: runtime.Version(),
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Components: []dockercompat.ComponentVersion{
			buildctlVersion(),
		},
	}
}

func ServerVersion(ctx context.Context, client *containerd.Client) (*dockercompat.ServerVersion, error) {
	daemonVersion, err := client.Version(ctx)
	if err != nil {
		return nil, err
	}

	v := &dockercompat.ServerVersion{
		Components: []dockercompat.ComponentVersion{
			{
				Name:    "containerd",
				Version: daemonVersion.Version,
				Details: map[string]string{"GitCommit": daemonVersion.Revision},
			},
			runcVersion(),
		},
	}

	return v, nil
}

func ServerSemVer(ctx context.Context, client *containerd.Client) (*semver.Version, error) {
	v, err := client.Version(ctx)
	if err != nil {
		return nil, err
	}

	sv, err := semver.NewVersion(v.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the containerd version %q: %w", v.Version, err)
	}

	return sv, nil
}

func buildctlVersion() dockercompat.ComponentVersion {
	buildctlBinary, err := buildkit.BuildctlBinary()
	if err != nil {
		log.L.Warnf("unable to determine buildctl version: %s", err.Error())
		return dockercompat.ComponentVersion{Name: "buildctl"}
	}

	stdout, err := exec.Command(buildctlBinary, "--version").Output()
	if err != nil {
		log.L.Warnf("unable to determine buildctl version: %s", err.Error())
		return dockercompat.ComponentVersion{Name: "buildctl"}
	}

	v, err := parseBuildctlVersion(stdout)
	if err != nil {
		log.L.Warn(err)
		return dockercompat.ComponentVersion{Name: "buildctl"}
	}

	return *v
}

func parseBuildctlVersion(buildctlVersionStdout []byte) (*dockercompat.ComponentVersion, error) {
	fields := strings.Fields(strings.TrimSpace(string(buildctlVersionStdout)))
	var v *dockercompat.ComponentVersion

	switch len(fields) {
	case 4:
		v = &dockercompat.ComponentVersion{
			Name:    fields[0],
			Version: fields[2],
			Details: map[string]string{"GitCommit": fields[3]},
		}
	case 3:
		v = &dockercompat.ComponentVersion{
			Name:    fields[0],
			Version: fields[2],
		}
	default:
		return nil, fmt.Errorf("unable to determine buildctl version, got %q", string(buildctlVersionStdout))
	}
	if v.Name != "buildctl" {
		return nil, fmt.Errorf("unable to determine buildctl version, got %q", string(buildctlVersionStdout))
	}

	return v, nil
}

func runcVersion() dockercompat.ComponentVersion {
	stdout, err := exec.Command("runc", "--version").Output()
	if err != nil {
		log.L.Warnf("unable to determine runc version: %s", err.Error())
		return dockercompat.ComponentVersion{Name: "runc"}
	}

	v, err := parseRuncVersion(stdout)
	if err != nil {
		log.L.Warn(err)
		return dockercompat.ComponentVersion{Name: "runc"}
	}

	return *v
}

func parseRuncVersion(runcVersionStdout []byte) (*dockercompat.ComponentVersion, error) {
	var versionList = strings.Split(strings.TrimSpace(string(runcVersionStdout)), "\n")
	firstLine := strings.Fields(versionList[0])
	if len(firstLine) != 3 || firstLine[0] != "runc" {
		return nil, fmt.Errorf("unable to determine runc version, got: %s", string(runcVersionStdout))
	}

	version := firstLine[2]

	details := map[string]string{}
	for _, detailsLine := range versionList[1:] {
		detail := strings.SplitN(detailsLine, ":", 2)
		if len(detail) != 2 {
			log.L.Warnf("unable to determine one of runc details, got: %s, %d", detail, len(detail))
			continue
		}
		if strings.TrimSpace(detail[0]) == "commit" {
			details["GitCommit"] = strings.TrimSpace(detail[1])
		}
	}

	return &dockercompat.ComponentVersion{
		Name:    "runc",
		Version: version,
		Details: details,
	}, nil
}

// BlockIOWeight return whether Block IO weight is supported or not
func BlockIOWeight(cgroupManager cgroups.Manager) bool {
	var info dockercompat.Info
	info.CgroupVersion = strconv.Itoa(int(cgroups.Version()))
	info.CgroupDriver = cgroupManager
	mobySysInfo := mobySysInfo(&info)

	// blkio weight is not available on cgroup v1 since kernel 5.0.
	// On cgroup v2, blkio weight is implemented using io.weight
	return mobySysInfo.BlkioWeight
}
