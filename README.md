# Lepton

## TL;DR

lepton is a modern containerd cli.

## Requirements

Any recent linux distro with a kernel >= 5.13 is supported.

Lepton obviously requires other components to be useful (provided in the full release):
- containerd 2.0+
- runc 1.2+ (linux only)
- cni plugins 1.6+

Building lepton itself requires:
- golang 1.23+

And building dependencies (containerd, etc):
- libseccomp

If you do want to use cgroup, lepton supports v2.

## Project goals

1. provide a ready-to-use library to easily build cli and applications communicating with containerd
2. provide a reference cli implementation, comparable to the docker cli or to nerdctl
3. primary focus is on stability, code quality and developer quality of life, not on features. Specifically,
deperecated features, older configurations, and a number of "alternative" choices are not supported

## Full distribution

Unlike nerdctl, dependencies are _not_ downloaded in binary form and repackaged.
They are compiled, from pinned commit shas, ensuring consistent compilation settings, golang version, and linking.

The full-release bundles together:
- containerd
- runc
- soci
- bypass4netns
- slirp4netns
- tini
- cni
- rootlesskit
- buildkit
- imgcrypt
- buildg

All C (or CGO) binaries are compiled as static PIE executables, with immediate binding and read-only relocations.

All pure go binaries are compiled statically with the same version of golang.

For detailed versions and compiler settings information, see the Dockerfile.

## Detailed relationship with nerdctl, and current status

Nerdctl objective is to provide a fully docker compatible experience with many advanced
or experimental additional features.
As a mature project, it is also conservatively (and rightfully) focused on backward compatibility.

Lepton departs from this in a few important ways:
- docker cli compatibility is best-effort, and will be broken where it makes sense
- before 1.0, there should be no expectation of backward compatibility - API will change,
  and only the latest versions of dependencies will be supported
- lepton is departing from the way nerdctl is storing data
  - the current filesystem layout of nerdctl needs a rehaul
- lepton is removing support for unstable or otherwise experimental, or lesser used features
  - this is meant to reduce maintenance burden, simplify code and increase quality

Furthermore, lepton overarching priority is to provide a clean SDK for people who want to author their
own stuff, specifically with more expressive and cleaner error management, better storage abstractions,
better performance, and better concurrency management.

Lepton started in 2024 as a private project, and was reset as a friendly fork of nerdctl, 14th of December 2024, from
https://github.com/containerd/nerdctl/commit/7e97f0618ceb160b044e95810e17fccf21fea3df

As such, a large fraction of its codebase is coming from https://github.com/containerd/nerdctl
(copyright The Containerd Authors, licensed under the Apache License, see NOTICE).

Lepton is regularly cherry-picking changes from nerdctl, and conversely, so far, about 100k lines of code
have been contributed back from lepton to nerdctl (last synced 2025-03-09).

Unlike nerdctl, lepton does not support (and has removed from its codebase):
- freebsd
- stargz (partly)
- cvmfs
- overlaybd
- nydus
- IPFS
- cgroup v1
- fuse-overlayfs

Also, lepton does not explicitly support and does not test anymore (might still work):
- containerd pre v2 (v1.7, v1.6)
- ubuntu 22.04 and earlier

So far, besides removal of unsupported code, lepton has been focused on cleanup, reviewing and moving "library"
packages up into https://github.com/farcloser/go-containers.


<!--
## Private notes

<!> testing notes: export CONTAINERD_NAMESPACE=lepton-test; ./extras/rootless/containerd-rootless-setuptool.sh install-buildkit-containerd




[[⬇️ **Download]**](https://github.com/containerd/nerdctl/releases)
[[📖 **Command reference]**](./docs/command-reference.md)
[[❓**FAQs & Troubleshooting]**](./docs/faq.md)
[[📚 **Additional documents]**](#additional-documents)

# nerdctl: Docker-compatible CLI for containerd

<picture>
  <source media="(prefers-color-scheme: light)" srcset="docs/images/nerdctl.svg">
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/nerdctl-white.svg">
  <img alt="logo" src="docs/images/nerdctl.svg">
</picture>

`nerdctl` is a Docker-compatible CLI for [contai**nerd**](https://containerd.io).

 ✅ Same UI/UX as `docker`

 ✅ Supports Docker Compose (`nerdctl compose up`)

 ✅ [Optional] Supports [rootless mode, without slirp overhead (bypass4netns)](./docs/rootless.md)

 ✅ [Optional] Supports [encrypted images (ocicrypt)](./docs/ocicrypt.md)

 ✅ [Optional] Supports [container image signing and verifying (cosign)](./docs/cosign.md)

nerdctl is a **non-core** subproject of containerd.

## Examples

### Basic usage

To run a container with the default `bridge` CNI network (10.4.0.0/24):

```console
# nerdctl run -it --rm alpine
```

To build an image using BuildKit:

```console
# nerdctl build -t foo /some-dockerfile-directory
# nerdctl run -it --rm foo
```

To build and send output to a local directory using BuildKit:

```console
# nerdctl build -o type=local,dest=. /some-dockerfile-directory
```

To run containers from `docker-compose.yaml`:

```console
# nerdctl compose -f ./examples/compose-wordpress/docker-compose.yaml up
```

See also [`./examples/compose-wordpress`](./examples/compose-wordpress).

### Debugging Kubernetes

To list local Kubernetes containers:

```console
# nerdctl --namespace k8s.io ps -a
```

To build an image for local Kubernetes without using registry:

```console
# nerdctl --namespace k8s.io build -t foo /some-dockerfile-directory
# kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
    - name: foo
      image: foo
      imagePullPolicy: Never
EOF
```

To load an image archive (`docker save` format or OCI format) into local Kubernetes:

```console
# nerdctl --namespace k8s.io load < /path/to/image.tar
```

To read logs (experimental):
```console
# nerdctl --namespace=k8s.io ps -a
CONTAINER ID    IMAGE                                                      COMMAND                   CREATED          STATUS    PORTS    NAMES
...
e8793b8cca8b    registry.k8s.io/coredns/coredns:v1.9.3                     "/coredns -conf /etc…"    2 minutes ago    Up                 k8s://kube-system/coredns-787d4945fb-mfx6b/coredns
...

# nerdctl --namespace=k8s.io logs -f e8793b8cca8b
[INFO] plugin/reload: Running configuration SHA512 = 591cf328cccc12bc490481273e738df59329c62c0b729d94e8b61db9961c2fa5f046dd37f1cf888b953814040d180f52594972691cd6ff41be96639138a43908
CoreDNS-1.9.3
linux/amd64, go1.18.2, 45b0a11
...
```

### Rootless mode

To launch rootless containerd:

```console
$ containerd-rootless-setuptool.sh install
```

To run a container with rootless containerd:

```console
$ nerdctl run -d -p 8080:80 --name nginx nginx:alpine
```

See [`./docs/rootless.md`](./docs/rootless.md).

## Install

Binaries are available here: <https://github.com/containerd/nerdctl/releases>

In addition to containerd, the following components should be installed:

- [CNI plugins](https://github.com/containernetworking/plugins): for using `nerdctl run`.
  - v1.1.0 or later is highly recommended.
- [BuildKit](https://github.com/moby/buildkit) (OPTIONAL): for using `nerdctl build`. BuildKit daemon (`buildkitd`) needs to be running. See also [the document about setting up BuildKit](./docs/build.md).
  - v0.11.0 or later is highly recommended. Some features, such as pruning caches with `nerdctl system prune`, do not work with older versions.
- [RootlessKit](https://github.com/rootless-containers/rootlesskit) and [slirp4netns](https://github.com/rootless-containers/slirp4netns) (OPTIONAL): for [Rootless mode](./docs/rootless.md)
  - RootlessKit needs to be v0.10.0 or later. v2.0.0 or later is recommended.
  - slirp4netns needs to be v0.4.0 or later. v1.1.7 or later is recommended.

These dependencies are included in `nerdctl-full-<VERSION>-<OS>-<ARCH>.tar.gz`, but not included in `nerdctl-<VERSION>-<OS>-<ARCH>.tar.gz`.

### Brew

On Linux systems you can install nerdctl via [brew](https://brew.sh):

```bash
brew install nerdctl
```

This is currently not supported for macOS. The section below shows how to install on macOS using brew.

### macOS

[Lima](https://github.com/lima-vm/lima) project provides Linux virtual machines for macOS, with built-in integration for nerdctl.

```console
$ brew install lima
$ limactl start
$ lima nerdctl run -d --name nginx -p 127.0.0.1:8080:80 nginx:alpine
```

### Windows

- Linux containers: Known to work on WSL2
- Windows containers: experimental support for Windows (see below for features that are currently known to work)

### Docker

To run containerd and nerdctl inside Docker:

```bash
docker build -t nerdctl .
docker run -it --rm --privileged nerdctl
```

## Motivation

The goal of `nerdctl` is to facilitate experimenting the cutting-edge features of containerd that are not present in Docker (see below).

Note that competing with Docker is _not_ the goal of `nerdctl`. Those cutting-edge features are expected to be eventually available in Docker as well.

Also, `nerdctl` might be potentially useful for debugging Kubernetes clusters, but it is not the primary goal.

## Features present in `nerdctl` but not present in Docker

Major:

- On-demand image pulling (lazy-pulling) using [SOCI](./docs/soci.md) Snapshotter: `nerdctl --snapshotter=soci run IMAGE` .
- [Image encryption and decryption using ocicrypt (imgcrypt)](./docs/ocicrypt.md): `nerdctl image (encrypt|decrypt) SRC DST`
- [Cosign integration](./docs/cosign.md): `nerdctl pull --verify=cosign` and `nerdctl push --sign=cosign`, and [in Compose](./docs/cosign.md#cosign-in-compose)
- [Accelerated rootless containers using bypass4netns](./docs/rootless.md): `nerdctl run --annotation nerdctl/bypass4netns=true`

Minor:

- Namespacing: `nerdctl --namespace=<NS> ps` .
  (NOTE: All Kubernetes containers are in the `k8s.io` containerd namespace regardless to Kubernetes namespaces)
- Exporting Docker/OCI dual-format archives: `nerdctl save` .
- Importing OCI archives as well as Docker archives: `nerdctl load` .
- Specifying a non-image rootfs: `nerdctl run -it --rootfs <ROOTFS> /bin/sh` . The CLI syntax conforms to Podman convention.
- Connecting a container to multiple networks at once: `nerdctl run --net foo --net bar`
- Better multi-platform support, e.g., `nerdctl pull --all-platforms IMAGE`
- Applying an (existing) AppArmor profile to rootless containers: `nerdctl run --security-opt apparmor=<PROFILE>`.
  Use `sudo nerdctl apparmor load` to load the `nerdctl-default` profile.
- Systemd compatibility support: `nerdctl run --systemd=always`

Trivial:

- Inspecting raw OCI config: `nerdctl container inspect --mode=native` .

## Features implemented in `nerdctl` ahead of Docker

- Recursive read-only (RRO) bind-mount: `nerdctl run -v /mnt:/mnt:rro` (make children such as `/mnt/usb` to be read-only, too).
  Requires kernel >= 5.12.
The same feature was later introduced in Docker v25 with a different syntax. nerdctl will support Docker v25 syntax too in the future.
## Similar tools

- [`ctr`](https://github.com/containerd/containerd/tree/main/cmd/ctr): incompatible with Docker CLI, and not friendly to users.
  Notably, `ctr` lacks the equivalents of the following nerdctl commands:
  - `nerdctl run -p <PORT>`
  - `nerdctl run --restart=always --net=bridge`
  - `nerdctl pull` with `~/.docker/config.json` and credential helper binaries such as `docker-credential-ecr-login`
  - `nerdctl logs`
  - `nerdctl build`
  - `nerdctl compose up`

- [`crictl`](https://github.com/kubernetes-sigs/cri-tools): incompatible with Docker CLI, not friendly to users, and does not support non-CRI features
- [k3c v0.2 (abandoned)](https://github.com/rancher/k3c/tree/v0.2.1): needs an extra daemon, and does not support non-CRI features
- [Rancher Kim (nee k3c v0.3)](https://github.com/rancher/kim): needs Kubernetes, and only focuses on image management commands such as `kim build` and `kim push`
- [PouchContainer (abandoned?)](https://github.com/alibaba/pouch): needs an extra daemon

## Developer guide

nerdctl is a containerd **non-core** subproject, licensed under the [Apache 2.0 license](./LICENSE).
As a containerd non-core subproject, you will find the:

- [Project governance](https://github.com/containerd/project/blob/main/GOVERNANCE.md),
- [Maintainers](./MAINTAINERS),
- and [Contributing guidelines](https://github.com/containerd/project/blob/main/CONTRIBUTING.md)

information in our [`containerd/project`](https://github.com/containerd/project) repository.

### Compiling nerdctl from source

Run `make && sudo make install`.

See the header of [`go.mod`](./go.mod) for the minimum supported version of Go.

Using `go install go.farcloser.world/lepton/cmd/lepton` is possible, but unrecommended because it does not fill version strings printed in `nerdctl version`

### Testing

See [testing nerdctl](docs/testing/README.md).

### Contributing to nerdctl

Lots of commands and flags are currently missing. Pull requests are highly welcome.

Please certify your [Developer Certificate of Origin (DCO)](https://developercertificate.org/), by signing off your commit with `git commit -s` and with your real name.

# Command reference

Moved to [`./docs/command-reference.md`](./docs/command-reference.md)

# Additional documents

Configuration guide:

- [`./docs/config.md`](./docs/config.md): Configuration (`/etc/nerdctl/nerdctl.toml`, `~/.config/nerdctl/nerdctl.toml`)
- [`./docs/registry.md`](./docs/registry.md): Registry authentication (`~/.docker/config.json`)

Basic features:

- [`./docs/compose.md`](./docs/compose.md):   Compose
- [`./docs/rootless.md`](./docs/rootless.md): Rootless mode
- [`./docs/cni.md`](./docs/cni.md): CNI for containers network
- [`./docs/build.md`](./docs/build.md): `nerdctl build` with BuildKit

Advanced features:

- [`./docs/ocicrypt.md`](./docs/ocicrypt.md): Running encrypted images
- [`./docs/gpu.md`](./docs/gpu.md):           Using GPUs inside containers
- [`./docs/multi-platform.md`](./docs/multi-platform.md):  Multi-platform mode

Experimental features:

- [`./docs/experimental.md`](./docs/experimental.md):  Experimental features
- [`./docs/builder-debug.md`](./docs/builder-debug.md): Interactive debugging of Dockerfile

Implementation details:

- [`./docs/dir.md`](./docs/dir.md):           Directory layout (`/var/lib/nerdctl`)

Misc:

- [`./docs/faq.md`](./docs/faq.md): FAQs and Troubleshooting

-->
