#!/usr/bin/env bash

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

set -o errexit -o errtrace -o functrace -o nounset -o pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]:-$PWD}")" 2>/dev/null 1>&2 && pwd)"
readonly root
# shellcheck source=/dev/null
. "$root/scripts/lib.sh"

GO_VERSION=1.24
CNI_PLUGINS_VERSION=v1.5.1

[ "$(uname -m)" == "aarch64" ] && GOARCH=arm64 || GOARCH=amd64

_rootful=

configure::rootful(){
  log::debug "Configuring rootful to: ${1:+true}"
  _rootful="${1:+true}"
}

# shellcheck disable=SC2120
install::kubectl(){
  local version="${1:-}"
  [ "$version" ] || version="$(http::get /dev/stdout https://dl.k8s.io/release/stable.txt)"
  local temp
  temp="$(fs::mktemp "install")"

  http::get "$temp"/kubectl "https://dl.k8s.io/release/$version/bin/linux/${GOARCH:-amd64}/kubectl"
  host::install "$temp"/kubectl
}

install::cni(){
  local version="$1"
  local temp
  temp="$(fs::mktemp "install")"

  http::get "$temp"/cni.tgz "https://github.com/containernetworking/plugins/releases/download/$version/cni-plugins-${GOOS:-linux}-${GOARCH:-amd64}-$version.tgz"
  sudo mkdir -p /opt/cni/bin
  sudo tar xzf "$temp"/cni.tgz -C /opt/cni/bin
  mkdir -p ~/opt/cni/bin
  tar xzf "$temp"/cni.tgz -C ~/opt/cni/bin
  rm "$temp"/cni.tgz
}

exec::kind(){
  local args=()
  [ ! "$_rootful" ] || args=(sudo env PATH="$PATH" KIND_EXPERIMENTAL_PROVIDER="$KIND_EXPERIMENTAL_PROVIDER")
  args+=(kind)

  log::debug "${args[*]} $*"
  "${args[@]}" "$@"
}

exec::cli(){
  local args=()
  [ ! "$_rootful" ] || args=(sudo env PATH="$PATH")
  args+=("$(pwd)"/_output/lepton)

  log::debug "${args[*]} $*"
  "${args[@]}" "$@"
}

# Install dependencies
main(){
  log::info "Configuring rootful"
  configure::rootful "${ROOTFUL:-}"

  log::info "Installing host dependencies if necessary"
  host::require kind 2>/dev/null
  host::require kubectl 2>/dev/null || install::kubectl

  # Build cli to use for kind
  make build
  ln -s "$(pwd)"/_output/lepton "$(pwd)"/_output/nerdctl
  PATH=$(pwd)/_output:"$PATH"
  export PATH

  # Add CNI plugins
  install::cni "$CNI_PLUGINS_VERSION"

  # Hack to get go into kind control plane
  exec::cli rm -f go-kind 2>/dev/null || true
  # FIXME: 429. Get rid of this and use straight golang install instead
  exec::cli run -d --quiet --name go-kind golang:"$GO_VERSION" sleep Inf
  exec::cli cp go-kind:/usr/local/go /tmp/go
  exec::cli rm -f go-kind

  # Create fresh cluster
  log::info "Creating new cluster"
  export KIND_EXPERIMENTAL_PROVIDER=nerdctl
  exec::kind delete cluster --name cli-test 2>/dev/null || true
  exec::kind create cluster --name cli-test --config=./hack/kind.yaml
}

main "$@"
