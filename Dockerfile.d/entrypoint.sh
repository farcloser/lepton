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

[ $# -ne 0 ] || {
	>&2 printf "ERROR: No command specified.\n"
	exit 1
}

[ -t 0 ] || {
	>&2 printf "ERROR: TTY needs to be enabled.\n"
	exit 1
}

env >/etc/entrypoint-env

printf " %q" "$@" >/etc/entrypoint-cmd
printf "\n" >>/etc/entrypoint-cmd

# FIXME: AppArmor fails loading profiles on Lima - /proc/sys/kernel/apparmor_restrict_unprivileged_userns
#sysctl -w kernel.apparmor_restrict_unprivileged_userns=1

exec systemd --unit=entrypoint.target
