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

rm -f /tmp/test-prevent-concurrency/.lock

# empty:
# apparmor

for pack in apparmor builder completion compose container helpers image inspect internal issues login . namespace network system volume; do
	go test /Users/dmp/Projects/go/lepton/hack/../cmd/nerdctl/$pack -p 1 -args -test.allow-kill-daemon -test.only-flaky=false
done

for pack in apparmor builder completion helpers inspect internal issues . namespace system volume network  . namespace system volume network image; do
	go test /Users/dmp/Projects/go/lepton/hack/../cmd/nerdctl/$pack -p 1 -args -test.allow-kill-daemon -test.only-flaky=true "$@"
done

#for pack in login; do
#	go test /Users/dmp/Projects/go/lepton/hack/../cmd/nerdctl/$pack -p 1 -args -test.allow-kill-daemon -test.only-flaky=true "$@"
#done

#for pack in container; do
#	go test /Users/dmp/Projects/go/lepton/hack/../cmd/nerdctl/$pack -p 1 -args -test.allow-kill-daemon -test.only-flaky=true "$@"
#done

for pack in compose; do
	go test /Users/dmp/Projects/go/lepton/hack/../cmd/nerdctl/$pack -p 1 -args -test.allow-kill-daemon -test.only-flaky=true "$@"
done
