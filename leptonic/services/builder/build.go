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

package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/archive"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/platforms"

	"go.farcloser.world/containers/reference"
	"go.farcloser.world/containers/specs"

	"go.farcloser.world/lepton/leptonic/buildkit"
	"go.farcloser.world/lepton/pkg/api/options"
	"go.farcloser.world/lepton/pkg/clientutil"
	"go.farcloser.world/lepton/pkg/containerutil"
	"go.farcloser.world/lepton/pkg/platformutil"
	"go.farcloser.world/lepton/pkg/strutil"
)

type PlatformParser interface {
	Parse(platform string) (platforms.Platform, error)
	DefaultSpec() platforms.Platform
}

type platformParser struct{}

func (p platformParser) Parse(platform string) (platforms.Platform, error) {
	return platforms.Parse(platform)
}

func (p platformParser) DefaultSpec() platforms.Platform {
	return platforms.DefaultSpec()
}

func Build(ctx context.Context, cli *client.Client, globalOptions *options.Global, opts *options.BuilderBuild) error {
	buildctlBinary, err := buildkit.BuildctlBinary()
	if err != nil {
		return err
	}

	buildctlArgs, needsLoading, metaFile, tags, cleanup, err := generateBuildctlArgs(ctx, cli, globalOptions, opts)
	if err != nil {
		return err
	}

	if cleanup != nil {
		defer cleanup()
	}

	buildctlCmd := exec.Command(buildctlBinary, buildctlArgs...)
	buildctlCmd.Env = os.Environ()

	var buildctlStdout io.Reader
	if needsLoading {
		buildctlStdout, err = buildctlCmd.StdoutPipe()
		if err != nil {
			return err
		}
	} else {
		buildctlCmd.Stdout = opts.Stdout
	}

	if !opts.Quiet {
		buildctlCmd.Stderr = opts.Stderr
	}

	if err := buildctlCmd.Start(); err != nil {
		return err
	}

	if needsLoading {
		platMC, err := platformutil.NewMatchComparer(false, opts.Platform)
		if err != nil {
			return err
		}
		if err = loadImage(ctx, buildctlStdout, globalOptions.Namespace, globalOptions.Address, globalOptions.Snapshotter, opts.Stdout, platMC, opts.Quiet); err != nil {
			return err
		}
	}

	if err = buildctlCmd.Wait(); err != nil {
		return err
	}

	if opts.IidFile != "" {
		id, err := getDigestFromMetaFile(metaFile)
		if err != nil {
			return err
		}
		if err := os.WriteFile(opts.IidFile, []byte(id), 0o644); err != nil {
			return err
		}
	}

	if len(tags) > 1 {
		imageService := cli.ImageService()
		image, err := imageService.Get(ctx, tags[0])
		if err != nil {
			return fmt.Errorf("unable to tag image: %w", err)
		}
		for _, targetRef := range tags[1:] {
			image.Name = targetRef
			if _, err := imageService.Create(ctx, image); err != nil {
				// if already exists; skip.
				if errors.Is(err, errdefs.ErrAlreadyExists) {
					if err = imageService.Delete(ctx, targetRef); err != nil {
						return err
					}
					if _, err = imageService.Create(ctx, image); err != nil {
						return err
					}
					continue
				}
				return fmt.Errorf("unable to tag image: %w", err)
			}
		}
	}

	return nil
}

// TODO: This struct and `loadImage` are duplicated with the code in `cmd/load.go`, remove it after `load.go` has been refactor
type readCounter struct {
	io.Reader
	N int
}

func loadImage(ctx context.Context, in io.Reader, namespace, address, snapshotter string, output io.Writer, platMC platforms.MatchComparer, quiet bool) error {
	// In addition to passing WithImagePlatform() to cli.Import(), we also need to pass WithDefaultPlatform() to NewClient().
	// Otherwise unpacking may fail.
	cli, ctx, cancel, err := clientutil.NewClientWithOpt(ctx, namespace, address, client.WithDefaultPlatform(platMC))
	if err != nil {
		return err
	}
	defer func() {
		cancel()
		cli.Close()
	}()

	r := &readCounter{Reader: in}

	imgs, err := cli.Import(ctx, r, client.WithDigestRef(archive.DigestTranslator(snapshotter)), client.WithSkipDigestRef(func(name string) bool { return name != "" }), client.WithImportPlatform(platMC))
	if err != nil {
		if r.N == 0 {
			// Avoid confusing "unrecognized image format"
			return errors.New("no image was built")
		}
		if errors.Is(err, images.ErrEmptyWalk) {
			err = fmt.Errorf("%w (Hint: set `--platform=PLATFORM` or `--all-platforms`)", err)
		}
		return err
	}

	for _, img := range imgs {
		image := client.NewImageWithPlatform(cli, img, platMC)

		// TODO: Show unpack status
		if !quiet {
			fmt.Fprintf(output, "unpacking %s (%s)...\n", img.Name, img.Target.Digest)
		}

		err = image.Unpack(ctx, snapshotter)
		if err != nil {
			return err
		}

		if quiet {
			fmt.Fprintln(output, img.Target.Digest)
		} else {
			fmt.Fprintf(output, "Loaded image: %s\n", img.Name)
		}
	}

	return nil
}

func generateBuildctlArgs(ctx context.Context, cli *client.Client, globalOptions *options.Global, opts *options.BuilderBuild) (
	buildctlArgs []string, needsLoading bool, metaFile string, tags []string, cleanup func(), err error) {

	output := opts.Output
	if output == "" {
		info, err := cli.Server(ctx)
		if err != nil {
			return nil, false, "", nil, nil, err
		}

		sharable, err := isImageSharable(opts.BuildKitHost, globalOptions.Namespace, info.UUID, globalOptions.Snapshotter, opts.Platform)
		if err != nil {
			return nil, false, "", nil, nil, err
		}

		if sharable {
			output = "type=image,unpack=true" // ensure the target stage is unlazied (needed for any snapshotters)
		} else {
			output = "type=docker"
			if len(opts.Platform) > 1 {
				// For avoiding `error: failed to solve: docker exporter does not currently support exporting manifest lists`
				// TODO: consider using type=oci for single-opts.Platform build too
				output = "type=oci"
			}
			needsLoading = true
		}
	} else {
		if !strings.Contains(output, "type=") {
			// should accept --output <DIR> as an alias of --output
			// type=local,dest=<DIR>
			output = "type=local,dest=" + output
		}

		if strings.Contains(output, "type=docker") || strings.Contains(output, "type=oci") {
			if !strings.Contains(output, "dest=") {
				needsLoading = true
			}
		}
	}

	if tags = strutil.DedupeStrSlice(opts.Tag); len(tags) > 0 {
		ref := tags[0]
		parsedReference, err := reference.Parse(ref)
		if err != nil {
			return nil, false, "", nil, nil, err
		}
		output += ",name=" + parsedReference.String()

		// pick the first tag and add it to output
		for idx, tag := range tags {
			parsedReference, err = reference.Parse(tag)
			if err != nil {
				return nil, false, "", nil, nil, err
			}
			tags[idx] = parsedReference.String()
		}
	} else if len(tags) == 0 {
		output += ",dangling-name-prefix=<none>"
	}

	buildctlArgs = buildkit.BuildctlBaseArgs(opts.BuildKitHost)

	buildctlArgs = append(buildctlArgs, []string{
		"build",
		"--progress=" + opts.Progress,
		"--frontend=dockerfile.v0",
		"--local=context=" + opts.BuildContext,
		"--output=" + output,
	}...)

	dir := opts.BuildContext
	file := buildkit.DefaultDockerfileName
	if opts.File != "" {
		if opts.File == "-" {
			// Super Warning: this is a special trick to update the dir variable, Don't move this line!!!!!!
			var err error
			dir, err = buildkit.WriteTempDockerfile(opts.Stdin)
			if err != nil {
				return nil, false, "", nil, nil, err
			}
			cleanup = func() {
				os.RemoveAll(dir)
			}
		} else {
			dir, file = filepath.Split(opts.File)
		}

		if dir == "" {
			dir = "."
		}
	}

	dir, file, err = buildkit.File(dir, file)
	if err != nil {
		return nil, false, "", nil, nil, err
	}

	buildCtx, err := parseContextNames(opts.ExtendedBuildContext)
	if err != nil {
		return nil, false, "", nil, nil, err
	}

	for k, v := range buildCtx {
		isURL := strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://")
		isDockerImage := strings.HasPrefix(v, "docker-image://") || strings.HasPrefix(v, "target:")

		if isURL || isDockerImage {
			buildctlArgs = append(buildctlArgs, fmt.Sprintf("--opt=context:%s=%s", k, v))
			continue
		}

		if isOCILayout := strings.HasPrefix(v, "oci-layout://"); isOCILayout {
			args, err := parseBuildContextFromOCILayout(k, v)
			if err != nil {
				return nil, false, "", nil, nil, err
			}

			buildctlArgs = append(buildctlArgs, args...)
			continue
		}

		path, err := filepath.Abs(v)
		if err != nil {
			return nil, false, "", nil, nil, err
		}
		buildctlArgs = append(buildctlArgs, fmt.Sprintf("--local=%s=%s", k, path))
		buildctlArgs = append(buildctlArgs, fmt.Sprintf("--opt=context:%s=local:%s", k, k))
	}

	buildctlArgs = append(buildctlArgs, "--local=dockerfile="+dir)
	buildctlArgs = append(buildctlArgs, "--opt=filename="+file)

	if opts.Target != "" {
		buildctlArgs = append(buildctlArgs, "--opt=target="+opts.Target)
	}

	if len(opts.Platform) > 0 {
		buildctlArgs = append(buildctlArgs, "--opt=platform="+strings.Join(opts.Platform, ","))
	}

	seenBuildArgs := make(map[string]struct{})
	for _, ba := range strutil.DedupeStrSlice(opts.BuildArgs) {
		arr := strings.Split(ba, "=")
		seenBuildArgs[arr[0]] = struct{}{}
		if len(arr) == 1 && len(arr[0]) > 0 {
			// Avoid masking default build arg value from Dockerfile if environment variable is not set
			// https://github.com/moby/moby/issues/24101
			val, ok := os.LookupEnv(arr[0])
			if ok {
				buildctlArgs = append(buildctlArgs, fmt.Sprintf("--opt=build-arg:%s=%s", ba, val))
			}
		} else if len(arr) > 1 && len(arr[0]) > 0 {
			buildctlArgs = append(buildctlArgs, "--opt=build-arg:"+ba)

			// Support `--build-arg BUILDKIT_INLINE_CACHE=1` for compatibility with `docker buildx build`
			// https://github.com/docker/buildx/blob/v0.6.3/docs/reference/buildx_build.md#-export-build-cache-to-an-external-cache-destination---cache-to
			if strings.HasPrefix(ba, "BUILDKIT_INLINE_CACHE=") {
				bic := strings.TrimPrefix(ba, "BUILDKIT_INLINE_CACHE=")
				bicParsed, err := strconv.ParseBool(bic)
				if err == nil {
					if bicParsed {
						buildctlArgs = append(buildctlArgs, "--export-cache=type=inline")
					}
				} else {
					log.L.WithError(err).Warnf("invalid BUILDKIT_INLINE_CACHE: %q", bic)
				}
			}
		} else {
			return nil, false, "", nil, nil, fmt.Errorf("invalid build arg %q", ba)
		}
	}

	// Propagate SOURCE_DATE_EPOCH from the client env
	// https://github.com/docker/buildx/pull/1482
	if v := os.Getenv("SOURCE_DATE_EPOCH"); v != "" {
		if _, ok := seenBuildArgs["SOURCE_DATE_EPOCH"]; !ok {
			buildctlArgs = append(buildctlArgs, "--opt=build-arg:SOURCE_DATE_EPOCH="+v)
		}
	}

	for _, l := range strutil.DedupeStrSlice(opts.Label) {
		buildctlArgs = append(buildctlArgs, "--opt=label:"+l)
	}

	if opts.NoCache {
		buildctlArgs = append(buildctlArgs, "--no-cache")
	}

	if opts.Pull != nil {
		switch *opts.Pull {
		case true:
			buildctlArgs = append(buildctlArgs, "--opt=image-resolve-mode=pull")
		case false:
			buildctlArgs = append(buildctlArgs, "--opt=image-resolve-mode=local")
		}
	}

	for _, s := range strutil.DedupeStrSlice(opts.Secret) {
		buildctlArgs = append(buildctlArgs, "--secret="+s)
	}

	for _, s := range strutil.DedupeStrSlice(opts.Allow) {
		buildctlArgs = append(buildctlArgs, "--allow="+s)
	}

	for _, s := range strutil.DedupeStrSlice(opts.Attest) {
		optAttestType, optAttestAttrs, _ := strings.Cut(s, ",")
		if strings.HasPrefix(optAttestType, "type=") {
			optAttestType := strings.TrimPrefix(optAttestType, "type=")
			buildctlArgs = append(buildctlArgs, fmt.Sprintf("--opt=attest:%s=%s", optAttestType, optAttestAttrs))
		} else {
			return nil, false, "", nil, nil, errors.New("attestation type not specified")
		}
	}

	for _, s := range strutil.DedupeStrSlice(opts.SSH) {
		buildctlArgs = append(buildctlArgs, "--ssh="+s)
	}

	for _, s := range strutil.DedupeStrSlice(opts.CacheFrom) {
		if !strings.Contains(s, "type=") {
			s = "type=registry,ref=" + s
		}
		buildctlArgs = append(buildctlArgs, "--import-cache="+s)
	}

	for _, s := range strutil.DedupeStrSlice(opts.CacheTo) {
		if !strings.Contains(s, "type=") {
			s = "type=registry,ref=" + s
		}
		buildctlArgs = append(buildctlArgs, "--export-cache="+s)
	}

	if !opts.Rm {
		log.L.Warn("ignoring deprecated flag: '--rm=false'")
	}

	if opts.IidFile != "" {
		file, err := os.CreateTemp("", "buildkit-meta-*")
		if err != nil {
			return nil, false, "", nil, cleanup, err
		}
		defer file.Close()
		metaFile = file.Name()
		buildctlArgs = append(buildctlArgs, "--metadata-file="+metaFile)
	}

	if opts.NetworkMode != "" {
		switch opts.NetworkMode {
		case "none":
			buildctlArgs = append(buildctlArgs, "--opt=force-network-mode="+opts.NetworkMode)
		case "host":
			buildctlArgs = append(buildctlArgs, "--opt=force-network-mode="+opts.NetworkMode, "--allow=network.host", "--allow=security.insecure")
		case "", "default":
		default:
		}
	}

	if len(opts.ExtraHosts) > 0 {
		extraHosts, err := containerutil.ParseExtraHosts(opts.ExtraHosts, globalOptions.HostGatewayIP, "=")
		if err != nil {
			return nil, false, "", nil, nil, err
		}
		buildctlArgs = append(buildctlArgs, "--opt=add-hosts="+strings.Join(extraHosts, ","))
	}

	return buildctlArgs, needsLoading, metaFile, tags, cleanup, nil
}

func getDigestFromMetaFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	defer os.Remove(path)

	metadata := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		log.L.WithError(err).Errorf("failed to unmarshal metadata file %s", path)
		return "", err
	}

	digestRaw, ok := metadata["containerimage.digest"]
	if !ok {
		return "", errors.New("failed to find containerimage.digest in metadata file")
	}

	var digest string
	if err := json.Unmarshal(digestRaw, &digest); err != nil {
		log.L.WithError(err).Errorf("failed to unmarshal digset")

		return "", err
	}

	return digest, nil
}

func isMatchingRuntimePlatform(platform string, parser PlatformParser) bool {
	p, err := parser.Parse(platform)
	if err != nil {
		return false
	}

	d := parser.DefaultSpec()
	if p.OS == d.OS && p.Architecture == d.Architecture && (p.Variant == "" || p.Variant == d.Variant) {
		return true
	}

	return false
}

func isBuildPlatformDefault(platform []string, parser PlatformParser) bool {
	if len(platform) == 0 {
		return true
	} else if len(platform) == 1 {
		return isMatchingRuntimePlatform(platform[0], parser)
	}

	return false
}

func isImageSharable(buildkitHost, namespace, uuid, snapshotter string, platform []string) (bool, error) {
	labels, err := buildkit.GetWorkerLabels(buildkitHost)
	if err != nil {
		return false, err
	}

	executor, ok := labels["org.mobyproject.buildkit.worker.executor"]
	if !ok {
		return false, nil
	}

	containerdUUID, ok := labels["org.mobyproject.buildkit.worker.client.uuid"]
	if !ok {
		return false, nil
	}

	containerdNamespace, ok := labels["org.mobyproject.buildkit.worker.client.namespace"]
	if !ok {
		return false, nil
	}

	workerSnapshotter, ok := labels["org.mobyproject.buildkit.worker.snapshotter"]
	if !ok {
		return false, nil
	}

	// NOTE: It's possible that BuildKit doesn't download the base image of non-default platform (e.g. when the provided
	//       Dockerfile doesn't contain instructions require base images like RUN) even if `--output type=image,unpack=true`
	//       is passed to BuildKit. Thus, we need to use `type=docker` or `type=oci` when we build non-default platform
	//       image using `platform` option.
	parser := new(platformParser)

	return executor == "client" && containerdUUID == uuid && containerdNamespace == namespace && workerSnapshotter == snapshotter && isBuildPlatformDefault(platform, parser), nil
}

func parseContextNames(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make(map[string]string, len(values))
	for _, value := range values {
		kv := strings.SplitN(value, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid context value: %s, expected key=value", value)
		}
		result[kv[0]] = kv[1]
	}

	return result, nil
}

var (
	ErrOCILayoutPrefixNotFound = errors.New("OCI layout prefix not found")
	ErrOCILayoutEmptyDigest    = errors.New("OCI layout cannot have empty digest")
)

func parseBuildContextFromOCILayout(name, path string) ([]string, error) {
	path, found := strings.CutPrefix(path, "oci-layout://")
	if !found {
		return []string{}, ErrOCILayoutPrefixNotFound
	}

	abspath, err := filepath.Abs(path)
	if err != nil {
		return []string{}, err
	}

	ociIndex, err := readOCIIndexFromPath(abspath)
	if err != nil {
		return []string{}, err
	}

	var digest string
	for _, manifest := range ociIndex.Manifests {
		if images.IsManifestType(manifest.MediaType) {
			digest = manifest.Digest.String()
		}
	}

	if digest == "" {
		return []string{}, ErrOCILayoutEmptyDigest
	}

	return []string{
		"--oci-layout=parent-image-key=" + abspath,
		fmt.Sprintf("--opt=context:%s=oci-layout:parent-image-key@%s", name, digest),
	}, nil
}

func readOCIIndexFromPath(path string) (*specs.Index, error) {
	ociIndexJSONFile, err := os.Open(filepath.Join(path, "index.json"))
	if err != nil {
		return nil, err
	}

	defer ociIndexJSONFile.Close()

	rawBytes, err := io.ReadAll(ociIndexJSONFile)
	if err != nil {
		return nil, err
	}

	var ociIndex *specs.Index

	err = json.Unmarshal(rawBytes, &ociIndex)
	if err != nil {
		return nil, err
	}

	return ociIndex, nil
}
