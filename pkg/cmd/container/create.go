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

package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	dockercliopts "github.com/docker/cli/opts"
	"go.farcloser.world/containers/reference"
	"go.farcloser.world/containers/specs"
	"go.farcloser.world/core/utils"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/containers"
	"github.com/containerd/containerd/v2/pkg/cio"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/go-cni"
	"github.com/containerd/log"

	"github.com/containerd/nerdctl/v2/leptonic/errs"
	"github.com/containerd/nerdctl/v2/pkg/annotations"
	"github.com/containerd/nerdctl/v2/pkg/api/options"
	"github.com/containerd/nerdctl/v2/pkg/clientutil"
	"github.com/containerd/nerdctl/v2/pkg/cmd/image"
	"github.com/containerd/nerdctl/v2/pkg/cmd/volume"
	"github.com/containerd/nerdctl/v2/pkg/containerutil"
	"github.com/containerd/nerdctl/v2/pkg/dnsutil/hostsstore"
	"github.com/containerd/nerdctl/v2/pkg/flagutil"
	"github.com/containerd/nerdctl/v2/pkg/idgen"
	"github.com/containerd/nerdctl/v2/pkg/imgutil"
	"github.com/containerd/nerdctl/v2/pkg/imgutil/load"
	"github.com/containerd/nerdctl/v2/pkg/inspecttypes/dockercompat"
	"github.com/containerd/nerdctl/v2/pkg/ipcutil"
	"github.com/containerd/nerdctl/v2/pkg/labels"
	"github.com/containerd/nerdctl/v2/pkg/logging"
	"github.com/containerd/nerdctl/v2/pkg/maputil"
	"github.com/containerd/nerdctl/v2/pkg/mountutil"
	"github.com/containerd/nerdctl/v2/pkg/namestore"
	"github.com/containerd/nerdctl/v2/pkg/platformutil"
	"github.com/containerd/nerdctl/v2/pkg/rootlessutil"
	"github.com/containerd/nerdctl/v2/pkg/strutil"
)

// Create will create a container.
func Create(ctx context.Context, client *containerd.Client, args []string, netManager containerutil.NetworkOptionsManager, opts *options.ContainerCreate) (containerd.Container, func(), error) {
	// Acquire an exclusive lock on the volume store until we are done to avoid being raced by any other
	// volume operations (or any other operation involving volume manipulation)
	volStore, err := volume.Store(opts.GOptions.Namespace, opts.GOptions.DataRoot, opts.GOptions.Address)
	if err != nil {
		return nil, nil, err
	}
	err = volStore.Lock()
	if err != nil {
		return nil, nil, err
	}
	defer volStore.Release()

	// simulate the behavior of double dash
	newArg := []string{}
	if len(args) >= 2 && args[1] == "--" {
		newArg = append(newArg, args[:1]...)
		newArg = append(newArg, args[2:]...)
		args = newArg
	}
	var internalLabels internalLabels
	internalLabels.platform = opts.Platform
	internalLabels.namespace = opts.GOptions.Namespace

	var (
		id       = idgen.GenerateID()
		specOpts []oci.SpecOpts
		cOpts    []containerd.NewContainerOpts
	)

	if opts.CidFile != "" {
		if err := writeCIDFile(opts.CidFile, id); err != nil {
			return nil, nil, err
		}
	}
	dataStore, err := clientutil.DataStore(opts.GOptions.DataRoot, opts.GOptions.Address)
	if err != nil {
		return nil, nil, err
	}

	internalLabels.stateDir, err = containerutil.ContainerStateDirPath(opts.GOptions.Namespace, dataStore, id)
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(internalLabels.stateDir, 0o700); err != nil {
		return nil, nil, err
	}

	specOpts = append(specOpts,
		oci.WithDefaultSpec(),
	)

	platformOpts, err := setPlatformOptions(ctx, client, id, netManager.NetworkOptions().UTSNamespace, &internalLabels, opts)
	if err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
	}
	specOpts = append(specOpts, platformOpts...)

	if _, err := reference.Parse(args[0]); errors.Is(err, reference.ErrLoadOCIArchiveRequired) {
		imageRef := args[0]

		// Load and create the platform specified by the user.
		// If none specified, fallback to the default platform.
		platform := []string{}
		if opts.Platform != "" {
			platform = append(platform, opts.Platform)
		}

		images, err := load.FromOCIArchive(ctx, client, imageRef, options.ImageLoad{
			Stdout:       opts.Stdout,
			GOptions:     opts.GOptions,
			Platform:     platform,
			AllPlatforms: false,
			Quiet:        opts.ImagePullOpt.Quiet,
		})
		if err != nil {
			return nil, nil, err
		} else if len(images) == 0 {
			// This is a regression and should not occur.
			return nil, nil, errors.New("OCI archive did not contain any images")
		}

		image := images[0].Name
		// Multiple images loaded from the provided archive. Default to the first image found.
		if len(images) != 1 {
			log.L.Warnf("multiple images are found for the platform, defaulting to image %s...", image)
		}

		args[0] = image
	}

	var ensuredImage *imgutil.EnsuredImage
	if !opts.Rootfs {
		var platformSS []string // len: 0 or 1
		if opts.Platform != "" {
			platformSS = append(platformSS, opts.Platform)
		}
		ocispecPlatforms, err := platformutil.NewOCISpecPlatformSlice(false, platformSS)
		if err != nil {
			return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
		}
		rawRef := args[0]

		opts.ImagePullOpt.Mode = opts.Pull
		opts.ImagePullOpt.OCISpecPlatform = ocispecPlatforms
		opts.ImagePullOpt.Unpack = nil

		ensuredImage, err = image.EnsureImage(ctx, client, rawRef, opts.ImagePullOpt)
		if err != nil {
			return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
		}
	}

	rootfsOpts, rootfsCOpts, err := generateRootfsOpts(args, id, ensuredImage, opts)
	if err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
	}
	specOpts = append(specOpts, rootfsOpts...)
	cOpts = append(cOpts, rootfsCOpts...)

	if opts.Workdir != "" {
		specOpts = append(specOpts, oci.WithProcessCwd(opts.Workdir))
	}

	envs, err := flagutil.MergeEnvFileAndOSEnv(opts.EnvFile, opts.Env)
	if err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
	}

	if opts.Interactive {
		if opts.Detach {
			return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), errors.New("currently flag -i and -d cannot be specified together (FIXME)")
		}
	}

	if opts.TTY {
		specOpts = append(specOpts, oci.WithTTY)
	}

	var mountOpts []oci.SpecOpts
	mountOpts, internalLabels.anonVolumes, internalLabels.mountPoints, err = generateMountOpts(ctx, client, ensuredImage, volStore, opts)
	if err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
	}
	specOpts = append(specOpts, mountOpts...)

	// Always set internalLabels.logURI
	// to support restart the container that run with "-it", like
	//
	// 1, nerdctl run --name demo -it imagename
	// 2, ctrl + c to stop demo container
	// 3, nerdctl start/restart demo
	logConfig, err := generateLogConfig(dataStore, id, opts.LogDriver, opts.LogOpt, opts.GOptions.Namespace, opts.GOptions.Address)
	if err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
	}
	internalLabels.logURI = logConfig.LogURI

	restartOpts, err := generateRestartOpts(ctx, client, opts.Restart, logConfig.LogURI, opts.InRun)
	if err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), err
	}
	cOpts = append(cOpts, restartOpts...)

	if err = netManager.VerifyNetworkOptions(ctx); err != nil {
		return nil, generateRemoveStateDirFunc(ctx, id, internalLabels), fmt.Errorf("failed to verify networking settings: %w", err)
	}

	netOpts, netNewContainerOpts, err := netManager.ContainerNetworkingOpts(ctx, id)
	if err != nil {
		return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), fmt.Errorf("failed to generate networking spec options: %w", err)
	}
	specOpts = append(specOpts, netOpts...)
	cOpts = append(cOpts, netNewContainerOpts...)

	netLabelOpts, err := netManager.InternalNetworkingOptionLabels(ctx)
	if err != nil {
		return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), fmt.Errorf("failed to generate internal networking labels: %w", err)
	}

	envs = append(envs, "HOSTNAME="+netLabelOpts.Hostname)
	specOpts = append(specOpts, oci.WithEnv(envs))

	internalLabels.loadNetOpts(netLabelOpts)

	// NOTE: OCI hooks are currently not supported on Windows so we skip setting them altogether.
	// The OCI hooks we define (whose logic can be found in pkg/ocihook) primarily
	// perform network setup and teardown when using CNI networking.
	// On Windows, we are forced to set up and tear down the networking from within nerdctl.
	if runtime.GOOS != "windows" {
		hookOpt, err := withOCIHook(opts.CliCmd, opts.CliArgs)
		if err != nil {
			return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
		}
		specOpts = append(specOpts, hookOpt)
	}

	uOpts := generateUserOpts(opts.User)
	specOpts = append(specOpts, uOpts...)
	gOpts := generateGroupsOpts(opts.GroupAdd)
	specOpts = append(specOpts, gOpts...)

	umaskOpts, err := generateUmaskOpts(opts.Umask)
	if err != nil {
		return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
	}
	specOpts = append(specOpts, umaskOpts...)

	rtCOpts := generateRuntimeCOpts(opts.GOptions.CgroupManager, opts.Runtime)
	cOpts = append(cOpts, rtCOpts...)

	lCOpts, err := withContainerLabels(opts.Label, opts.LabelFile, ensuredImage)
	if err != nil {
		return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
	}
	cOpts = append(cOpts, lCOpts...)

	var containerNameStore namestore.NameStore
	if opts.Name == "" && !opts.NameChanged {
		// Automatically set the container name, unless `--name=""` was explicitly specified.
		var imageRef string
		if ensuredImage != nil {
			imageRef = ensuredImage.Ref
		}
		parsedReference, err := reference.Parse(imageRef)
		// Ignore cases where the imageRef is ""
		if err != nil && imageRef != "" {
			return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
		}
		opts.Name = parsedReference.SuggestContainerName(id)
	}
	if opts.Name != "" {
		containerNameStore, err = namestore.New(dataStore, opts.GOptions.Namespace)
		if err != nil {
			return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
		}
		if err := containerNameStore.Acquire(opts.Name, id); err != nil {
			return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
		}
	}
	internalLabels.name = opts.Name
	internalLabels.pidFile = opts.PidFile

	extraHosts, err := containerutil.ParseExtraHosts(netManager.NetworkOptions().AddHost, opts.GOptions.HostGatewayIP, ":")
	if err != nil {
		return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
	}
	internalLabels.extraHosts = extraHosts

	internalLabels.rm = containerutil.EncodeContainerRmOptLabel(opts.Rm)

	// TODO: abolish internal labels and only use annotations
	ilOpt, err := withInternalLabels(internalLabels)
	if err != nil {
		return nil, generateRemoveOrphanedDirsFunc(ctx, id, dataStore, internalLabels), err
	}
	cOpts = append(cOpts, ilOpt)

	specOpts = append(specOpts, propagateInternalContainerdLabelsToOCIAnnotations(),
		oci.WithAnnotations(utils.KeyValueStringsToMap(opts.Annotations)))

	var s specs.Spec
	spec := containerd.WithSpec(&s, specOpts...)

	cOpts = append(cOpts, spec)

	c, containerErr := client.NewContainer(ctx, id, cOpts...)
	var netSetupErr error
	if containerErr == nil {
		netSetupErr = netManager.SetupNetworking(ctx, id)
		if netSetupErr != nil {
			log.G(ctx).WithError(netSetupErr).Warnf("networking setup error has occurred")
		}
	}

	if containerErr != nil || netSetupErr != nil {
		returnedError := containerErr
		if netSetupErr != nil {
			returnedError = netSetupErr // mutually exclusive
		}
		return nil, generateGcFunc(ctx, c, opts.GOptions.Namespace, id, opts.Name, dataStore, containerErr, containerNameStore, netManager, internalLabels), returnedError
	}

	return c, nil, nil
}

func generateRootfsOpts(args []string, id string, ensured *imgutil.EnsuredImage, options *options.ContainerCreate) (opts []oci.SpecOpts, cOpts []containerd.NewContainerOpts, err error) {
	if !options.Rootfs {
		cOpts = append(cOpts,
			containerd.WithImage(ensured.Image),
			containerd.WithSnapshotter(ensured.Snapshotter),
			containerd.WithNewSnapshot(id, ensured.Image),
			containerd.WithImageStopSignal(ensured.Image, "SIGTERM"),
		)

		if len(ensured.ImageConfig.Env) == 0 {
			opts = append(opts, oci.WithDefaultPathEnv)
		}
		for ind, env := range ensured.ImageConfig.Env {
			if strings.HasPrefix(env, "PATH=") {
				break
			}
			if ind == len(ensured.ImageConfig.Env)-1 {
				opts = append(opts, oci.WithDefaultPathEnv)
			}
		}
	} else {
		absRootfs, err := filepath.Abs(args[0])
		if err != nil {
			return nil, nil, err
		}
		opts = append(opts, oci.WithRootFSPath(absRootfs), oci.WithDefaultPathEnv)
	}

	entrypointPath := ""
	if ensured != nil {
		if len(ensured.ImageConfig.Entrypoint) > 0 {
			entrypointPath = ensured.ImageConfig.Entrypoint[0]
		} else if len(ensured.ImageConfig.Cmd) > 0 {
			entrypointPath = ensured.ImageConfig.Cmd[0]
		}
	}

	if !options.Rootfs && !options.EntrypointChanged {
		opts = append(opts, oci.WithImageConfigArgs(ensured.Image, args[1:]))
	} else {
		if !options.Rootfs {
			opts = append(opts, oci.WithImageConfig(ensured.Image))
		}
		var processArgs []string
		if len(options.Entrypoint) != 0 {
			processArgs = append(processArgs, options.Entrypoint...)
		}
		if len(args) > 1 {
			processArgs = append(processArgs, args[1:]...)
		}
		if len(processArgs) == 0 {
			// error message is from Podman
			return nil, nil, errors.New("no command or entrypoint provided, and no CMD or ENTRYPOINT from image")
		}

		entrypointPath = processArgs[0]

		opts = append(opts, oci.WithProcessArgs(processArgs...))
	}

	isEntryPointSystemd := entrypointPath == "/sbin/init" ||
		entrypointPath == "/usr/sbin/init" ||
		entrypointPath == "/usr/local/sbin/init"

	stopSignal := options.StopSignal

	if options.Systemd == "always" || (options.Systemd == "true" && isEntryPointSystemd) {
		if options.Privileged {
			securityOptsMap := utils.KeyValueStringsToMap(strutil.DedupeStrSlice(options.SecurityOpt))
			privilegedWithoutHostDevices, err := maputil.MapBoolValueAsOpt(securityOptsMap, "privileged-without-host-devices")
			if err != nil {
				return nil, nil, err
			}

			// See: https://github.com/containers/podman/issues/15878
			if !privilegedWithoutHostDevices {
				return nil, nil, errors.New("if --privileged is used with systemd `--security-opt privileged-without-host-devices` must also be used")
			}
		}

		opts = append(opts,
			oci.WithoutMounts("/sys/fs/cgroup"),
			oci.WithMounts([]specs.Mount{
				{Type: "cgroup", Source: "cgroup", Destination: "/sys/fs/cgroup", Options: []string{"rw"}},
				{Type: "tmpfs", Source: "tmpfs", Destination: "/run"},
				{Type: "tmpfs", Source: "tmpfs", Destination: "/run/lock"},
				{Type: "tmpfs", Source: "tmpfs", Destination: "/tmp"},
				{Type: "tmpfs", Source: "tmpfs", Destination: "/var/lib/journal"},
			}),
		)
		stopSignal = "SIGRTMIN+3"
	}

	cOpts = append(cOpts, withStop(stopSignal, options.StopTimeout, ensured))

	if options.InitBinary != nil {
		options.InitProcessFlag = true
	}
	if options.InitProcessFlag {
		binaryPath, err := exec.LookPath(*options.InitBinary)
		if err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return nil, nil, fmt.Errorf(`init binary %q not found`, *options.InitBinary)
			}
			return nil, nil, err
		}
		inContainerPath := filepath.Join("/sbin", filepath.Base(*options.InitBinary))
		opts = append(opts, func(_ context.Context, _ oci.Client, _ *containers.Container, spec *oci.Spec) error {
			spec.Process.Args = append([]string{inContainerPath, "--"}, spec.Process.Args...)
			spec.Mounts = append([]specs.Mount{{
				Destination: inContainerPath,
				Type:        "bind",
				Source:      binaryPath,
				Options:     []string{"bind", "ro"},
			}}, spec.Mounts...)
			return nil
		})
	}
	if options.ReadOnly {
		opts = append(opts, oci.WithRootFSReadonly())
	}
	return opts, cOpts, nil
}

// GenerateLogURI generates a log URI for the current container store
func GenerateLogURI(dataStore string) (*url.URL, error) {
	selfExe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	args := map[string]string{
		logging.MagicArgv1: dataStore,
	}

	return cio.LogURIGenerator("binary", selfExe, args)
}

func withOCIHook(cmd string, args []string) (oci.SpecOpts, error) {
	if rootlessutil.IsRootless() {
		detachedNetNS, err := rootlessutil.DetachedNetNS()
		if err != nil {
			return nil, fmt.Errorf("failed to check whether RootlessKit is running with --detach-netns: %w", err)
		}
		if detachedNetNS != "" {
			// Rewrite {cmd, args} if RootlessKit is running with --detach-netns, so that the hook can gain
			// CAP_NET_ADMIN in the namespaces.
			//   - Old:
			//     - cmd:  "/usr/local/bin/nerdctl"
			//     - args: {"--data-root=/foo", "internal", "oci-hook"}
			//   - New:
			//     - cmd:  "/usr/bin/nsenter"
			//     - args: {"-n/run/user/1000/containerd-rootless/netns", "-F", "--", "/usr/local/bin/nerdctl", "--data-root=/foo", "internal", "oci-hook"}
			oldCmd, oldArgs := cmd, args
			cmd, err = exec.LookPath("nsenter")
			if err != nil {
				return nil, err
			}
			args = append([]string{"-n" + detachedNetNS, "-F", "--", oldCmd}, oldArgs...)
		}
	}

	args = append([]string{cmd}, append(args, "internal", "oci-hook")...)
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *specs.Spec) error {
		if s.Hooks == nil {
			s.Hooks = &specs.Hooks{}
		}
		crArgs := append(args, "createRuntime")
		s.Hooks.CreateRuntime = append(s.Hooks.CreateRuntime, specs.Hook{
			Path: cmd,
			Args: crArgs,
			Env:  os.Environ(),
		})
		argsCopy := append([]string(nil), args...)
		psArgs := append(argsCopy, "postStop")
		s.Hooks.Poststop = append(s.Hooks.Poststop, specs.Hook{
			Path: cmd,
			Args: psArgs,
			Env:  os.Environ(),
		})
		return nil
	}, nil
}

func withContainerLabels(label, labelFile []string, ensuredImage *imgutil.EnsuredImage) ([]containerd.NewContainerOpts, error) {
	var opts []containerd.NewContainerOpts

	// add labels defined by image
	if ensuredImage != nil {
		imageLabelOpts := containerd.WithAdditionalContainerLabels(ensuredImage.ImageConfig.Labels)
		opts = append(opts, imageLabelOpts)
	}

	labelMap, err := readKVStringsMapfFromLabel(label, labelFile)
	if err != nil {
		return nil, err
	}
	for k := range labelMap {
		if strings.HasPrefix(k, annotations.Bypass4netns) {
			log.L.Warnf("Label %q is deprecated, use an annotation instead", k)
		} else if strings.HasPrefix(k, labels.Prefix) {
			return nil, fmt.Errorf("internal label %q must not be specified manually", k)
		}
	}
	o := containerd.WithAdditionalContainerLabels(labelMap)
	opts = append(opts, o)

	return opts, nil
}

func readKVStringsMapfFromLabel(label, labelFile []string) (map[string]string, error) {
	labelsMap := strutil.DedupeStrSlice(label)
	labelsFilePath := strutil.DedupeStrSlice(labelFile)
	kvStrings, err := dockercliopts.ReadKVStrings(labelsFilePath, labelsMap)
	if err != nil {
		return nil, err
	}
	return utils.KeyValueStringsToMap(kvStrings), nil
}

// parseKVStringsMapFromLogOpt parse log options KV entries and convert to Map
func parseKVStringsMapFromLogOpt(logOpt []string, logDriver string) (map[string]string, error) {
	logOptArray := strutil.DedupeStrSlice(logOpt)
	logOptMap := utils.KeyValueStringsToMap(logOptArray)
	if logDriver == "json-file" {
		if _, ok := logOptMap[logging.MaxSize]; !ok {
			delete(logOptMap, logging.MaxFile)
		}
	}
	if err := logging.ValidateLogOpts(logDriver, logOptMap); err != nil {
		return nil, err
	}
	return logOptMap, nil
}

func withStop(stopSignal string, stopTimeout int, ensuredImage *imgutil.EnsuredImage) containerd.NewContainerOpts {
	return func(ctx context.Context, _ *containerd.Client, c *containers.Container) error {
		if c.Labels == nil {
			c.Labels = make(map[string]string)
		}
		var err error
		if ensuredImage != nil {
			stopSignal, err = containerd.GetOCIStopSignal(ctx, ensuredImage.Image, stopSignal)
			if err != nil {
				return err
			}
		}
		c.Labels[containerd.StopSignalLabel] = stopSignal
		if stopTimeout != 0 {
			c.Labels[labels.StopTimeout] = strconv.Itoa(stopTimeout)
		}
		return nil
	}
}

type internalLabels struct {
	// labels from cmd options
	namespace  string
	platform   string
	extraHosts []string
	pidFile    string
	// labels from cmd options or automatically set
	name     string
	hostname string
	// automatically generated
	stateDir string
	// network
	networks   []string
	ipAddress  string
	ip6Address string
	ports      []cni.PortMapping
	macAddress string
	// volume
	mountPoints []*mountutil.Processed
	anonVolumes []string
	// pid namespace
	pidContainer string
	// ipc namespace & dev/shm
	ipc string
	// log
	logURI string
	// a label to check whether the --rm option is specified.
	rm string
}

// WithInternalLabels sets the internal labels for a container.
func withInternalLabels(internalLabels internalLabels) (containerd.NewContainerOpts, error) {
	m := make(map[string]string)
	m[labels.Namespace] = internalLabels.namespace
	if internalLabels.name != "" {
		m[labels.Name] = internalLabels.name
	}
	m[labels.Hostname] = internalLabels.hostname
	extraHostsJSON, err := json.Marshal(internalLabels.extraHosts)
	if err != nil {
		return nil, err
	}
	m[labels.ExtraHosts] = string(extraHostsJSON)
	m[labels.StateDir] = internalLabels.stateDir
	networksJSON, err := json.Marshal(internalLabels.networks)
	if err != nil {
		return nil, err
	}
	m[labels.Networks] = string(networksJSON)
	if len(internalLabels.ports) > 0 {
		portsJSON, err := json.Marshal(internalLabels.ports)
		if err != nil {
			return nil, err
		}
		m[labels.Ports] = string(portsJSON)
	}
	if internalLabels.logURI != "" {
		m[labels.LogURI] = internalLabels.logURI
	}
	if len(internalLabels.anonVolumes) > 0 {
		anonVolumeJSON, err := json.Marshal(internalLabels.anonVolumes)
		if err != nil {
			return nil, err
		}
		m[labels.AnonymousVolumes] = string(anonVolumeJSON)
	}

	if internalLabels.pidFile != "" {
		m[labels.PIDFile] = internalLabels.pidFile
	}

	if internalLabels.ipAddress != "" {
		m[labels.IPAddress] = internalLabels.ipAddress
	}

	if internalLabels.ip6Address != "" {
		m[labels.IP6Address] = internalLabels.ip6Address
	}

	m[labels.Platform], err = platformutil.NormalizeString(internalLabels.platform)
	if err != nil {
		return nil, err
	}

	if len(internalLabels.mountPoints) > 0 {
		mounts := dockercompatMounts(internalLabels.mountPoints)
		mountPointsJSON, err := json.Marshal(mounts)
		if err != nil {
			return nil, err
		}
		m[labels.Mounts] = string(mountPointsJSON)
	}

	if internalLabels.macAddress != "" {
		m[labels.MACAddress] = internalLabels.macAddress
	}

	if internalLabels.pidContainer != "" {
		m[labels.PIDContainer] = internalLabels.pidContainer
	}

	if internalLabels.ipc != "" {
		m[labels.IPC] = internalLabels.ipc
	}

	if internalLabels.rm != "" {
		m[labels.ContainerAutoRemove] = internalLabels.rm
	}

	return containerd.WithAdditionalContainerLabels(m), nil
}

// loadNetOpts loads network options into InternalLabels.
func (il *internalLabels) loadNetOpts(opts options.ContainerNetwork) {
	il.hostname = opts.Hostname
	il.ports = opts.PortMappings
	il.ipAddress = opts.IPAddress
	il.ip6Address = opts.IP6Address
	il.networks = opts.NetworkSlice
	il.macAddress = opts.MACAddress
}

func dockercompatMounts(mountPoints []*mountutil.Processed) []dockercompat.MountPoint {
	result := make([]dockercompat.MountPoint, len(mountPoints))
	for i := range mountPoints {
		mp := mountPoints[i]
		result[i] = dockercompat.MountPoint{
			Type:        mp.Type,
			Name:        mp.Name,
			Source:      mp.Mount.Source,
			Destination: mp.Mount.Destination,
			Driver:      "",
			Mode:        mp.Mode,
		}
		result[i].RW, result[i].Propagation = dockercompat.ParseMountProperties(strings.Split(mp.Mode, ","))

		// it's an anonymous volume
		if mp.AnonymousVolume != "" {
			result[i].Name = mp.AnonymousVolume
		}

		// volume only support local driver
		if mp.Type == "volume" {
			result[i].Driver = "local"
		}
	}
	return result
}

func processeds(mountPoints []dockercompat.MountPoint) []*mountutil.Processed {
	result := make([]*mountutil.Processed, len(mountPoints))
	for i := range mountPoints {
		mp := mountPoints[i]
		result[i] = &mountutil.Processed{
			Type: mp.Type,
			Name: mp.Name,
			Mount: specs.Mount{
				Source:      mp.Source,
				Destination: mp.Destination,
			},
			Mode: mp.Mode,
		}
	}
	return result
}

func propagateInternalContainerdLabelsToOCIAnnotations() oci.SpecOpts {
	return func(ctx context.Context, oc oci.Client, c *containers.Container, s *oci.Spec) error {
		allowed := make(map[string]string)
		for k, v := range c.Labels {
			if strings.Contains(k, labels.Prefix) {
				allowed[k] = v
			}
		}
		return oci.WithAnnotations(allowed)(ctx, oc, c, s)
	}
}

func writeCIDFile(path, id string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("container ID file found, make sure the other container isn't running or delete %s", path)
	} else if errors.Is(err, os.ErrNotExist) {
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create the container ID file: %s for %s, err: %w", path, id, err)
		}
		defer f.Close()

		if _, err := f.WriteString(id); err != nil {
			return err
		}
		return nil
	}
	return err
}

// generateLogConfig creates a LogConfig for the current container store
func generateLogConfig(dataStore string, id string, logDriver string, logOpt []string, ns, address string) (logConfig logging.LogConfig, err error) {
	var u *url.URL
	if u, err = url.Parse(logDriver); err == nil && u.Scheme != "" {
		logConfig.LogURI = logDriver
	} else {
		logConfig.Driver = logDriver
		logConfig.Address = address
		logConfig.Opts, err = parseKVStringsMapFromLogOpt(logOpt, logDriver)
		if err != nil {
			return logConfig, err
		}
		var (
			logDriverInst logging.Driver
			logConfigB    []byte
			lu            *url.URL
		)
		logDriverInst, err = logging.GetDriver(logDriver, logConfig.Opts, logConfig.Address)
		if err != nil {
			return logConfig, err
		}
		if err = logDriverInst.Init(dataStore, ns, id); err != nil {
			return logConfig, err
		}

		logConfigB, err = json.Marshal(logConfig)
		if err != nil {
			return logConfig, err
		}

		logConfigFilePath := logging.LogConfigFilePath(dataStore, ns, id)
		if err = os.WriteFile(logConfigFilePath, logConfigB, 0o600); err != nil {
			return logConfig, err
		}

		lu, err = GenerateLogURI(dataStore)
		if err != nil {
			return logConfig, err
		}
		if lu != nil {
			log.L.Debugf("generated log driver: %s", lu.String())
			logConfig.LogURI = lu.String()
		}
	}
	return logConfig, nil
}

func generateRemoveStateDirFunc(ctx context.Context, id string, internalLabels internalLabels) func() {
	return func() {
		if rmErr := os.RemoveAll(internalLabels.stateDir); rmErr != nil {
			log.G(ctx).WithError(rmErr).Warnf("failed to remove container %q state dir %q", id, internalLabels.stateDir)
		}
	}
}

func generateRemoveOrphanedDirsFunc(ctx context.Context, id, dataStore string, internalLabels internalLabels) func() {
	return func() {
		if rmErr := os.RemoveAll(internalLabels.stateDir); rmErr != nil {
			log.G(ctx).WithError(rmErr).Warnf("failed to remove container %q state dir %q", id, internalLabels.stateDir)
		}

		hs, err := hostsstore.New(dataStore, internalLabels.namespace)
		if err != nil {
			log.G(ctx).WithError(err).Warnf("failed to instantiate hostsstore for %q", internalLabels.namespace)
		} else if err = hs.DeallocHostsFile(id); err != nil {
			log.G(ctx).WithError(err).Warnf("failed to remove an etchosts directory for container %q", id)
		}
	}
}

func generateGcFunc(ctx context.Context, container containerd.Container, ns, id, name, dataStore string, containerErr error, containerNameStore namestore.NameStore, netManager containerutil.NetworkOptionsManager, internalLabels internalLabels) func() {
	return func() {
		if containerErr == nil {
			netGcErr := netManager.CleanupNetworking(ctx, container)
			if netGcErr != nil {
				log.G(ctx).WithError(netGcErr).Warnf("failed to revert container %q networking settings", id)
			}
		} else {
			hs, err := hostsstore.New(dataStore, internalLabels.namespace)
			if err != nil {
				log.G(ctx).WithError(err).Warnf("failed to instantiate hostsstore for %q", internalLabels.namespace)
			} else {
				if _, err := hs.HostsPath(id); err != nil {
					log.G(ctx).WithError(err).Warnf("an etchosts directory for container %q dosen't exist", id)
				} else if err = hs.DeallocHostsFile(id); err != nil {
					log.G(ctx).WithError(err).Warnf("failed to remove an etchosts directory for container %q", id)
				}
			}
		}

		ipc, ipcErr := ipcutil.DecodeIPCLabel(internalLabels.ipc)
		if ipcErr != nil {
			log.G(ctx).WithError(ipcErr).Warnf("failed to decode ipc label for container %q", id)
		}
		if ipcErr := ipcutil.CleanUp(ipc); ipcErr != nil {
			log.G(ctx).WithError(ipcErr).Warnf("failed to clean up ipc for container %q", id)
		}
		if rmErr := os.RemoveAll(internalLabels.stateDir); rmErr != nil {
			log.G(ctx).WithError(rmErr).Warnf("failed to remove container %q state dir %q", id, internalLabels.stateDir)
		}

		if name != "" {
			var errE error
			if containerNameStore, errE = namestore.New(dataStore, ns); errE != nil {
				log.G(ctx).WithError(errE).Warnf("failed to instantiate container name store during cleanup for container %q", id)
			}
			// Double-releasing may happen with containers started with --rm, so, ignore NotFound errors
			if errE := containerNameStore.Release(name, id); errE != nil && !errors.Is(errE, errs.ErrNotFound) {
				log.G(ctx).WithError(errE).Warnf("failed to release container name store for container %q (%s)", name, id)
			}
		}
	}
}
