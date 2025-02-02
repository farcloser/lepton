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
ARG CONTAINERD_VERSION=v2.0.2
ARG RUNC_VERSION=v1.2.4
ARG CNI_PLUGINS_VERSION=v1.6.2

# Extra deps: Build
ARG BUILDKIT_VERSION=v0.19.0
# Extra deps: Encryption
ARG IMGCRYPT_VERSION=v2.0.0
# Extra deps: Rootless
ARG ROOTLESSKIT_VERSION=v2.3.2
ARG SLIRP4NETNS_VERSION=v1.3.1
# Extra deps: bypass4netns
ARG BYPASS4NETNS_VERSION=v0.4.2
# Extra deps: FUSE-OverlayFS
ARG FUSE_OVERLAYFS_VERSION=v1.14
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION=v2.1.1
# Extra deps: Init
ARG TINI_VERSION=v0.19.0
# Extra deps: Debug
ARG BUILDG_VERSION=v0.4.1

# Test deps
ARG GO_VERSION=1.23
ARG UBUNTU_VERSION=24.04
ARG CONTAINERIZED_SYSTEMD_VERSION=v0.1.1
ARG GOTESTSUM_VERSION=v1.12.0
ARG SOCI_SNAPSHOTTER_VERSION=0.8.0

FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.6.1 AS xx


FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build-base-debian
COPY --from=xx / /
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -qq && apt-get install -qq --no-install-recommends \
    git \
    dpkg-dev
ARG TARGETARCH
# libbtrfs: for containerd
# libseccomp: for runc and bypass4netns
RUN xx-apt-get update -qq && xx-apt-get install -qq --no-install-recommends \
    binutils \
    gcc \
    libc6-dev \
    libbtrfs-dev \
    libseccomp-dev \
    pkg-config
RUN git config --global advice.detachedHead false

FROM build-base-debian AS build-containerd
ARG TARGETARCH
ARG CONTAINERD_VERSION
RUN git clone https://github.com/containerd/containerd.git /go/src/github.com/containerd/containerd
WORKDIR /go/src/github.com/containerd/containerd
RUN git checkout ${CONTAINERD_VERSION} && \
  mkdir -p /out /out/$TARGETARCH && \
  cp -a containerd.service /out
RUN GO=xx-go make STATIC=1 && \
  cp -a bin/containerd bin/containerd-shim-runc-v2 bin/ctr /out/$TARGETARCH

FROM build-base-debian AS build-runc
ARG RUNC_VERSION
ARG TARGETARCH
RUN git clone https://github.com/opencontainers/runc.git /go/src/github.com/opencontainers/runc
WORKDIR /go/src/github.com/opencontainers/runc
RUN git checkout ${RUNC_VERSION} && \
  mkdir -p /out
ENV CGO_ENABLED=1
RUN GO=xx-go CC=$(xx-info)-gcc STRIP=$(xx-info)-strip make static && \
  xx-verify --static runc && cp -v -a runc /out/runc.${TARGETARCH}

FROM build-base-debian AS build-bypass4netns
ARG BYPASS4NETNS_VERSION
ARG TARGETARCH
RUN git clone https://github.com/rootless-containers/bypass4netns.git /go/src/github.com/rootless-containers/bypass4netns
WORKDIR /go/src/github.com/rootless-containers/bypass4netns
RUN git checkout ${BYPASS4NETNS_VERSION} && \
  mkdir -p /out/${TARGETARCH}
ENV CGO_ENABLED=1
RUN GO=xx-go make static && \
  xx-verify --static bypass4netns && cp -a bypass4netns bypass4netnsd /out/${TARGETARCH}

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build-base
RUN apk add --no-cache make git curl
RUN git config --global advice.detachedHead false

FROM build-base AS build-minimal
RUN BINDIR=/out/bin make binaries install
# We do not set CMD to `go test` here, because it requires systemd

FROM build-base AS build-dependencies
ARG TARGETARCH
ARG BINARY_NAME
ENV GOARCH=${TARGETARCH}
COPY ./Dockerfile.d/SHA256SUMS.d/ /SHA256SUMS.d
WORKDIR /nowhere
RUN echo "${TARGETARCH:-amd64}" | sed -e s/amd64/x86_64/ -e s/arm64/aarch64/ | tee /target_uname_m
RUN mkdir -p /out/share/doc/${BINARY_NAME}-full && touch /out/share/doc/${BINARY_NAME}-full/README.md
ARG CONTAINERD_VERSION
COPY --from=build-containerd /out/${TARGETARCH:-amd64}/* /out/bin/
COPY --from=build-containerd /out/containerd.service /out/lib/systemd/system/containerd.service
RUN echo "- containerd: ${CONTAINERD_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG RUNC_VERSION
COPY --from=build-runc /out/runc.${TARGETARCH:-amd64} /out/bin/runc
RUN echo "- runc: ${RUNC_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG CNI_PLUGINS_VERSION
RUN fname="cni-plugins-${TARGETOS:-linux}-${TARGETARCH:-amd64}-${CNI_PLUGINS_VERSION}.tgz" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/containernetworking/plugins/releases/download/${CNI_PLUGINS_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/cni-plugins-${CNI_PLUGINS_VERSION}" | sha256sum -c && \
  mkdir -p /out/libexec/cni && \
  tar xzf "${fname}" -C /out/libexec/cni && \
  rm -f "${fname}" && \
  echo "- CNI plugins: ${CNI_PLUGINS_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG BUILDKIT_VERSION
RUN fname="buildkit-${BUILDKIT_VERSION}.${TARGETOS:-linux}-${TARGETARCH:-amd64}.tar.gz" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/moby/buildkit/releases/download/${BUILDKIT_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/buildkit-${BUILDKIT_VERSION}" | sha256sum -c && \
  tar xzf "${fname}" -C /out && \
  rm -f "${fname}" /out/bin/buildkit-qemu-* /out/bin/buildkit-cni-* /out/bin/buildkit-runc && \
  for f in /out/libexec/cni/*; do ln -s ../libexec/cni/$(basename $f) /out/bin/buildkit-cni-$(basename $f); done && \
  echo "- BuildKit: ${BUILDKIT_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
# NOTE: github.com/moby/buildkit/examples/systemd is not included in BuildKit v0.8.x, will be included in v0.9.x
RUN cd /out/lib/systemd/system && \
  sedcomm='s@bin/containerd@bin/buildkitd@g; s@(Description|Documentation)=.*@@' && \
  sed -E "${sedcomm}" containerd.service > buildkit.service && \
  echo "" >> buildkit.service && \
  echo "# This file was converted from containerd.service, with \`sed -E '${sedcomm}'\`" >> buildkit.service
ARG IMGCRYPT_VERSION
RUN git clone https://github.com/containerd/imgcrypt.git /go/src/github.com/containerd/imgcrypt && \
  cd /go/src/github.com/containerd/imgcrypt && \
  git checkout "${IMGCRYPT_VERSION}" && \
  CGO_ENABLED=0 make && DESTDIR=/out make install && \
  echo "- imgcrypt: ${IMGCRYPT_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG SLIRP4NETNS_VERSION
RUN fname="slirp4netns-$(cat /target_uname_m)" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/rootless-containers/slirp4netns/releases/download/${SLIRP4NETNS_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/slirp4netns-${SLIRP4NETNS_VERSION}" | sha256sum -c && \
  mv "${fname}" /out/bin/slirp4netns && \
  chmod +x /out/bin/slirp4netns && \
  echo "- slirp4netns: ${SLIRP4NETNS_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG BYPASS4NETNS_VERSION
COPY --from=build-bypass4netns /out/${TARGETARCH:-amd64}/* /out/bin/
RUN echo "- bypass4netns: ${BYPASS4NETNS_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG FUSE_OVERLAYFS_VERSION
RUN fname="fuse-overlayfs-$(cat /target_uname_m)" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/containers/fuse-overlayfs/releases/download/${FUSE_OVERLAYFS_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/fuse-overlayfs-${FUSE_OVERLAYFS_VERSION}" | sha256sum -c && \
  mv "${fname}" /out/bin/fuse-overlayfs && \
  chmod +x /out/bin/fuse-overlayfs && \
  echo "- fuse-overlayfs: ${FUSE_OVERLAYFS_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG CONTAINERD_FUSE_OVERLAYFS_VERSION
RUN fname="containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION/v}-${TARGETOS:-linux}-${TARGETARCH:-amd64}.tar.gz" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/containerd/fuse-overlayfs-snapshotter/releases/download/${CONTAINERD_FUSE_OVERLAYFS_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/containerd-fuse-overlayfs-${CONTAINERD_FUSE_OVERLAYFS_VERSION}" | sha256sum -c && \
  tar xzf "${fname}" -C /out/bin && \
  rm -f "${fname}" && \
  echo "- containerd-fuse-overlayfs: ${CONTAINERD_FUSE_OVERLAYFS_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG TINI_VERSION
RUN fname="tini-static-${TARGETARCH:-amd64}" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/krallin/tini/releases/download/${TINI_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/tini-${TINI_VERSION}" | sha256sum -c && \
  cp -a "${fname}" /out/bin/tini && chmod +x /out/bin/tini && \
  echo "- Tini: ${TINI_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG BUILDG_VERSION
RUN fname="buildg-${BUILDG_VERSION}-${TARGETOS:-linux}-${TARGETARCH:-amd64}.tar.gz" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/ktock/buildg/releases/download/${BUILDG_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/buildg-${BUILDG_VERSION}" | sha256sum -c && \
  tar xzf "${fname}" -C /out/bin && \
  rm -f "${fname}" && \
  echo "- buildg: ${BUILDG_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md
ARG ROOTLESSKIT_VERSION
RUN fname="rootlesskit-$(cat /target_uname_m).tar.gz" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/rootless-containers/rootlesskit/releases/download/${ROOTLESSKIT_VERSION}/${fname}" && \
  grep "${fname}" "/SHA256SUMS.d/rootlesskit-${ROOTLESSKIT_VERSION}" | sha256sum -c && \
  tar xzf "${fname}" -C /out/bin && \
  rm -f "${fname}" /out/bin/rootlesskit-docker-proxy && \
  echo "- RootlessKit: ${ROOTLESSKIT_VERSION}" >> /out/share/doc/${BINARY_NAME}-full/README.md

RUN echo "" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  echo "## License" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  echo "- bin/slirp4netns:    [GNU GENERAL PUBLIC LICENSE, Version 2](https://github.com/rootless-containers/slirp4netns/blob/${SLIRP4NETNS_VERSION}/COPYING)" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  echo "- bin/fuse-overlayfs: [GNU GENERAL PUBLIC LICENSE, Version 2](https://github.com/containers/fuse-overlayfs/blob/${FUSE_OVERLAYFS_VERSION}/COPYING)" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  echo "- bin/{runc,bypass4netns,bypass4netnsd}: Apache License 2.0, statically linked with libseccomp ([LGPL 2.1](https://github.com/seccomp/libseccomp/blob/main/LICENSE), source code available at https://github.com/seccomp/libseccomp/)" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  echo "- bin/tini: [MIT License](https://github.com/krallin/tini/blob/${TINI_VERSION}/LICENSE)" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  echo "- Other files: [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)" >> /out/share/doc/${BINARY_NAME}-full/README.md && \
  (cd /out && find ! -type d | sort | xargs sha256sum > /tmp/SHA256SUMS ) && \
  mv /tmp/SHA256SUMS /out/share/doc/${BINARY_NAME}-full/SHA256SUMS && \
  chown -R 0:0 /out

FROM build-dependencies AS build-full
ARG BINARY_NAME
COPY . /go/src/go.farcloser.world/lepton
RUN { echo "# ${BINARY_NAME} (full distribution)"; echo "- ${BINARY_NAME}: $(cd /go/src/go.farcloser.world/lepton && git describe --tags)"; cat /out/share/doc/${BINARY_NAME}-full/README.md; } > /out/share/doc/${BINARY_NAME}-full/README.md.new; mv /out/share/doc/${BINARY_NAME}-full/README.md.new /out/share/doc/${BINARY_NAME}-full/README.md
WORKDIR /go/src/go.farcloser.world/lepton
RUN BINDIR=/out/bin make binaries install
COPY README.md /out/share/doc/${BINARY_NAME}/
COPY docs /out/share/doc/${BINARY_NAME}/docs

FROM scratch AS out-full
COPY --from=build-full /out /

FROM ubuntu:${UBUNTU_VERSION} AS base
ARG BINARY_NAME
RUN apt-get update -qq && apt-get install -qq -y --no-install-recommends \
    apparmor \
    bash-completion \
    ca-certificates curl \
    iproute2 iptables \
    dbus dbus-user-session systemd systemd-sysv
ARG CONTAINERIZED_SYSTEMD_VERSION
RUN curl -o /docker-entrypoint.sh -fsSL --proto '=https' --tlsv1.2 https://raw.githubusercontent.com/AkihiroSuda/containerized-systemd/${CONTAINERIZED_SYSTEMD_VERSION}/docker-entrypoint.sh && \
  chmod +x /docker-entrypoint.sh
COPY --from=out-full / /usr/local/
RUN perl -pi -e 's/multi-user.target/docker-entrypoint.target/g' /usr/local/lib/systemd/system/*.service && \
  systemctl enable containerd buildkit && \
  mkdir -p /etc/bash_completion.d && \
  ${BINARY_NAME} completion bash >/etc/bash_completion.d/${BINARY_NAME} && \
  mkdir -p -m 0755 /etc/cni
COPY ./Dockerfile.d/etc_containerd_config.toml /etc/containerd/config.toml
COPY ./Dockerfile.d/etc_buildkit_buildkitd.toml /etc/buildkit/buildkitd.toml
VOLUME /var/lib/containerd
VOLUME /var/lib/buildkit
VOLUME /var/lib/${BINARY_NAME}
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["bash", "--login", "-i"]

# convert GO_VERSION=1.16 to the latest release such as "go1.16.1"
FROM golang:${GO_VERSION}-alpine AS goversion
RUN go env GOVERSION > /GOVERSION

FROM base AS test-integration
ARG DEBIAN_FRONTEND=noninteractive
# `expect` package contains `unbuffer(1)`, which is used for emulating TTY for testing
RUN apt-get update -qq && apt-get install -qq --no-install-recommends \
    expect \
    git \
    make
COPY --from=goversion /GOVERSION /GOVERSION
ARG TARGETARCH
RUN curl -fsSL --proto '=https' --tlsv1.2 https://golang.org/dl/$(cat /GOVERSION).linux-${TARGETARCH:-amd64}.tar.gz | tar xzvC /usr/local
ENV PATH=/usr/local/go/bin:$PATH
ARG GOTESTSUM_VERSION
RUN GOBIN=/usr/local/bin go install gotest.tools/gotestsum@${GOTESTSUM_VERSION}
COPY . /go/src/go.farcloser.world/lepton
WORKDIR /go/src/go.farcloser.world/lepton
VOLUME /tmp
ENV CGO_ENABLED=0
# copy cosign binary for integration test
COPY --from=ghcr.io/sigstore/cosign/cosign:v2.2.3@sha256:8fc9cad121611e8479f65f79f2e5bea58949e8a87ffac2a42cb99cf0ff079ba7 /ko-app/cosign /usr/local/bin/cosign
# installing soci for integration test
ARG SOCI_SNAPSHOTTER_VERSION
RUN fname="soci-snapshotter-${SOCI_SNAPSHOTTER_VERSION}-${TARGETOS:-linux}-${TARGETARCH:-amd64}.tar.gz" && \
  curl -o "${fname}" -fsSL --proto '=https' --tlsv1.2 "https://github.com/awslabs/soci-snapshotter/releases/download/v${SOCI_SNAPSHOTTER_VERSION}/${fname}" && \
  tar -C /usr/local/bin -xvf "${fname}" soci soci-snapshotter-grpc
COPY ./Dockerfile.d/test-integration-buildkit-test.service /usr/local/lib/systemd/system/
COPY ./Dockerfile.d/test-integration-soci-snapshotter.service /usr/local/lib/systemd/system/
RUN cp /usr/local/bin/tini /usr/local/bin/tini-custom
# using test integration containerd config
COPY ./Dockerfile.d/test-integration-etc_containerd_config.toml /etc/containerd/config.toml
RUN systemctl enable test-integration-buildkit-test test-integration-soci-snapshotter
CMD ["./hack/test-integration.sh"]

FROM test-integration AS test-integration-rootless
# Install SSH for creating systemd user session.
# (`sudo` does not work for this purpose,
#  OTOH `machinectl shell` can create the session but does not propagate exit code)
RUN apt-get update -qq && apt-get install -qq --no-install-recommends \
    uidmap \
    openssh-server \
    openssh-client
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

FROM base AS demo
