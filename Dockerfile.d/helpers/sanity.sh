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

readonly CHECKLIST=(BIND_NOW PIE STACK_PROTECTED STACK_CLASH FORTIFIED RO_RELOCATIONS STATIC RUNNING NO_SYSTEM_LINK)
com="$1"
binary="$2"
shift
shift

HARDENING_CHECK="$(hardening-check "$binary" 2>/dev/null || true)"
readonly HARDENING_CHECK
READELF="$(readelf -d "$binary")"
readonly READELF
DYN="$(readelf -p .interp "$binary" 2>/dev/null)"
readonly DYN
# ldd is ridiculously buggy
LDDL="$(ldd "$binary" 2>/dev/null | grep -v " => /lib/" || true)"
readonly LDDL
# ldd is ridiculously buggy
LDDB="$(readelf -d "$binary" | grep NEEDED || true)"
readonly LDDB

passed=()
inconclusive=()
failed_ignored=()
failed=()

do_check(){
  local check_source="$1"
  local pattern="$2"
  local success_pattern="$3"
  local indecisive_pattern="$4"
  local required="$5"
  echo "$check_source" | grep "$pattern" | grep -Eq "$success_pattern" && passed+=("$check") || {
    echo "$check_source" | grep "$pattern" | grep -Eq "$indecisive_pattern"  && inconclusive+=("$check")
  } || { [ "$required" == true ] && failed+=("$check"); } || failed_ignored+=("$check")
}

validate(){
  local binary="$1"
  local check

  [ ! -d "$binary" ] || {
    print "%s is a directory. Doing nothing" "$binary"
    return
  }

  for check in "${CHECKLIST[@]}"; do
    case "$check" in
      "BIND_NOW")
        do_check "$READELF" "" "BIND_NOW" "no indecisive_pattern" "${!check:-}"
      ;;
      "PIE")
        do_check "$READELF" "" "PIE" "no indecisive_pattern" "${!check:-}"
      ;;
      "STACK_PROTECTED")
        do_check "$HARDENING_CHECK" "Stack protected" "yes" "unknown, no symbols found" "${!check:-}"
      ;;
      "STACK_CLASH")
        do_check "$HARDENING_CHECK" "Stack clash protection" "yes" "unknown, no -fstack-clash-protection instructions found" "${!check:-}"
      ;;
      "FORTIFIED")
        do_check "$HARDENING_CHECK" "Fortify Source functions" "yes" "unknown, no protectable libc functions used" "${!check:-}"
      ;;
      "RO_RELOCATIONS")
        do_check "$HARDENING_CHECK" "Read-only relocations" "yes" "^$" "${!check:-}"
      ;;
      "STATIC")
        do_check "$DYN" "" "^$" "" "${!check:-}"
      ;;
      "RUNNING")
        "$binary" "--version" >/dev/null 2>&1 \
        || "$binary" version >/dev/null 2>&1  \
        || "$binary" --help >/dev/null 2>&1 \
        && passed+=("$check") || { [ "${!check:-}" != true ] && failed_ignored+=("$check"); } || {
          failed+=("$check")
          >&2 printf "FAILING TO RUN BINARY. This is usually quite bad. Output was:\n"
          >&2 "$binary" "--version" || true
        }
      ;;
      "NO_SYSTEM_LINK")
        [ ! "${STATIC:-}" ] || { passed+=("$check"); continue; }
        local faillink=
        while read -r line; do
          [ "$line" ] || continue
          printf "%s" "$line" | grep -q /dist || { faillink=true; failed+=("$check ($line)"); }
        done <<<"$(printf "%s" "$LDDL" | grep "=>")"
        [ "$faillink" ] || passed+=("$check")
      ;;
    esac
  done

}

case "$com" in
  "")
  ;;
  *)
    validate "$binary"
  ;;
esac


printf "***************************************\n"
printf "Binary report for %s\n" "$binary"
printf "***************************************\n"
tput setaf 2
printf "Successfull checks: %s\n" "${passed[*]}"
tput op
printf "\n"
tput setaf 3
printf "Inconclusive: %s\n" "${inconclusive[*]}"
tput op
printf "\n"
tput setaf 3
printf "Failed, but not required: %s\n" "${failed_ignored[*]}"
tput op
printf "\n"
tput setaf 1
printf "Failed tests: %s\n" "${failed[*]}"
tput op
if [ "${#failed[@]}" != 0 ]; then
  printf "\n"
  printf "LDDB: %s\n" "$LDDB"
  printf "LDDL: %s\n" "$LDDL"
  printf "DYN: %s\n" "$DYN"
  printf "READELF: %s\n" "$READELF"
  printf "HARDENING_CHECK: %s\n" "$HARDENING_CHECK"
fi