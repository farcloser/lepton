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

readonly binary=lepton

# This is mildly annoying
x=0
# FIXME: this won't work in rootless
! command -v systemctl >/dev/null || while ! systemctl is-active containerd >/dev/null && [ "$x" -lt 20 ]; do
  x=$((x+1))
  sleep 0.5
done
[ "$x" -lt 20 ] || {
  echo "failed waiting for systemd units to be active"
  exit 42
}

readonly timeout="60m"
readonly retries="2"
readonly needsudo="${WITH_SUDO:-}"

args=(--format=testname --jsonfile /tmp/test-integration.log --packages="$root"/../cmd/"$binary"/...)

if [ "$#" == 0 ]; then
  "$root"/test-integration.sh -test.only-flaky=false
  "$root"/test-integration.sh -test.only-flaky=true
  exit
fi

for arg in "$@"; do
  if [ "$arg" == "-test.only-flaky=true" ] || [ "$arg" == "-test.only-flaky" ]; then
    args+=("--rerun-fails=$retries")
    break
  fi
done

if [ "$needsudo" == "true" ] || [ "$needsudo" == "yes" ] || [ "$needsudo" == "1" ]; then
  gotestsum "${args[@]}" -- -timeout="$timeout" -p 1 -exec sudo -args -test.allow-kill-daemon "$@"
else
  gotestsum "${args[@]}" -- -timeout="$timeout" -p 1 -args -test.allow-kill-daemon "$@"
fi

echo "These are the tests that took more than 10 seconds:"
gotestsum tool slowest --threshold 10s --jsonfile /tmp/test-integration.log
