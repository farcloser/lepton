#   Copyright Farcloser.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# -----------------------------------------------------------------------------
# Usage: `docker run -it --privileged <IMAGE>`. Make sure to add `-t` and `--privileged`.

# TODO: verify commit hash

ARG BINARY_NAME=lepton

# Basic deps
ARG CONTAINERD_VERSION=v2.0.3
ARG RUNC_VERSION=v1.2.5
ARG CNI_PLUGINS_VERSION=v1.6.2

# Extra dependencies
# - Build
ARG BUILDKIT_VERSION=v0.19.0
# - Encryption
ARG IMGCRYPT_VERSION=v2.0.0
# - Rootless
ARG ROOTLESSKIT_VERSION=v2.3.2
ARG SLIRP4NETNS_VERSION=v1.3.1
# - bypass4netns
ARG BYPASS4NETNS_VERSION=v0.4.2
# - fuse-overlayfs
ARG FUSE_OVERLAYFS_VERSION=v1.14
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION=v2.1.1
# - Init
ARG TINI_VERSION=v0.19.0
# - Debug
ARG BUILDG_VERSION=v0.4.1

# Test and demo dependencies
ARG UBUNTU_VERSION=24.04
ARG SOCI_SNAPSHOTTER_VERSION=0.8.0

# Tooling versions
ARG DEBIAN_VERSION=bookworm
ARG XX_VERSION=1.6.1
ARG GO_VERSION=1.24.0
ARG CONTAINERIZED_SYSTEMD_VERSION=v0.1.1

########################################################################################################################
# Base images
# These stages are purely tooling that other stages can leverage.
# They are highly cacheable, and will only change if one of the following is changing:
# - DEBIAN_VERSION (=bookworm) or content of it
# - XX_VERSION
# - GO_VERSION
# - CONTAINERIZED_SYSTEMD_VERSION
########################################################################################################################
# tooling-base is the single base image we use for all other tooling image
FROM --platform=$BUILDPLATFORM ghcr.io/apostasie/debian:$DEBIAN_VERSION AS tooling-base
SHELL ["/bin/bash", "-o", "errexit", "-o", "errtrace", "-o", "functrace", "-o", "nounset", "-o", "pipefail", "-c"]
ENV DEBIAN_FRONTEND="noninteractive"
ENV TERM="xterm"
ENV LANG="C.UTF-8"
ENV LC_ALL="C.UTF-8"
ENV TZ="America/Los_Angeles"
ARG BINARY_NAME
RUN apt-get update -qq >/dev/null && apt-get install -qq --no-install-recommends \
  ca-certificates >/dev/null

# xx provides tooling to ease cross-compilation
FROM --platform=$BUILDPLATFORM ghcr.io/apostasie/xx:$XX_VERSION AS tooling-xx

# tooling-downloader purpose is to enable later stages to download content directly using curl
FROM --platform=$BUILDPLATFORM tooling-base AS tooling-downloader
# Current work directory where downloads will arrive
WORKDIR /src
# /out is meant to hold final / distributable assets
RUN mkdir -p /out/bin
# This directory is meant to hold transient information useful to build the final README (VERSION, LICENSE, etc)
RUN mkdir -p /metadata
# Get curl and jq
RUN apt-get install -qq --no-install-recommends \
  curl \
  jq >/dev/null

# tooling-downloader-golang will download a golang archive and expand it into /out/usr/local
# You may set GO_VERSION to an explicit, complete version (eg: 1.23.0), or you can also set it to:
# - canary: that will retrieve the latest alpha/beta/RC
# - stable (or ""): that will retrieve the latest stable version
# Note that for these last two, you need to bust the cache for this stage if you expect a refresh
# Finally note that we are retrieving both architectures we are currently supporting (arm64 and amd64) in one stage,
# and do NOT leverage TARGETARCH, as that would force cross compilation to use a non-native binary in dependent stages.
FROM --platform=$BUILDPLATFORM tooling-downloader AS tooling-downloader-golang
ARG BUILDPLATFORM
ARG GO_VERSION
# This run does:
# a. retrieve golang list of versions
# b. parse it to extract just the files for the requested GO_VERSION and GOOS
# c. for both arm64 and amd64, extract the final filename
# d. download the archive and extract it to /out/usr/local/GOOS/GOARCH
# Consuming stages later on can just COPY --from=tooling-downloader-golang /out/usr/local/$BUILDPLATFORM /usr/local
# to get native go for their current execution platform
# Note that though we dynamically retrieve GOOS here, we only support linux
RUN os="${BUILDPLATFORM%%/*}"; \
    all_versions="$(curl -fsSL --proto '=https' --tlsv1.2 "https://go.dev/dl/?mode=json&include=all")"; \
    candidates="$(case "$GO_VERSION" in \
      canary) condition=".stable==false" ;; \
      stable|"") condition=".stable==true" ;; \
      *) condition='.version | startswith("go'"$GO_VERSION"'")' ;; \
    esac; \
    jq -rc 'map(select('"$condition"'))[0].files | map(select(.os=="'"$os"'"))' <(printf "$all_versions"))"; \
    arch=arm64; \
    filename="$(jq -r 'map(select(.arch=="'"$arch"'"))[0].filename' <(printf "$candidates"))"; \
    mkdir -p /out/usr/local/linux/"$arch"; \
    [ "$filename" != "" ] && curl -fsSL --proto '=https' --tlsv1.2 https://go.dev/dl/"$filename" | tar xzC /out/usr/local/linux/"$arch" || {  \
      echo "Failed retrieving go download for $arch: $GO_VERSION"; \
      exit 1; \
    }; \
    arch=amd64; \
    filename="$(jq -r 'map(select(.arch=="'"$arch"'"))[0].filename' <(printf "$candidates"))"; \
    mkdir -p /out/usr/local/linux/"$arch"; \
    [ "$filename" != "" ] && curl -fsSL --proto '=https' --tlsv1.2 https://go.dev/dl/"$filename" | tar xzC /out/usr/local/linux/"$arch" || {  \
      echo "Failed retrieving go download for $arch: $GO_VERSION"; \
      exit 1; \
    }

# tooling-builder is a go enabled stage with minimal build tooling installed that can be used to build non-cgo projects
FROM --platform=$BUILDPLATFORM tooling-base AS tooling-builder
# We do not want fancy display when building
ENV NO_COLOR=true
ARG BUILDPLATFORM
WORKDIR /src
RUN mkdir -p /out/bin
RUN mkdir -p /metadata
# libmagic-mgc libmagic1 file: runc, ipfs, bypassnetns
RUN apt-get install -qq --no-install-recommends \
  git \
  make \
  libmagic-mgc libmagic1 file >/dev/null
# Prevent git from complaining on detached head
RUN git config --global advice.detachedHead false
# Add cross compilation tools
COPY --from=tooling-xx / /
# Add golang
ENV PATH="/root/go/bin:/usr/local/go/bin:$PATH"
COPY --from=tooling-downloader-golang /out/usr/local/$BUILDPLATFORM /usr/local
# Disable CGO
ENV CGO_ENABLED=0
# Set xx-go as go
ENV GO=xx-go

# tooling-builder-with-c-dependencies is an expansion of the previous stages that adds extra c dependencies.
# It is meant for (cross-compilation of) c and cgo projects.
FROM --platform=$BUILDPLATFORM tooling-builder AS tooling-builder-with-c-dependencies
ARG TARGETARCH
# libbtrfs: for containerd
# libseccomp: for runc and bypass4netns
RUN xx-apt-get install -qq --no-install-recommends \
  binutils \
  gcc \
  dpkg-dev \
  libc6-dev \
  libbtrfs-dev \
  libseccomp-dev \
  pkg-config >/dev/null
# Set default linker options for CGO projects
ENV GOFLAGS="$GOFLAGS -ldflags=-linkmode=external -tags=netgo,osusergo"
# Enable CGO
ENV CGO_ENABLED=1

# tooling-runtime is the base stage that is used to build demo and testing images
# Note that unlike every other tooling- stage, this is a multi-architecture stage
FROM ghcr.io/apostasie/ubuntu:${UBUNTU_VERSION} AS tooling-runtime
SHELL ["/bin/bash", "-o", "errexit", "-o", "errtrace", "-o", "functrace", "-o", "nounset", "-o", "pipefail", "-c"]
ENV DEBIAN_FRONTEND="noninteractive"
ENV TERM="xterm"
ENV LANG="C.UTF-8"
ENV LC_ALL="C.UTF-8"
ENV TZ="America/Los_Angeles"
ARG BINARY_NAME
# fuse3 is required by stargz snapshotter
RUN apt-get update -qq && apt-get install -qq --no-install-recommends \
  ca-certificates \
  apparmor \
  bash-completion \
  iproute2 iptables \
  dbus dbus-user-session systemd systemd-sysv \
  curl \
  fuse3 >/dev/null
ARG CONTAINERIZED_SYSTEMD_VERSION
RUN curl -o /docker-entrypoint.sh -fsSL --proto '=https' --tlsv1.2 https://raw.githubusercontent.com/AkihiroSuda/containerized-systemd/${CONTAINERIZED_SYSTEMD_VERSION}/docker-entrypoint.sh && \
  chmod +x /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]

#COPY /etc/systemd/system/docker-entrypoint.target
#systemctl mask systemd-firstboot.service systemd-udevd.service systemd-modules-load.service
#systemctl unmask systemd-logind

CMD ["bash", "--login", "-i"]

########################################################################################################################
# Dependencies targets
# These stages are downloading and building all projects, using the base tooling stages from above
# Note that:
# - clone are restricted to the exact tag and depth 1 to reduce overall clone traffic
# - clones (and where appropriate go mod download) are in separate single stages so that we clone only ONCE
# - clones checkouts are then mounted into the build stage, so that we avoid creating useless copy layers
# - clone sources are mounted `locked` to avoid conflicts between parallel cross compilations using the same src
# (`locked` comes at a price, as parallel cross compilation build will be sequential, but it beats `private` in term
# of performance)
# - on build, GOPROXY is set to `off` to ensure we did perform any network operation properly during the download phase
# - mod downloads are using a shared cache so that dependencies shared accross projects are not duplicated
########################################################################################################################

# IMPORTANT: containerd is compiled statically so that we avoid having to build for both glibc and musl
# That comes at a cost:
# - pkcs11 support is very likely broken, as it relies on dlopen
# - we cannnot build with PIE
# We *could* instead build for both musl and glibc here and link (pie) dynamically, but that is very likely
# a full can of worms wrt binary compatibility of different host system libc versions.
# containerd itself is ambiguous on its position wrt static (called out in the documentation as a bad idea,
# though the official release is indeed static).
# Finally note that building statically also impairs mDNS, so, do not expect resolution to work for containerd
# level operations (that would require netcgo and linking dynamically against *glibc* - musl does not support NSS anyhow)
# See https://medium.com/@dubo-dubon-duponey/a-beginners-guide-to-cross-compiling-static-cgo-pie-binaries-golang-1-16-792eea92d5aa
# for the full tartine.
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-containerd
ARG CONTAINERD_VERSION
RUN echo "- containerd: ${CONTAINERD_VERSION}" >> /metadata/VERSION
# containerd does vendor its deps, no need to mod download
RUN git clone --depth 1 --branch "$CONTAINERD_VERSION" https://github.com/containerd/containerd.git .

# Note that only containerd itself is built with CGO. For ctr and shim, we do not need CGO, so, reset the flags there.
FROM --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-containerd
ARG TARGETARCH
ENV GOPROXY=off
RUN --mount=target=/src,type=cache,from=dependencies-download-containerd,source=/src,sharing=locked \
  make bin/containerd STATIC=1 && \
  GOFLAGS="" CGO_ENABLED=0 make bin/ctr && \
  GOFLAGS="" CGO_ENABLED=0 make bin/containerd-shim-runc-v2 && \
  cp -a containerd.service / && \
  cp -a bin/containerd bin/containerd-shim-runc-v2 bin/ctr \
    /out/bin

FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-runc
ARG RUNC_VERSION
RUN echo "- runc: ${RUNC_VERSION}" >> /metadata/VERSION
# runc does vendor its deps, no need to mod download
RUN git clone --depth 1 --branch "$RUNC_VERSION" https://github.com/opencontainers/runc.git .

FROM --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-runc
ARG TARGETARCH
ENV GOPROXY=off
RUN --mount=target=/src,type=cache,from=dependencies-download-runc,source=/src,sharing=locked \
  CC=$(xx-info)-gcc STRIP=$(xx-info)-strip make static && \
  cp -a runc /out/bin

# bypass4netns
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-bypass4netns
ARG BYPASS4NETNS_VERSION
RUN echo "- bypass4netns: ${BYPASS4NETNS_VERSION}" >> /metadata/VERSION
RUN git clone --depth 1 --branch "$BYPASS4NETNS_VERSION" https://github.com/rootless-containers/bypass4netns.git .

FROM --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-bypass4netns
ARG TARGETARCH
# We would need to call the dynamic task to be able to build static _pie_ instead of static
# Also note that the make file passes -ldflags on the command-line, so we need to re-stuff `linkmode` into their custom
# "GO_BUILD_LDFLAGS" variable.
ENV GO_BUILD_LDFLAGS="-linkmode=external"
RUN --mount=target=/src,type=cache,from=dependencies-download-bypass4netns,source=/src,sharing=locked \
    --mount=target=/root/go/pkg/mod,type=cache \
  make static && \
  cp -a bypass4netns bypass4netnsd /out/bin

# imgcrypt
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-imgcrypt
ARG IMGCRYPT_VERSION
RUN echo "- imgcrypt: ${IMGCRYPT_VERSION}" >> /metadata/VERSION
RUN git clone --depth 1 --branch "$IMGCRYPT_VERSION" https://github.com/containerd/imgcrypt.git .

# imgrcrypt does not allow overriding GO, so, wrap instead
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-imgcrypt
ARG TARGETARCH
RUN --mount=target=/src,type=cache,from=dependencies-download-imgcrypt,source=/src,sharing=locked \
    --mount=target=/root/go/pkg/mod,type=cache \
  xx-go --wrap && \
  make && \
  DESTDIR=/out make install

# buildg
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-buildg
ARG BUILDG_VERSION
RUN echo "- buildg: ${BUILDG_VERSION}" >> /metadata/VERSION
RUN git clone --depth 1 --branch "$BUILDG_VERSION" https://github.com/ktock/buildg.git .

# buildg does not allow overriding GO, so, wrap instead
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-buildg
ARG TARGETARCH
RUN --mount=target=/src,type=cache,from=dependencies-download-buildg,source=/src,sharing=locked \
    --mount=target=/root/go/pkg/mod,type=cache \
  xx-go --wrap && \
  CMD_DESTDIR=/out make buildg install

# cli binary is built from the local context
FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-cli
COPY . /src
COPY docs /out/share/doc/"$BINARY_NAME"/docs
RUN { echo "# "$BINARY_NAME" (full distribution)"; echo "- "$BINARY_NAME": $(git describe --tags)"; } \
  > /metadata/VERSION

FROM --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-cli
ARG TARGETARCH
RUN  --mount=target=/root/go/pkg/mod,type=cache \
     --mount=target=/src,type=cache,from=dependencies-download-cli,source=/src,sharing=locked \
  BINDIR=/out/bin make build install

# dependencies-download will retrieve all dependencies that are not compiled from source and used in binary form
# FIXME: some of these binary dependencies seem very large. We might consider building some of these from source instead.
FROM --platform=$BUILDPLATFORM tooling-downloader AS dependencies-download
ARG TARGETARCH
# Last updated in 2020
ARG TINI_VERSION
# Updated 1 time in 2024
ARG FUSE_OVERLAYFS_VERSION
# Updated 1 time in 2024
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION
# Updated 3 times in 2024
ARG SLIRP4NETNS_VERSION
# Updated 4 times in 2024
ARG STARGZ_SNAPSHOTTER_VERSION
# Updated 5 times in 2024
ARG CNI_PLUGINS_VERSION
# Updated 8 times in 2024, and is also sharding in two different major versions by the builder
ARG ROOTLESSKIT_VERSION
# Updates often
ARG BUILDKIT_VERSION
# Get the debian arch
RUN echo "$TARGETARCH" | sed -e s/amd64/x86_64/ -e s/arm64/aarch64/ | tee /target_uname_m
# Copy in the shasums - note this any change in there will invalidate the cache for all subsequent steps
# As this does not happen very often, this is fine.
COPY ./Dockerfile.d/SHA256SUMS.d /SHA256SUMS.d

# C
RUN fname="tini-static-$TARGETARCH" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/krallin/tini/releases/download/${TINI_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/tini-${TINI_VERSION}" | sha256sum -c && \
  cp -a "$fname" /out/bin/tini && chmod +x /out/bin/tini && \
  rm "$fname" && \
  echo "- Tini: ${TINI_VERSION}" >> /metadata/VERSION && \
  echo "- bin/tini: [MIT License](https://github.com/krallin/tini/blob/${TINI_VERSION}/LICENSE)" >> /metadata/LICENSE

# C
RUN fname="fuse-overlayfs-$(cat /target_uname_m)" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/containers/fuse-overlayfs/releases/download/${FUSE_OVERLAYFS_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/fuse-overlayfs-${FUSE_OVERLAYFS_VERSION}" | sha256sum -c && \
  mv "$fname" /out/bin/fuse-overlayfs && \
  chmod +x /out/bin/fuse-overlayfs && \
  echo "- fuse-overlayfs: ${FUSE_OVERLAYFS_VERSION}" >> /metadata/VERSION && \
  echo "- bin/fuse-overlayfs: [GNU GENERAL PUBLIC LICENSE, Version 2](https://github.com/containers/fuse-overlayfs/blob/${FUSE_OVERLAYFS_VERSION}/COPYING)" >> /metadata/LICENSE

# golang CGO_ENABLED=0
RUN fname="containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION/v}-${TARGETOS:-linux}-$TARGETARCH.tar.gz" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/containerd/fuse-overlayfs-snapshotter/releases/download/${CONTAINERD_FUSE_OVERLAYFS_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION}" | sha256sum -c && \
  tar xzf "$fname" -C /out/bin && \
  rm -f "$fname" && \
  echo "- containerd-fuse-overlayfs: ${CONTAINERD_FUSE_OVERLAYFS_VERSION}" >> /metadata/VERSION

# C
RUN fname="slirp4netns-$(cat /target_uname_m)" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/rootless-containers/slirp4netns/releases/download/${SLIRP4NETNS_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/slirp4netns-${SLIRP4NETNS_VERSION}" | sha256sum -c && \
  mv "$fname" /out/bin/slirp4netns && \
  chmod +x /out/bin/slirp4netns && \
  echo "- slirp4netns: ${SLIRP4NETNS_VERSION}" >> /metadata/VERSION && \
  echo "- bin/slirp4netns:    [GNU GENERAL PUBLIC LICENSE, Version 2](https://github.com/rootless-containers/slirp4netns/blob/${SLIRP4NETNS_VERSION}/COPYING)" >> /metadata/LICENSE

# golang CGO_ENABLED=0, vendored
RUN fname="cni-plugins-${TARGETOS:-linux}-$TARGETARCH-${CNI_PLUGINS_VERSION}.tgz" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/cni-plugins-${CNI_PLUGINS_VERSION}" | sha256sum -c && \
  mkdir -p /out/libexec/cni && \
  tar xzf "$fname" -C /out/libexec/cni && \
  rm -f "$fname" && \
  echo "- CNI plugins: ${CNI_PLUGINS_VERSION}" >> /metadata/VERSION

# golang CGO_ENABLED=0?
RUN fname="buildkit-${BUILDKIT_VERSION}.${TARGETOS:-linux}-$TARGETARCH.tar.gz" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/moby/buildkit/releases/download/${BUILDKIT_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/buildkit-${BUILDKIT_VERSION}" | sha256sum -c && \
  tar xzf "$fname" -C /out && \
  rm -f "$fname" /out/bin/buildkit-qemu-* /out/bin/buildkit-cni-* /out/bin/buildkit-runc && \
  for f in /out/libexec/cni/*; do ln -s ../libexec/cni/$(basename $f) /out/bin/buildkit-cni-$(basename $f); done && \
  rm /out/bin/buildkit-cni-LICENSE /out/bin/buildkit-cni-README.md && \
  echo "- BuildKit: ${BUILDKIT_VERSION}" >> /metadata/VERSION

# golang CGO_ENABLED=0
RUN fname="rootlesskit-$(cat /target_uname_m).tar.gz" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/rootless-containers/rootlesskit/releases/download/${ROOTLESSKIT_VERSION}/${fname}" && \
  grep "$fname" "/SHA256SUMS.d/rootlesskit-${ROOTLESSKIT_VERSION}" | sha256sum -c && \
  tar xzf "$fname" -C /out/bin && \
  rm -f "$fname" /out/bin/rootlesskit-docker-proxy && \
  echo "- RootlessKit: ${ROOTLESSKIT_VERSION}" >> /metadata/VERSION

# These are not part of the full-release so they are in their own stage (soci, cosign)
FROM --platform=$BUILDPLATFORM tooling-downloader AS dependencies-download-no-release
ARG TARGETARCH
ARG SOCI_SNAPSHOTTER_VERSION
RUN fname="soci-snapshotter-${SOCI_SNAPSHOTTER_VERSION}-${TARGETOS:-linux}-$TARGETARCH.tar.gz" && \
  curl -o "$fname" -fsSL --proto '=https' --tlsv1.2 "https://github.com/awslabs/soci-snapshotter/releases/download/v${SOCI_SNAPSHOTTER_VERSION}/${fname}" && \
  tar xzf "$fname" -C /out/bin soci soci-snapshotter-grpc && \
  rm "$fname"
# FIXME: parameterize version
COPY --from=ghcr.io/sigstore/cosign/cosign:v2.2.3@sha256:8fc9cad121611e8479f65f79f2e5bea58949e8a87ffac2a42cb99cf0ff079ba7 /ko-app/cosign /out/bin/cosign

########################################################################################################################
# Assembly
# These stages are here to assemble all build and download dependencies together for various purposes:
# - full-release distribution
# - test-integration images
# - demo image
########################################################################################################################
# assembly-release-assets is single platform, and prepares the non-architecture dependent files for the full release
FROM --platform=$BUILDPLATFORM tooling-builder AS assembly-release-assets
RUN mkdir -p /out/lib/systemd/system /out/share/doc/"$BINARY_NAME"-full
COPY --from=dependencies-build-containerd /containerd.service /out/lib/systemd/system/containerd.service
# NOTE: github.com/moby/buildkit/examples/systemd is not included in BuildKit v0.8.x, will be included in v0.9.x
# FIXME: now that we are at buildkit 0.20+, do we want to move over to their example systemd file?
RUN cd /out/lib/systemd/system && \
  sedcomm='s@bin/containerd@bin/buildkitd@g; s@(Description|Documentation)=.*@@' && \
  sed -E "${sedcomm}" containerd.service > buildkit.service && \
  echo "" >> buildkit.service && \
  echo "# This file was converted from containerd.service, with \`sed -E '${sedcomm}'\`" >> buildkit.service
COPY --from=dependencies-download-cli /out/share /out/share
RUN --mount=target=/metadata,type=cache,from=dependencies-download-cli,source=/metadata \
    cat /metadata/VERSION > /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata,type=cache,from=dependencies-download-containerd,source=/metadata \
    cat /metadata/VERSION >> /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata,type=cache,from=dependencies-download-runc,source=/metadata \
    cat /metadata/VERSION >> /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata,type=cache,from=dependencies-download-bypass4netns,source=/metadata \
    cat /metadata/VERSION >> /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata,type=cache,from=dependencies-download-imgcrypt,source=/metadata \
    cat /metadata/VERSION >> /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata,type=cache,from=dependencies-download-buildg,source=/metadata \
    cat /metadata/VERSION >> /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata,type=cache,from=dependencies-download,source=/metadata \
    cat /metadata/VERSION >> /out/share/doc/"$BINARY_NAME"-full/README.md
RUN --mount=target=/metadata/LICENSE,type=cache,from=dependencies-download,source=/metadata/LICENSE \
  echo "" >> /out/share/doc/"$BINARY_NAME"-full/README.md && \
  echo "## License" >> /out/share/doc/"$BINARY_NAME"-full/README.md && \
  cat /metadata/LICENSE >>  /out/share/doc/"$BINARY_NAME"-full/README.md && \
  echo "- bin/{runc,bypass4netns,bypass4netnsd}: Apache License 2.0, statically linked with libseccomp ([LGPL 2.1](https://github.com/seccomp/libseccomp/blob/main/LICENSE), source code available at https://github.com/seccomp/libseccomp/)" >> /out/share/doc/"$BINARY_NAME"-full/README.md && \
  echo "- Other files: [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)" >> /out/share/doc/"$BINARY_NAME"-full/README.md

# assembly-release is multi-architecture, and is the stage assembling all assets for full-release
# Once done, shasums will be generated and stuffed in to produce the full release
FROM scratch AS assembly-release
COPY --from=dependencies-build-containerd /out /
COPY --from=dependencies-build-runc /out /
COPY --from=dependencies-build-bypass4netns /out /
COPY --from=dependencies-build-imgcrypt /out /
COPY --from=dependencies-build-buildg /out /
COPY --from=dependencies-download /out /
COPY --from=dependencies-build-cli /out /
COPY --from=assembly-release-assets /out /

# assembly-shasum prepares the shasum file from above
FROM --platform=$BUILDPLATFORM tooling-builder AS assembly-shasum
ARG TARGETARCH
RUN --mount=target=/src,type=cache,from=assembly-release,source=/ \
  (cd /src && find ! -type d | sort | xargs sha256sum > /out/SHA256SUMS ) && \
  chown -R 0:0 /out

# assembly-runtime is the basis for the test integration environment
# this stage purposedly does NOT depend on the cli, so, it should be highly cacheable
FROM tooling-runtime AS assembly-runtime
ARG TARGETPLATFORM
# FIXME: finish removing unbuffer from the test codebase and then remove expect
# SSH is necessary for rootless testing (to create systemd user session).
# (`sudo` does not work for this purpose,
# OTOH `machinectl shell` can create the session but does not propagate exit code)
RUN apt-get install -qq --no-install-recommends \
  expect \
  git \
  make \
  uidmap \
  openssh-server \
  openssh-client >/dev/null
# Add go
ENV PATH="/root/go/bin:/usr/local/go/bin:$PATH"
COPY --from=tooling-downloader-golang /out/usr/local/$TARGETPLATFORM /usr/local
# Add all needed dependencies, but not the cli yet to avoid busting cache
COPY --from=dependencies-build-containerd /out /usr/local
COPY --from=dependencies-build-runc /out /usr/local
COPY --from=dependencies-build-bypass4netns /out /usr/local
COPY --from=dependencies-build-imgcrypt /out /usr/local
COPY --from=dependencies-build-buildg /out /usr/local
COPY --from=dependencies-download /out /usr/local
COPY --from=dependencies-download-no-release /out /usr/local/
# Add assets
COPY --from=dependencies-build-containerd /containerd.service /usr/local/lib/systemd/system/containerd.service
# NOTE: github.com/moby/buildkit/examples/systemd is not included in BuildKit v0.8.x, will be included in v0.9.x
# FIXME: now that we are at buildkit 0.20+, do we want to move over to their example systemd file?
RUN cd /usr/local/lib/systemd/system && \
  sedcomm='s@bin/containerd@bin/buildkitd@g; s@(Description|Documentation)=.*@@' && \
  sed -E "${sedcomm}" containerd.service > buildkit.service && \
  echo "" >> buildkit.service && \
  echo "# This file was converted from containerd.service, with \`sed -E '${sedcomm}'\`" >> buildkit.service
# Final preparations
RUN cp /usr/local/bin/tini /usr/local/bin/tini-custom
RUN mkdir -p -m 0755 /etc/cni
VOLUME /var/lib/containerd
VOLUME /var/lib/buildkit
VOLUME /var/lib/containerd-stargz-grpc
VOLUME /var/lib/"$BINARY_NAME"
VOLUME /tmp

########################################################################################################################
# Final
# These stages are high-level targets that correspond to a specific task (release, integration, etc)
########################################################################################################################
# release-full is the final stage producing the -full releases, including SHASUM
FROM assembly-release AS release-full
# Stuff in the shasums
COPY --from=assembly-shasum /out/SHA256SUMS /share/doc/"$BINARY_NAME"-full/SHA256SUMS

# test-integration is the final stage for the integration testing environment
# it is multi-architecture, and not fully cacheable, as changing anything in the cli will invalidate cache here
FROM assembly-runtime AS test-integration
ARG TARGETARCH
WORKDIR /src
# Copy config and service files
COPY ./Dockerfile.d/etc_containerd_config.toml /etc/containerd/config.toml
COPY ./Dockerfile.d/etc_buildkit_buildkitd.toml /etc/buildkit/buildkitd.toml
COPY ./Dockerfile.d/test-integration-buildkit-test.service /usr/local/lib/systemd/system/
COPY ./Dockerfile.d/test-integration-soci-snapshotter.service /usr/local/lib/systemd/system/
# using test integration containerd config
COPY ./Dockerfile.d/test-integration-etc_containerd_config.toml /etc/containerd/config.toml
RUN perl -pi -e 's/multi-user.target/docker-entrypoint.target/g' /usr/local/lib/systemd/system/*.service
# install ipfs service. avoid using 5001(api)/8080(gateway) which are reserved by tests.
RUN systemctl enable containerd  \
    buildkit \
    test-integration-buildkit-test  \
    test-integration-soci-snapshotter
# Install dev tools
COPY Makefile .
RUN --mount=target=/root/go/pkg/mod,type=cache \
  apt-get install -qq file; file go; \
  go version; \
  NO_COLOR=true make install-dev-tools; chmod -R a+rx /root/go/bin
# Add binary and source
COPY --from=dependencies-download-cli /src /src
# Warm-up cache for runtime
RUN go mod download
COPY --from=dependencies-build-cli /out /usr/local
# Install shell completion
RUN mkdir -p /etc/bash_completion.d && \
  "$BINARY_NAME" completion bash >/etc/bash_completion.d/"$BINARY_NAME"
CMD ["./hack/test-integration.sh"]

# test-integration-rootless
FROM test-integration AS test-integration-rootless
# TODO: update containerized-systemd to enable sshd by default, or allow `systemctl wants <TARGET> ssh` here
RUN ssh-keygen -q -t rsa -f /root/.ssh/id_rsa -N '' && \
  useradd -m -s /bin/bash rootless && \
  mkdir -p -m 0700 /home/rootless/.ssh && \
  cp -a /root/.ssh/id_rsa.pub /home/rootless/.ssh/authorized_keys && \
  mkdir -p /home/rootless/.local/share && \
  chown -R rootless:rootless /home/rootless
COPY ./Dockerfile.d/etc_systemd_system_user@.service.d_delegate.conf /etc/systemd/system/user@.service.d/delegate.conf
VOLUME /home/rootless/.local/share
COPY ./Dockerfile.d/test-integration-rootless.sh /
RUN chmod a+rx /test-integration-rootless.sh
CMD ["/test-integration-rootless.sh", "./hack/test-integration.sh"]

# test for CONTAINERD_ROOTLESS_ROOTLESSKIT_PORT_DRIVER=slirp4netns
FROM test-integration-rootless AS test-integration-rootless-port-slirp4netns
COPY ./Dockerfile.d/home_rootless_.config_systemd_user_containerd.service.d_port-slirp4netns.conf /home/rootless/.config/systemd/user/containerd.service.d/port-slirp4netns.conf
RUN chown -R rootless:rootless /home/rootless/.config
