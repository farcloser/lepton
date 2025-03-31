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

# shellcheck disable=SC2034
set -o errexit -o errtrace -o functrace -o nounset -o pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]:-$PWD}")" 2>/dev/null 1>&2 && pwd)"
readonly root
# shellcheck source=/dev/null
. "$root/scripts/lib.sh"

# "Blacklisting" here means that any dependency which name is blacklisted will be left untouched, at the version
# currently pinned in the Dockerfile.
# This is convenient so that currently broken alpha/beta/RC can be held back temporarily to keep the build green
blacklist=(
)

# List all the repositories we depend on to build and run integration tests
dependencies=(
  containerd/containerd
  opencontainers/runc
  containernetworking/plugins
  moby/buildkit
  containerd/imgcrypt
  sigstore/cosign
  rootless-containers/rootlesskit
  rootless-containers/slirp4netns
  rootless-containers/bypass4netns
  ktock/buildg
  awslabs/soci-snapshotter
  krallin/tini
)

canary::build::integration(){
  extras=()

  for dep in "${dependencies[@]}"; do
    local bl=""
    shortname="${dep##*/}"
    [ "$shortname" != "plugins" ] || shortname="cni-plugins"
    for bl in "${blacklist[@]}"; do
      if [ "$bl" == "$shortname" ]; then
        log::warning "Dependency $shortname is blacklisted and will be left to its currently pinned version"
        break
      fi
    done
    [ "$bl" != "$shortname" ] || continue

    shortsafename="$(printf "%s" "$shortname" | tr '[:lower:]' '[:upper:]' | tr '-' '_')"

    higher_readable="$(github::tags::latest "$dep")"
    revision="$(jq -rc .commit.sha <<<"$higher_readable")"
    higher_readable="$(jq -rc .name <<<"$higher_readable")"

    while read -r line; do
      # Extract value after "=" from a possible dockerfile `ARG XXX_VERSION`
      old_version=$(echo "$line" | grep -E "ARG\s+${shortsafename}_VERSION=") || true
      old_version="${old_version##*=}"
      [ "$old_version" != "" ] || continue

      if [ "$old_version" != "$higher_readable" ]; then
        log::warning "Dependency ${shortsafename} is going to use an updated version $higher_readable (currently: $old_version)"
        extras+=(--opt build-arg:"${shortsafename}_VERSION=$higher_readable" --opt build-arg:"${shortsafename}_REVISION=$revision")
      fi
    done < ./Dockerfile
  done

  extras+=(--opt build-arg:"GO_VERSION=${GO_VERSION:-canary}")

  export extras
}
