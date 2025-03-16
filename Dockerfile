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

ARG         DEBIAN_VERSION=bookworm-slim
ARG         UBUNTU_VERSION=24.04
ARG         GO_VERSION=1.24.0

ARG         LICENSE_APACHE_V2="Apache License, version 2.0, https://opensource.org/license/apache-2-0"
ARG         LICENSE_MIT="The MIT License, https://opensource.org/license/mit"
ARG         LICENSE_3CLAUSES_BSD="The 3-Clause BSD License, https://opensource.org/license/bsd-3-clause"
ARG         LICENSE_GPL_V2="GNU General Public License version 2, https://opensource.org/license/gpl-2-0"
ARG         LICENSE_LGPL_V21="GNU Lesser General Public License version 2.1, https://opensource.org/license/lgpl-2-1"
ARG         LICENSE_ZLIB="The zlib/libpng License, https://opensource.org/license/zlib"

ARG         BINARY_NAME=lepton
ARG         BINARY_LICENSE="$LICENSE_APACHE_V2"

ARG         CONTAINERD_VERSION=v2.0.3
ARG         CONTAINERD_REVISION=06b99ca80cdbfbc6cc8bd567021738c9af2b36ce
ARG         CONTAINERD_LICENSE="$LICENSE_APACHE_V2"
ARG         CONTAINERD_REPO=github.com/containerd/containerd

ARG         RUNC_VERSION=v1.2.5
ARG         RUNC_REVISION=59923ef18c98053ddb1acf23ecba10344056c28e
ARG         RUNC_LICENSE="$LICENSE_APACHE_V2"
ARG         RUNC_REPO=github.com/opencontainers/runc

ARG         CNI_PLUGINS_VERSION=v1.6.2
ARG         CNI_PLUGINS_REVISION=7f756b411efc3d3730c707e2cc1f2baf1a66e28c
ARG         CNI_PLUGINS_LICENSE="$LICENSE_APACHE_V2"
ARG         CNI_PLUGINS_REPO=github.com/containernetworking/plugins

ARG         BUILDKIT_VERSION=v0.20.1
ARG         BUILDKIT_REVISION=de56a3c5056341667b5bad71f414ece70b50724f
ARG         BUILDKIT_LICENSE="$LICENSE_APACHE_V2"
ARG         BUILDKIT_REPO=github.com/moby/buildkit

ARG         IMGCRYPT_VERSION=v2.0.0
ARG         IMGCRYPT_REVISION=1e301ef2620964bedfa68ee4b841ff80f4887736
ARG         IMGCRYPT_LICENSE="$LICENSE_APACHE_V2"
ARG         IMGCRYPT_REPO=github.com/containerd/imgcrypt

ARG         COSIGN_VERSION=v2.4.3
ARG         COSIGN_REVISION=6a7abbf3ae7eb6949883a80c8f6007cc065d2dfb
ARG         COSIGN_LICENSE="$LICENSE_APACHE_V2"
ARG         COSIGN_REPO=github.com/sigstore/cosign

ARG         ROOTLESSKIT_VERSION=v2.3.4
ARG         ROOTLESSKIT_REVISION=59a459df858d39ad5f4eafa305545907bf0c48ab
ARG         ROOTLESSKIT_LICENSE="$LICENSE_APACHE_V2"
ARG         ROOTLESSKIT_REPO=github.com/rootless-containers/rootlesskit

ARG         LIBSLIRP_VERSION=v4.9.0
ARG         LIBSLIRP_REVISION=c32a8a1ccaae8490142e67e078336a95c5ffc956
ARG         LIBSLIRP_LICENSE="$LICENSE_3CLAUSES_BSD"
# Maintenance for a week, March 2025
# ARG         LIBSLIRP_REPO=gitlab.freedesktop.org/slirp/libslirp
ARG         LIBSLIRP_REPO=gitlab.com/qemu-project/libslirp

ARG         SLIRP4NETNS_VERSION=v1.3.2
ARG         SLIRP4NETNS_REVISION=0f13345bcef588d2bb70d662d41e92ee8a816d85
ARG         SLIRP4NETNS_LICENSE="$LICENSE_GPL_V2"
ARG         SLIRP4NETNS_REPO=github.com/rootless-containers/slirp4netns

ARG         BYPASS4NETNS_VERSION=v0.4.2
ARG         BYPASS4NETNS_REVISION=aa04bd3dcc48c6dae6d7327ba219bda8fe2a4634
ARG         BYPASS4NETNS_LICENSE="$LICENSE_APACHE_V2"
ARG         BYPASS4NETNS_REPO=github.com/rootless-containers/bypass4netns

ARG         BUILDG_VERSION=v0.4.1
ARG         BUILDG_REVISION=8dd12a26f4ab05ad20f3fe9811fb42aff6bf472a
ARG         BUILDG_LICENSE="$LICENSE_APACHE_V2"
ARG         BUILDG_REPO=github.com/ktock/buildg

ARG         SOCI_SNAPSHOTTER_VERSION=v0.9.0
ARG         SOCI_SNAPSHOTTER_REVISION=737f61a3db40c386f997c1f126344158aa3ad43c
ARG         SOCI_SNAPSHOTTER_LICENSE="$LICENSE_APACHE_V2"
ARG         SOCI_SNAPSHOTTER_REPO=github.com/awslabs/soci-snapshotter

ARG         TINI_VERSION=v0.19.0
ARG         TINI_REVISION=de40ad007797e0dcd8b7126f27bb87401d224240
ARG         TINI_LICENSE="$LICENSE_MIT"
ARG         TINI_REPO=github.com/krallin/tini

ARG         SECCOMP_LICENSE="$LICENSE_LGPL_V21"
ARG         ZLIB_LICENSE="$LICENSE_ZLIB"
ARG         GLIB_LICENSE="$LICENSE_APACHE_V2"
ARG         LIBCAP_LICENSE="$LICENSE_3CLAUSES_BSD"

ARG         NO_COLOR=true
ARG         DEBIAN_IMAGE=ghcr.io/apostasie/debian
ARG         UBUNTU_IMAGE=ghcr.io/apostasie/ubuntu

########################################################################################################################
# Base images
# These stages are purely tooling that other stages can leverage.
# They are highly cacheable, and will only change if one of the following is changing:
# - DEBIAN_VERSION (=bookworm-slim) or content of it
# - GO_VERSION
########################################################################################################################

#           tooling-base is the single base image we use for all other tooling image
#           Note: technically, we should rm -rf /var/lib/apt/lists/* - however that means forcing apt-get update everytime
#           The cost is about 20MB on a single arch.
FROM        --platform=$BUILDPLATFORM $DEBIAN_IMAGE:$DEBIAN_VERSION AS tooling-base
SHELL       ["/bin/bash", "-o", "errexit", "-o", "errtrace", "-o", "functrace", "-o", "nounset", "-o", "pipefail", "-c"]
ENV         DEBIAN_FRONTEND="noninteractive"
ENV         TERM="xterm"
ENV         LANG="C.UTF-8"
ENV         LC_ALL="C.UTF-8"
ENV         TZ="America/Los_Angeles"
RUN         echo "force-unsafe-io" > /etc/dpkg/dpkg.cfg.d/farcloser-speedup && \
            echo 'Acquire::Languages "none";' > /etc/apt/apt.conf.d/farcloser-no-language && \
            echo 'Acquire::GzipIndexes "true";' > /etc/apt/apt.conf.d/farcloser-gzip-indexes && \
            apt-get update -qq >/dev/null && \
            apt-get install -qq --no-install-recommends \
                ca-certificates \
                    >/dev/null

#           tooling-downloader-golang will download a golang archive and expand it into /out/usr/local
#           You may set GO_VERSION to an explicit, complete version (eg: 1.23.0), or you can also set it to:
#           - canary: that will retrieve the latest alpha/beta/RC
#           - stable (or ""): that will retrieve the latest stable version
#           Note that for these last two, you need to bust the cache for this stage if you expect a refresh
#           Finally note that we are retrieving both architectures we are currently supporting (arm64 and amd64) in one stage,
#           and do NOT leverage TARGETARCH, as that would force cross compilation to use a non-native binary in dependent stages.
FROM        --platform=$BUILDPLATFORM tooling-base AS tooling-downloader-golang
ARG         GO_VERSION
RUN         apt-get install -qq --no-install-recommends \
               curl \
               jq \
                   >/dev/null; \
            os=linux; \
            all_versions="$(curl -fsSL --proto '=https' --tlsv1.3 "https://go.dev/dl/?mode=json&include=all")"; \
            candidates="$(case "$GO_VERSION" in \
                    canary) condition=".stable==false" ;; \
                    stable|"") condition=".stable==true" ;; \
                    *) condition='.version | startswith("go'"$GO_VERSION"'")' ;; \
                esac; \
                jq -rc 'map(select('"$condition"'))[0].files | map(select(.os=="'"$os"'"))' <(printf "$all_versions"))"; \
            arch=arm64; \
            filename="$(jq -r 'map(select(.arch=="'"$arch"'"))[0].filename' <(printf "$candidates"))"; \
            mkdir -p /out/usr/local/linux/"$arch"; \
            curl -fsSL --proto '=https' --tlsv1.3 https://go.dev/dl/"$filename" | tar xzC /out/usr/local/linux/"$arch" || {  \
                echo "Failed retrieving go download for $arch: $GO_VERSION"; \
                exit 1; \
            }; \
            arch=amd64; \
            filename="$(jq -r 'map(select(.arch=="'"$arch"'"))[0].filename' <(printf "$candidates"))"; \
            mkdir -p /out/usr/local/linux/"$arch"; \
            curl -fsSL --proto '=https' --tlsv1.3 https://go.dev/dl/"$filename" | tar xzC /out/usr/local/linux/"$arch" || {  \
                echo "Failed retrieving go download for $arch: $GO_VERSION"; \
                exit 1; \
            }; \
            apt-get purge -qq curl jq

#           tooling-builder is a go enabled stage with minimal build tooling installed that can be used to build non-cgo projects
FROM        --platform=$BUILDPLATFORM tooling-base AS tooling-builder
ARG         BUILDPLATFORM
WORKDIR     /src
RUN         mkdir -p /out/bin; mkdir -p /metadata && \
            apt-get install -qq --no-install-recommends \
                git \
                make \
                    >/dev/null && \
            git config --global advice.detachedHead false # Prevent git from complaining on detached head
#           Configure base environment
ENV         CGO_ENABLED=0
ENV         GOFIPS140=v1.0.0
ENV         GOTOOLCHAIN=local
#           Add golang
COPY        --from=tooling-downloader-golang /out/usr/local/$BUILDPLATFORM /usr/local
ENV         PATH="/root/go/bin:/usr/local/go/bin:$PATH"
ARG         NO_COLOR
ENV         GOFLAGS="-trimpath"

# tooling-builder-with-c-dependencies is an expansion of the previous stages that adds extra c dependencies.
# It is meant for (cross-compilation of) c and cgo projects.
FROM        --platform=$BUILDPLATFORM tooling-builder AS tooling-builder-with-c-dependencies-base
# Enable CGO
ENV         CGO_ENABLED=1
## https://gcc.gnu.org/onlinedocs/gcc/Warning-Options.html
ENV         WARNING_OPTIONS="-Wall -Werror=format-security"
## https://gcc.gnu.org/onlinedocs/gcc/Optimize-Options.html#Optimize-Options
ENV         OPTIMIZATION_OPTIONS="-O2"
## https://gcc.gnu.org/onlinedocs/gcc/Debugging-Options.html#Debugging-Options
ENV         DEBUGGING_OPTIONS="-grecord-gcc-switches -g"
## https://gcc.gnu.org/onlinedocs/gcc/Preprocessor-Options.html#Preprocessor-Options
# https://www.gnu.org/software/libc/manual/html_node/Source-Fortification.html
ENV         SECURITY_OPTIONS="-fstack-protector-strong -fstack-clash-protection -fPIE -D_FORTIFY_SOURCE=2 -D_GLIBCXX_ASSERTIONS"
## Control flow integrity is amd64 only
# -mcet -fcf-protection
## https://gcc.gnu.org/onlinedocs/gcc/Link-Options.html#Link-Options
ENV         LDFLAGS_COMMON="-Wl,-z,relro -Wl,-z,now -Wl,-z,defs -Wl,-z,noexecstack"
ENV         LDFLAGS_NOPIE="$LDFLAGS_COMMON -static"
ENV         LDFLAGS_PIE="$LDFLAGS_COMMON -pie -static-pie"
ENV         LDFLAGS="$LDFLAGS_PIE"
# -s strips symbol and relocation info
# -pipe gives a little speed-up by using pipes instead of temp files
ENV         CFLAGS="$WARNING_OPTIONS $OPTIMIZATION_OPTIONS $DEBUGGING_OPTIONS $SECURITY_OPTIONS -s -pipe"
ENV         CPPFLAGS="-D_FORTIFY_SOURCE=2 -D_GLIBCXX_ASSERTIONS"
ENV         CXXFLAGS="$WARNING_OPTIONS $OPTIMIZATION_OPTIONS $DEBUGGING_OPTIONS $SECURITY_OPTIONS -s -pipe"
ENV         CGO_CFLAGS="$CFLAGS"
ENV         CGO_CPPFLAGS="$CPPFLAGS"
ENV         CGO_CXXFLAGS="$CXXFLAGS"
ENV         CGO_CPPFLAGS="$CPPFLAGS"
ENV         CGO_LDFLAGS="$LDFLAGS"
# More reading:
## https://news.ycombinator.com/item?id=18874113
## https://developers.redhat.com/blog/2018/03/21/compiler-and-linker-flags-gcc
## https://gcc.gnu.org/onlinedocs/gcc/Instrumentation-Options.html
# https://github.com/golang/go/issues/26849
ENV         GOFLAGS_COMMON="$GOFLAGS -ldflags=-linkmode=external -tags=cgo,netgo,osusergo,static_build"
ENV         GOFLAGS_NOPIE="$GOFLAGS_COMMON"
ENV         GOFLAGS_PIE="$GOFLAGS_COMMON -buildmode=pie"
# Set default linker options for CGO projects
ENV         GOFLAGS="$GOFLAGS_PIE"
# TODO: -s -w - extldflags -static-pie and -static?
# -gcflags=all="-N -l"?
# CGO_LDFLAGS=-fuse-ld=lld?

# cross-build-essential brings in gcc, g++ (along with binutils) and dpkg-cross
# pkg-config: go libseccomp
# cmake: tini
# meson: libslirp
# automake: slirp4netns
# libseccomp: runc, bypass4netns, slirp4netns
RUN         apt-get install -qq --no-install-recommends \
                cmake \
                meson \
                automake \
                    >/dev/null; \
            for architecture in arm64 amd64; do \
                dpkg --add-architecture "$architecture"; \
            done; \
            apt-get update -qq >/dev/null; \
            for architecture in amd64 arm64; do \
                debarch="$(sed -e 's/arm64/aarch64/' -e 's/amd64/x86-64/' <<<"$architecture")"; \
                apt-get install -qq \
                    crossbuild-essential-"$architecture" \
                    pkg-config:"$architecture" \
                    libc6-dev:"$architecture" \
                    libseccomp-dev:"$architecture" \
                        >/dev/null; \
            done

FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies-base AS tooling-builder-with-c-dependencies
ARG         TARGETARCH
RUN         arch="$(sed -e 's/arm64/aarch64/' -e 's/amd64/x86_64/' <<<"$TARGETARCH")"; \
            echo "export PKG_CONFIG=${arch}-linux-gnu-pkg-config" >> /.env; \
            echo "export AR=${arch}-linux-gnu-ar" >> /.env; \
            echo "export CC=${arch}-linux-gnu-gcc" >> /.env; \
            echo "export CXX=${arch}-linux-gnu-g++" >> /.env; \
            echo "export STRIP=${arch}-linux-gnu-strip" >> /.env

########################################################################################################################
# Downloading sources
# These stages are downloading all projects, using the base tooling stage from above
# Note:
# - clone are restricted to the exact tag and depth 1 to reduce overall clone traffic
# - clones (and vendorization) are in separate single stages so that we do that only ONCE when targetting multiple archs
# - clones checkouts are then mounted into the build stage to avoid creating useless copy layers
# - mod downloads are using a shared cache when vendorizing (if necessary) so that dependencies shared accross projects
# are not retrieved too many times
########################################################################################################################

########################################################################################################################
# containerd (vendored)
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-containerd
ARG         CONTAINERD_VERSION
ARG         CONTAINERD_REVISION
ARG         CONTAINERD_LICENSE
ARG         CONTAINERD_REPO
ARG         _REPO=$CONTAINERD_REPO
ARG         _VERSION=$CONTAINERD_VERSION
ARG         _REVISION=$CONTAINERD_REVISION
ARG         _LICENSE=$CONTAINERD_LICENSE
RUN         echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# runc (vendored)
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-runc
ARG         RUNC_VERSION
ARG         RUNC_REVISION
ARG         RUNC_LICENSE
ARG         RUNC_REPO
ARG         _REPO=$RUNC_REPO
ARG         _VERSION=$RUNC_VERSION
ARG         _REVISION=$RUNC_REVISION
ARG         _LICENSE=$RUNC_LICENSE
RUN         echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# buildkit (vendored)
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-buildkit
ARG         BUILDKIT_VERSION
ARG         BUILDKIT_REVISION
ARG         BUILDKIT_LICENSE
ARG         BUILDKIT_REPO
ARG         _REPO=$BUILDKIT_REPO
ARG         _VERSION=$BUILDKIT_VERSION
ARG         _REVISION=$BUILDKIT_REVISION
ARG         _LICENSE=$BUILDKIT_LICENSE
RUN         echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# cni-plugins (vendored)
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-cni
ARG         CNI_PLUGINS_VERSION
ARG         CNI_PLUGINS_REVISION
ARG         CNI_PLUGINS_LICENSE
ARG         CNI_PLUGINS_REPO
ARG         _REPO=$CNI_PLUGINS_REPO
ARG         _VERSION=$CNI_PLUGINS_VERSION
ARG         _REVISION=$CNI_PLUGINS_REVISION
ARG         _LICENSE=$CNI_PLUGINS_LICENSE
RUN         echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# bypass4netns
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-bypass4netns
ARG         BYPASS4NETNS_VERSION
ARG         BYPASS4NETNS_REVISION
ARG         BYPASS4NETNS_LICENSE
ARG         BYPASS4NETNS_REPO
ARG         _REPO=$BYPASS4NETNS_REPO
ARG         _VERSION=$BYPASS4NETNS_VERSION
ARG         _REVISION=$BYPASS4NETNS_REVISION
ARG         _LICENSE=$BYPASS4NETNS_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            go mod vendor && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# imgcrypt
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-imgcrypt
ARG         IMGCRYPT_VERSION
ARG         IMGCRYPT_REVISION
ARG         IMGCRYPT_LICENSE
ARG         IMGCRYPT_REPO
ARG         _REPO=$IMGCRYPT_REPO
ARG         _VERSION=$IMGCRYPT_VERSION
ARG         _REVISION=$IMGCRYPT_REVISION
ARG         _LICENSE=$IMGCRYPT_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            go mod vendor && cd cmd && go mod vendor && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# buildg
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-buildg
ARG         BUILDG_VERSION
ARG         BUILDG_REVISION
ARG         BUILDG_LICENSE
ARG         BUILDG_REPO
ARG         _REPO=$BUILDG_REPO
ARG         _VERSION=$BUILDG_VERSION
ARG         _REVISION=$BUILDG_REVISION
ARG         _LICENSE=$BUILDG_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            go mod vendor && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# rootlesskit
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-rootlesskit
ARG         ROOTLESSKIT_VERSION
ARG         ROOTLESSKIT_REVISION
ARG         ROOTLESSKIT_LICENSE
ARG         ROOTLESSKIT_REPO
ARG         _REPO=$ROOTLESSKIT_REPO
ARG         _VERSION=$ROOTLESSKIT_VERSION
ARG         _REVISION=$ROOTLESSKIT_REVISION
ARG         _LICENSE=$ROOTLESSKIT_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            go mod vendor && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# cosign
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-cosign
ARG         COSIGN_VERSION
ARG         COSIGN_REVISION
ARG         COSIGN_LICENSE
ARG         COSIGN_REPO
ARG         _REPO=$COSIGN_REPO
ARG         _VERSION=$COSIGN_VERSION
ARG         _REVISION=$COSIGN_REVISION
ARG         _LICENSE=$COSIGN_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            go mod vendor && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# soci
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-soci
ARG         SOCI_SNAPSHOTTER_VERSION
ARG         SOCI_SNAPSHOTTER_REVISION
ARG         SOCI_SNAPSHOTTER_LICENSE
ARG         SOCI_SNAPSHOTTER_REPO
ARG         _REPO=$SOCI_SNAPSHOTTER_REPO
ARG         _VERSION=$SOCI_SNAPSHOTTER_VERSION
ARG         _REVISION=$SOCI_SNAPSHOTTER_REVISION
ARG         _LICENSE=$SOCI_SNAPSHOTTER_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            go mod vendor && cd cmd && go mod vendor && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# tini
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-tini
ARG         TINI_VERSION
ARG         TINI_REVISION
ARG         TINI_LICENSE
ARG         TINI_REPO
ARG         _REPO=$TINI_REPO
ARG         _VERSION=$TINI_VERSION
ARG         _REVISION=$TINI_REVISION
ARG         _LICENSE=$TINI_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# libslirp
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-libslirp
ARG         LIBSLIRP_VERSION
ARG         LIBSLIRP_REVISION
ARG         LIBSLIRP_LICENSE
ARG         LIBSLIRP_REPO
ARG         _REPO=$LIBSLIRP_REPO
ARG         _VERSION=$LIBSLIRP_VERSION
ARG         _REVISION=$LIBSLIRP_REVISION
ARG         _LICENSE=$LIBSLIRP_LICENSE
RUN         echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# slirp4netns
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-slirp4netns
ARG         SLIRP4NETNS_VERSION
ARG         SLIRP4NETNS_REVISION
ARG         SLIRP4NETNS_LICENSE
ARG         SLIRP4NETNS_REPO
ARG         _REPO=$SLIRP4NETNS_REPO
ARG         _VERSION=$SLIRP4NETNS_VERSION
ARG         _REVISION=$SLIRP4NETNS_REVISION
ARG         _LICENSE=$SLIRP4NETNS_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            echo "$_VERSION" > /metadata/VERSION && echo "$_REVISION" > /metadata/REVISION && echo "$_LICENSE" > /metadata/LICENSE && \
            git clone --quiet --depth 1 --branch "$_VERSION" https://"$_REPO".git . && \
            [ "$_REVISION" == "$(git rev-parse HEAD)" ] || { echo "ERROR: commit hash $(git rev-parse HEAD) does not match expectations $_REVISION"; exit 42; }

########################################################################################################################
# cli binary is built from the local context
########################################################################################################################
# FIXME: leptonic is temporary
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-download-cli
ARG         BINARY_NAME
ARG         BINARY_LICENSE
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            --mount=target=go.mod,source=go.mod,type=bind \
            --mount=target=go.sum,source=go.sum,type=bind \
            --mount=target=pkg,source=pkg,type=bind \
            --mount=target=cmd,source=cmd,type=bind \
            --mount=target=leptonic,source=leptonic,type=bind \
            go mod vendor
COPY        docs /out/share/doc/"$BINARY_NAME"/docs
#           CAREFUL: this will fail to retrieve a tag with a shallow clone. So, full-release should better be done
#           with a full history clone if version is expected to be sensical.
RUN         --mount=target=/src,type=bind \
            { printf "%s" "$(git rev-parse HEAD)"; if ! git diff --no-ext-diff --quiet --exit-code; then printf .m; fi; } > /metadata/REVISION && \
            { git describe --tags --match 'v[0-9]*' --dirty='.m' --always 2>/dev/null || true; } > /metadata/VERSION && \
            echo "$BINARY_LICENSE" > /metadata/LICENSE

########################################################################################################################
# Building
# From the source above, source layers are mounted.
# Note:
# - we are systematically bypassing native Makefile and other ways to build as:
#   - most of them do not allow building out of tree (problematic for sharing the layer accross multiple arch)
#   - they all have different ways to pass additional arguments, and do not enforce the same compiler or linker parameters
########################################################################################################################

########################################################################################################################
# containerd shim and ctr
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-containerd-tools
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/containerd/containerd/v2
RUN         --mount=from=dependencies-download-containerd,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-containerd,type=bind,target=/metadata,source=/metadata \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION) -X $PKG/version.Package=$PKG" \
                -o /out/bin/ctr ./cmd/ctr && \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION) -X $PKG/version.Package=$PKG" \
                -o /out/bin/containerd-shim-runc-v2 ./cmd/containerd-shim-runc-v2

########################################################################################################################
# buildctl and buildkitd
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-buildkit
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/moby/buildkit
RUN         --mount=from=dependencies-download-buildkit,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-buildkit,type=bind,target=/metadata,source=/metadata \
            GOOS=linux GOARCH=$TARGETARCH go build -mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION) -X $PKG/version.Package=$PKG" \
                -o /out/bin/buildctl ./cmd/buildctl && \
            GOOS=linux GOARCH=$TARGETARCH go build -mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION) -X $PKG/version.Package=$PKG" \
                -o /out/bin/buildkitd ./cmd/buildkitd

########################################################################################################################
# bypass4netnsd
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-bypass4netnsd
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/rootless-containers/bypass4netns/pkg
RUN         --mount=from=dependencies-download-bypass4netns,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-bypass4netns,type=bind,target=/metadata,source=/metadata \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION)" \
                -o /out/bin/bypass4netnsd ./cmd/bypass4netnsd

########################################################################################################################
# imgcrypt
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-imgcrypt
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/containerd/containerd/v2
RUN         --mount=from=dependencies-download-imgcrypt,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-imgcrypt,type=bind,target=/metadata,source=/metadata \
            cd cmd && \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION)" \
                -o /out/bin/ctr-enc ./ctr && \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -o /out/bin/ctd-decoder ./ctd-decoder

########################################################################################################################
# buildg
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-buildg
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/ktock/buildg/pkg
RUN         --mount=from=dependencies-download-buildg,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-buildg,type=bind,target=/metadata,source=/metadata \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION)" \
                -o /out/bin/buildg .

########################################################################################################################
# rootlesskit
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-rootlesskit
ARG         TARGETARCH
ARG         GOPROXY=off
RUN         --mount=from=dependencies-download-rootlesskit,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-rootlesskit,type=bind,target=/metadata,source=/metadata \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -o /out/bin/rootlesskit ./cmd/rootlesskit

########################################################################################################################
# cni
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-cni
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/containernetworking/plugins/pkg/utils
RUN         --mount=from=dependencies-download-cni,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-cni,type=bind,target=/metadata,source=/metadata \
            mkdir -p /out/libexec/cni; \
            for d in plugins/meta/* plugins/main/* plugins/ipam/*; do \
                plugin="$(basename "$d")"; \
                [ "${plugin}" != "windows" ] || continue; \
                GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                    -ldflags "-X $PKG/buildversion.BuildVersion=$(cat /metadata/VERSION)" \
                    -o /out/libexec/cni/"$plugin" ./"$d"; \
                ln -s ../libexec/cni/"$plugin" /out/bin/buildkit-cni-"$plugin"; \
            done

########################################################################################################################
# cosign
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-cosign
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=sigs.k8s.io/release-utils
RUN         --mount=from=dependencies-download-cosign,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-cosign,type=bind,target=/metadata,source=/metadata \
            epoch="$(git log -1 --no-show-signature --pretty=%ct)"; format="+%Y-%m-%dT%H:%M:%SZ"; date="$(date -u -d "@$epoch" "$format")"; \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.gitVersion=$(cat /metadata/VERSION) \
                    -X $PKG/version.gitCommit=$(cat /metadata/REVISION) \
                    -X $PKG/version.gitTreeState=clean \
                    -X $PKG/version.buildDate=$date" \
                -o /out/bin/cosign ./cmd/cosign

########################################################################################################################
# CGO: bypass4netns
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-bypass4netns
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/rootless-containers/bypass4netns/pkg
RUN         --mount=from=dependencies-download-bypass4netns,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-bypass4netns,type=bind,target=/metadata,source=/metadata \
            . /.env; \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION)" \
                -o /out/bin/bypass4netns ./cmd/bypass4netns

########################################################################################################################
# CGO: runc
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-runc
ARG         TARGETARCH
ARG         GOPROXY=off
RUN         --mount=from=dependencies-download-runc,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-runc,type=bind,target=/metadata,source=/metadata \
            . /.env; \
            GOOS=linux GOARCH=$TARGETARCH go build -mod=vendor \
                -ldflags "-X main.gitCommit=$(cat /metadata/REVISION) -X main.version=$(cat /metadata/VERSION)" \
                -tags=urfave_cli_no_docs,seccomp,cgo,netgo,osusergo,static_build \
                -o /out/bin/runc .

########################################################################################################################
# CGO: containerd
########################################################################################################################
# FIXME: do we want rdt?
FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-containerd
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/containerd/containerd/v2
RUN         --mount=from=dependencies-download-containerd,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-containerd,type=bind,target=/metadata,source=/metadata \
            . /.env; \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION) -X $PKG/version.Package=$PKG" \
                -tags=no_btrfs,no_devmapper,no_zfs,seccomp,urfave_cli_no_docs,cgo,osusergo,netgo,static_build \
                -o /out/bin/containerd ./cmd/containerd && \
            cp -a containerd.service /; [ ! -e /out/bin/core ] || { go env; ls -lA /out/bin; exit 42; }
# FIXME: ^ remove core debug stance when confident this does not happen

########################################################################################################################
# CGO: soci
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-soci
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         PKG=github.com/awslabs/soci-snapshotter
RUN         apt-get install -qq --no-install-recommends \
                zlib1g-dev:"$TARGETARCH" \
                    >/dev/null
RUN         --mount=from=dependencies-download-soci,type=bind,target=/src,source=/src \
            --mount=from=dependencies-download-soci,type=bind,target=/metadata,source=/metadata \
            . /.env; \
            cd cmd && \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION)" \
                -o /out/bin/soci ./soci && \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/version.Revision=$(cat /metadata/REVISION)" \
                -o /out/bin/soci-snapshotter-grpc ./soci-snapshotter-grpc

########################################################################################################################
# CGO: cosign-pivkey-pkcs11key
# FIXME: currently failing to link against pcsclite
########################################################################################################################
#FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-cosign-pkcs
#ARG         TARGETARCH
#ARG         GOPROXY=off
#ARG         PKG=sigs.k8s.io/release-utils
#RUN         xx-apt-get install -qq --no-install-recommends libpcsclite-dev >/dev/null
#RUN         --mount=from=dependencies-download-cosign,type=bind,target=/src,source=/src \
#    --mount=from=dependencies-download-cosign,type=bind,target=/metadata,source=/metadata \
#  . /.env; \
#  epoch="$(git log -1 --no-show-signature --pretty=%ct)"; format="+%Y-%m-%dT%H:%M:%SZ"; date="$(date -u -d "@$epoch" "$format")"; \
#  GOOS=linux GOARCH=$TARGETARCH \
#    go build --mod=vendor \
#      -tags=pivkey,pkcs11key,cgo,osusergo,netgo,static_build \
#      -ldflags "-X $PKG/version.gitVersion=$(cat /metadata/VERSION) \
#                -X $PKG/version.gitCommit=$(cat /metadata/REVISION) \
#                -X $PKG/version.gitTreeState=clean -X $PKG/version.buildDate=$date" \
#      -o /out/bin/cosign-pivkey-pkcs11key ./cmd/cosign

########################################################################################################################
# C: tini
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-tini
ARG         TARGETARCH
RUN         --mount=from=dependencies-download-tini,type=bind,target=/src,source=/src,rw \
            --mount=from=dependencies-download-tini,type=bind,target=/metadata,source=/metadata \
            --mount=type=tmpfs,target=/build \
            . /.env; \
            exec 42>.lock; flock -x 42; \
            cd /build && cmake /src && make tini; mv tini /out/bin; \
            flock -u 42

########################################################################################################################
# C: libslirp & slirp4netns
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder-with-c-dependencies AS dependencies-build-slirp4netns
ARG         TARGETARCH
RUN         apt-get install -qq --no-install-recommends \
                libglib2.0-dev:$TARGETARCH \
                libcap-dev:$TARGETARCH \
                    >/dev/null
RUN         --mount=from=dependencies-download-slirp4netns,type=bind,target=/src_slirp,source=/src,rw \
            --mount=from=dependencies-download-slirp4netns,type=bind,target=/metadata,source=/metadata \
            --mount=from=dependencies-download-libslirp,type=bind,target=/src,source=/src \
            --mount=type=tmpfs,target=/build \
            . /.env; \
            # Note: libslirp script won't install unless building both dyn and static versions of the lib
            LDFLAGS="$LDFLAGS_COMMON"; \
            meson setup --default-library=both /build && ninja -C /build install; \
            LDFLAGS="$LDFLAGS_PIE"; \
            cd /src_slirp; \
            exec 42>.lock; flock -x 42; \
            ./autogen.sh; ./configure; make; mv slirp4netns /out/bin; \
            flock -u 42

########################################################################################################################
# cli
########################################################################################################################
FROM        --platform=$BUILDPLATFORM tooling-builder AS dependencies-build-cli
ARG         TARGETARCH
ARG         GOPROXY=off
ARG         BINARY_NAME
ARG         PKG=go.farcloser.world/lepton/pkg
RUN         --mount=from=dependencies-download-cli,type=bind,target=/metadata,source=/metadata \
            --mount=from=dependencies-download-cli,type=bind,target=vendor,source=/src/vendor \
            --mount=target=go.mod,source=go.mod,type=bind \
            --mount=target=go.sum,source=go.sum,type=bind \
            --mount=target=pkg,source=pkg,type=bind \
            --mount=target=cmd,source=cmd,type=bind \
            --mount=target=leptonic,source=leptonic,type=bind \
            --mount=target=extras,source=extras,type=bind \
            GOOS=linux GOARCH=$TARGETARCH go build --mod=vendor \
                -ldflags "-X $PKG/version.Version=$(cat /metadata/VERSION) -X $PKG/pkg/version.Revision=$(cat /metadata/REVISION)" \
                -o /out/bin/$BINARY_NAME ./cmd/$BINARY_NAME

########################################################################################################################
# Assembly
# These stages are here to assemble all build and download dependencies together for various purposes:
# - full-release distribution
# - test-integration images
# - demo image
########################################################################################################################
# assembly-release-assets is single platform, and prepares the non-architecture dependent files for the full release
FROM        --platform=$BUILDPLATFORM tooling-builder AS assembly-release-assets
ARG         BINARY_NAME
ARG         SECCOMP_LICENSE
ARG         ZLIB_LICENSE
ARG         GLIB_LICENSE
ARG         LIBCAP_LICENSE
RUN         mkdir -p /out/lib/systemd/system /out/share/doc/"$BINARY_NAME"-full
COPY        --from=dependencies-build-containerd /containerd.service /out/lib/systemd/system/containerd.service
# NOTE: github.com/moby/buildkit/examples/systemd is not included in BuildKit v0.8.x, will be included in v0.9.x
# FIXME: now that we are at buildkit 0.20+, do we want to move over to their example systemd file?
RUN         cd /out/lib/systemd/system && \
            sedcomm='s@bin/containerd@bin/buildkitd@g; s@(Description|Documentation)=.*@@' && \
            sed -E "${sedcomm}" containerd.service > buildkit.service && \
            echo "" >> buildkit.service && \
            echo "# This file was converted from containerd.service, with \`sed -E '${sedcomm}'\`" >> buildkit.service
COPY        --from=dependencies-download-cli /out/share /out/share
RUN         --mount=target=/metadata-$BINARY_NAME,type=cache,from=dependencies-download-cli,source=/metadata \
            --mount=target=/metadata-containerd,type=cache,from=dependencies-download-containerd,source=/metadata \
            --mount=target=/metadata-runc,type=cache,from=dependencies-download-runc,source=/metadata \
            --mount=target=/metadata-soci,type=cache,from=dependencies-download-soci,source=/metadata \
            --mount=target=/metadata-bypass4netns,type=cache,from=dependencies-download-bypass4netns,source=/metadata \
            --mount=target=/metadata-slirp4netns,type=cache,from=dependencies-download-slirp4netns,source=/metadata \
            --mount=target=/metadata-tini,type=cache,from=dependencies-download-tini,source=/metadata \
            --mount=target=/metadata-cni,type=cache,from=dependencies-download-cni,source=/metadata \
            --mount=target=/metadata-rootlesskit,type=cache,from=dependencies-download-rootlesskit,source=/metadata \
            --mount=target=/metadata-buildg,type=cache,from=dependencies-download-buildg,source=/metadata \
            --mount=target=/metadata-imgcrypt,type=cache,from=dependencies-download-imgcrypt,source=/metadata \
            --mount=target=/metadata-buildkit,type=cache,from=dependencies-download-buildkit,source=/metadata \
            --mount=target=/metadata-cosign,type=cache,from=dependencies-download-cosign,source=/metadata \
            for item in /metadata-*; do \
                item="${item##*-}"; \
                printf "* %s:\n    - version: %s\n    - license: %s\n" "$item" "$(cat /metadata-$item/VERSION)" "$(cat /metadata-$item/LICENSE)" >> /out/share/doc/"$BINARY_NAME"-full/README.md; \
            done; \
            printf "Statically compiled (runc and others):\n* %s:\n    - license: %s\n" "libseccomp" "$SECCOMP_LICENSE" >> /out/share/doc/"$BINARY_NAME"-full/README.md; \
            printf "Statically compiled (soci):\n* %s:\n    - license: %s\n" "zlib1g" "$ZLIB_LICENSE" >> /out/share/doc/"$BINARY_NAME"-full/README.md; \
            printf "Statically compiled (slirp4netns):\n* %s:\n    - license: %s\n" "libglib2.0" "$GLIB_LICENSE" >> /out/share/doc/"$BINARY_NAME"-full/README.md; \
            printf "* %s:\n    - license: %s\n" "libcap" "$LIBCAP_LICENSE" >> /out/share/doc/"$BINARY_NAME"-full/README.md; \
            printf "* %s:\n    - license: %s\n" "libslirp" "$LIBSLIRP_LICENSE" >> /out/share/doc/"$BINARY_NAME"-full/README.md

# assembly-release is multi-architecture, and is the stage assembling all assets for full-release
# Once done, shasums will be generated and stuffed in to produce the full release
FROM        scratch AS assembly-release
COPY        --from=dependencies-build-containerd /out /
COPY        --from=dependencies-build-containerd-tools /out /
COPY        --from=dependencies-build-runc /out /
COPY        --from=dependencies-build-soci /out /
COPY        --from=dependencies-build-bypass4netns /out /
COPY        --from=dependencies-build-bypass4netnsd /out /
COPY        --from=dependencies-build-slirp4netns /out /
COPY        --from=dependencies-build-tini /out /
COPY        --from=dependencies-build-cni /out /
COPY        --from=dependencies-build-rootlesskit /out /
COPY        --from=dependencies-build-buildg /out /
COPY        --from=dependencies-build-imgcrypt /out /
COPY        --from=dependencies-build-buildkit /out /
COPY        --from=dependencies-build-cosign /out /usr/local/
#COPY        --from=dependencies-build-cosign-pkcs /out /usr/local/
COPY        --from=assembly-release-assets /out /
COPY        --from=dependencies-build-cli /out /

# assembly-shasum prepares the shasum file from above
FROM        --platform=$BUILDPLATFORM tooling-builder AS assembly-shasum
ARG         TARGETARCH
RUN         --mount=target=/src,type=cache,from=assembly-release,source=/ \
            (cd /src && find ! -type d | sort | xargs sha256sum > /out/SHA256SUMS ) && \
            chown -R 0:0 /out




#           tooling-runtime is the base stage that is used to build demo and testing images
#           Note that unlike every other tooling- stage, this is a multi-architecture stage
FROM        $UBUNTU_IMAGE:$UBUNTU_VERSION AS tooling-runtime
SHELL       ["/bin/bash", "-o", "errexit", "-o", "errtrace", "-o", "functrace", "-o", "nounset", "-o", "pipefail", "-c"]
ENV         DEBIAN_FRONTEND="noninteractive"
ENV         TERM="xterm"
ENV         LANG="C.UTF-8"
ENV         LC_ALL="C.UTF-8"
ENV         TZ="America/Los_Angeles"
#           FIXME: curl is only necessary for a single netns test. Fix the test and remove curl.
RUN         echo "force-unsafe-io" > /etc/dpkg/dpkg.cfg.d/farcloser-speedup && \
            echo 'Acquire::Languages "none";' > /etc/apt/apt.conf.d/farcloser-no-language && \
            echo 'Acquire::GzipIndexes "true";' > /etc/apt/apt.conf.d/farcloser-gzip-indexes && \
            apt-get update -qq >/dev/null && \
            apt-get install -qq --no-install-recommends \
                ca-certificates \
                apparmor \
                bash-completion \
                iptables \
                iproute2 \
                dbus dbus-user-session systemd systemd-sysv \
                curl \
                uidmap \
                openssh-server \
                openssh-client \
                    >/dev/null
ARG         BINARY_NAME
COPY        Dockerfile.d/systemd/entrypoint.service /etc/systemd/system/
COPY        Dockerfile.d/systemd/entrypoint.target /etc/systemd/system/
COPY        Dockerfile.d/systemd/entrypoint.sh /entrypoint.sh
RUN         systemctl mask systemd-firstboot.service systemd-udevd.service systemd-modules-load.service && \
            systemctl unmask systemd-logind && \
            systemctl enable entrypoint.service
ENTRYPOINT  ["/entrypoint.sh"]
CMD         ["bash", "--login", "-i"]

# assembly-runtime is the basis for the test integration environment
# this stage purposedly does NOT depend on the cli, so, it should be highly cacheable
FROM        tooling-runtime AS assembly-runtime
ARG         TARGETPLATFORM
# FIXME: finish removing unbuffer from the test codebase and then remove expect
# SSH is necessary for rootless testing (to create systemd user session).
# (`sudo` does not work for this purpose,
# OTOH `machinectl shell` can create the session but does not propagate exit code)
RUN         apt-get install -qq --no-install-recommends \
                expect \
                git \
                make \
                    >/dev/null
# Add all needed dependencies, but not the cli yet to avoid busting cache
COPY        --from=dependencies-build-containerd /out /usr/local
COPY        --from=dependencies-build-containerd-tools /out /usr/local
COPY        --from=dependencies-build-runc /out /usr/local
COPY        --from=dependencies-build-soci /out /usr/local/
COPY        --from=dependencies-build-bypass4netns /out /usr/local
COPY        --from=dependencies-build-bypass4netnsd /out /usr/local
COPY        --from=dependencies-build-slirp4netns /out /usr/local/
COPY        --from=dependencies-build-tini /out /usr/local/
COPY        --from=dependencies-build-cni /out /usr/local/
COPY        --from=dependencies-build-rootlesskit /out /usr/local/
COPY        --from=dependencies-build-buildg /out /usr/local
COPY        --from=dependencies-build-imgcrypt /out /usr/local
COPY        --from=dependencies-build-buildkit /out /usr/local/
COPY        --from=dependencies-build-cosign /out /usr/local/
#COPY        --from=dependencies-build-cosign-pkcs /out /usr/local/
# Add assets
COPY        --from=dependencies-build-containerd /containerd.service /usr/local/lib/systemd/system/containerd.service
# NOTE: github.com/moby/buildkit/examples/systemd is not included in BuildKit v0.8.x, will be included in v0.9.x
# FIXME: now that we are at buildkit 0.20+, do we want to move over to their example systemd file?
RUN         cd /usr/local/lib/systemd/system && \
            sedcomm='s@bin/containerd@bin/buildkitd@g; s@(Description|Documentation)=.*@@' && \
            sed -E "${sedcomm}" containerd.service > buildkit.service && \
            echo "" >> buildkit.service && \
            echo "# This file was converted from containerd.service, with \`sed -E '${sedcomm}'\`" >> buildkit.service
# Final preparations
RUN         mkdir -p -m 0755 /etc/cni
# Add go
ENV         PATH="/root/go/bin:/usr/local/go/bin:$PATH"
COPY        --from=tooling-downloader-golang /out/usr/local/$TARGETPLATFORM /usr/local
ENV         CGO_ENABLED=0
ENV         GOFIPS140=v1.0.0
ENV         GOTOOLCHAIN=local
ENV         GOFLAGS="$GOFLAGS -mod=vendor"
VOLUME      /var/lib/containerd
VOLUME      /var/lib/buildkit
VOLUME      /var/lib/"$BINARY_NAME"
VOLUME      /tmp

FROM        assembly-runtime AS assembly-integration
WORKDIR     /src
# Copy config and service files
COPY        ./Dockerfile.d/etc_containerd_config.toml /etc/containerd/config.toml
COPY        ./Dockerfile.d/etc_buildkit_buildkitd.toml /etc/buildkit/buildkitd.toml
COPY        ./Dockerfile.d/systemd/test-integration-buildkit-test.service /usr/local/lib/systemd/system/
COPY        ./Dockerfile.d/systemd/test-integration-soci-snapshotter.service /usr/local/lib/systemd/system/
# using test integration containerd config
COPY        ./Dockerfile.d/test-integration-etc_containerd_config.toml /etc/containerd/config.toml
RUN         perl -pi -e 's/multi-user.target/entrypoint.target/g' /usr/local/lib/systemd/system/*.service
# install ipfs service. avoid using 5001(api)/8080(gateway) which are reserved by tests.
RUN         systemctl enable \
                containerd  \
                buildkit \
                test-integration-buildkit-test  \
                test-integration-soci-snapshotter
# Install dev tools
RUN         --mount=target=/root/go/pkg/mod,type=cache \
            --mount=target=/src/Makefile,source=Makefile,type=bind \
            NO_COLOR=true GOFLAGS= make install-dev-gotestsum; chmod -R a+rx /root/go/bin

########################################################################################################################
# Final
# These stages are high-level targets that correspond to a specific task (release, integration, etc)
########################################################################################################################
#           release-full is the final stage producing the -full releases, adding the computed SHASUM
FROM        assembly-release AS release-full
ARG         BINARY_NAME
COPY        --from=assembly-shasum /out/SHA256SUMS /share/doc/"$BINARY_NAME"-full/SHA256SUMS

#           release-demo is a fully running stack in a container
FROM        tooling-runtime AS release-demo
COPY        --from=release-full / /usr/local
RUN         mkdir -p /etc/bash_completion.d && \
            "$BINARY_NAME" completion bash >/usr/share/bash-completion/completions/"$BINARY_NAME"

#           test-integration is the final stage for the integration testing environment
FROM        assembly-integration AS test-integration
COPY        --from=dependencies-build-cli /out /usr/local
RUN         mkdir -p /etc/bash_completion.d && \
            "$BINARY_NAME" completion bash >/usr/share/bash-completion/completions/"$BINARY_NAME"
COPY        --from=dependencies-download-cli /src/vendor /src/vendor
#           copy source - note this is volatile and not cacheable
COPY        . /src
CMD         ["./hack/test-integration.sh"]

#           test-integration-rootless
FROM        test-integration AS test-integration-rootless
# TODO: update containerized-systemd to enable sshd by default, or allow `systemctl wants <TARGET> ssh` here
RUN         ssh-keygen -q -t rsa -f /root/.ssh/id_rsa -N '' && \
            useradd -m -s /bin/bash rootless && \
            mkdir -p -m 0700 /home/rootless/.ssh && \
            cp -a /root/.ssh/id_rsa.pub /home/rootless/.ssh/authorized_keys && \
            mkdir -p /home/rootless/.local/share && \
            chown -R rootless:rootless /home/rootless
COPY        ./Dockerfile.d/etc_systemd_system_user@.service.d_delegate.conf /etc/systemd/system/user@.service.d/delegate.conf
VOLUME      /home/rootless/.local/share
COPY        ./Dockerfile.d/test-integration-rootless.sh /
RUN         chmod a+rx /test-integration-rootless.sh
CMD         ["/test-integration-rootless.sh", "./hack/test-integration.sh"]

# test for CONTAINERD_ROOTLESS_ROOTLESSKIT_PORT_DRIVER=slirp4netns
FROM        test-integration-rootless AS test-integration-rootless-port-slirp4netns
COPY        ./Dockerfile.d/home_rootless_.config_systemd_user_containerd.service.d_port-slirp4netns.conf /home/rootless/.config/systemd/user/containerd.service.d/port-slirp4netns.conf
RUN         chown -R rootless:rootless /home/rootless/.config

########################################################################################################################
# Auditing
# These stages are meant to perform additional analysis on the binaries that do not belong in test nor linting
########################################################################################################################
# this stage does run sanity checks on the output:
# - verify all binaries architecture is matching the target
# - verify all binaries are static and running
FROM        tooling-runtime AS release-full-sanity
ARG         TARGETARCH
RUN         apt-get install -qq --no-install-recommends \
                binutils \
                patchelf \
                devscripts \
                    >/dev/null
COPY        ./Dockerfile.d/helpers/sanity.sh /
#           All binaries are expected to be static and to run
ARG         STATIC=true
ARG         RUNNING=true
#           All CGO and C binaries must be PIE + BIND_NOW + RO_RELOCATIONS
ARG         CBINS="runc containerd bypass4netns soci soci-snapshotter-grpc slirp4netns tini"
#           We do not link against protectable libc functions, so...
ARG         FORTIFIED=true
ARG         STACK_CLASH=true
ARG         STACK_PROTECTED=false
#           bypass4netns will crash if this is not set
ENV         XDG_RUNTIME_DIR=/tmp
WORKDIR     /src
RUN         --mount=target=/src,type=cache,from=release-full,source=/ \
            sha256sum -c share/doc/*/SHA256SUMS; \
            cd bin; \
            filearch="$(echo "$TARGETARCH" | sed -e s/amd64/x86-64/ -e s/arm64/aarch64/)"; \
            for i in ./*; do \
                echo "Auditing $i"; \
                ff="$(file -L "$i")"; \
                ! grep -q "POSIX shell script," <(echo "$ff") || { \
                    echo "Skipping test for shell script"; \
                    continue; \
                }; \
                grep -q "$filearch," <(echo "$ff") || { \
                    echo "Wrong architecture: $ff (expected: $filearch)"; \
                    exit 1; \
                }; \
                [[ "$CBINS" == *"$(basename "$i")"* ]] && \
                    { export PIE=true; export BIND_NOW=true; export RO_RELOCATIONS=true; } || \
                    { export PIE=false; export BIND_NOW=false; export RO_RELOCATIONS=false; }; \
                /sanity.sh validate "$i";  \
            done
