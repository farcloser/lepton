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

if [[ "$(id -u)" = "0" ]]; then
	if [ -e /sys/kernel/security/apparmor/profiles ]; then
		# Load the "prefix-default" profile for TestRunApparmor
		lepton apparmor load
	fi

  sysctl -w net.ipv4.ip_unprivileged_port_start=0

  # Get what we want to passthrough from the environment in the whitelist and bypass variables
  export BYPATH="$PATH"
  export BYPWD="$PWD"
  whitelist="BYPATH,BYPWD"
  while read -r line; do
    ! grep -Eq "^(GO|LEPTON|NAMESPACE).*" <<<"$line" || whitelist+=",${line%%=*}"
  done < <(env)

  # Login as rootless and restart the script
  loginctl enable-linger rootless
	exec systemd-run --system --scope su --whitelist-environment="$whitelist" --pty --login rootless -c "$0 $*"
else
  # Recover the path and current working directory (PATH cannot be whitelisted with su)
  cd "${BYPWD:-.}"
  export PATH="$BYPATH"
  # Set dbus and XDG runtime
  DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$(id -u)/bus"
  XDG_RUNTIME_DIR="/run/user/$(id -u)"
  export DBUS_SESSION_BUS_ADDRESS
  export XDG_RUNTIME_DIR
  export XDG_SESSION_CLASS=user

  # Wait for systemd to be active...
  x=0
  while ! systemctl --user show-environment >/dev/null 2>&1 && [ "$x" -lt 20 ]; do
    x=$((x+1))
    sleep 0.5
  done
  [ "$x" -lt 20 ] || {
    echo "failed waiting for systemd --user to be active"
    exit 42
  }

	# why would we need this?
  #	if grep -q "options use-vc" /etc/resolv.conf; then
  #		containerd-rootless-setuptool.sh nsenter -- sh -euc 'echo "options use-vc" >>/etc/resolv.conf'
  #	fi

  # Setup rootless services
	containerd-rootless-setuptool.sh install
	CONTAINERD_NAMESPACE="$NAMESPACE" containerd-rootless-setuptool.sh install-buildkit-containerd
	containerd-rootless-setuptool.sh install-bypass4netnsd

	if [ ! -f "/home/rootless/.config/containerd/config.toml" ] ; then
	  echo "version = 2" > /home/rootless/.config/containerd/config.toml
	  # cp /etc/containerd/config.toml /home/rootless/.config/containerd/config.toml
	fi

	systemctl --user restart containerd.service

  exec "$@"
fi