#!/usr/bin/env bash
# FIXME: goimports-reviser is currently broken when it comes to ./...
# Specifically, it will ignore arguments, and will return exit 0 regardless
# This here is a workaround, until they fix it upstream: https://github.com/incu6us/goimports-reviser/pull/157

# shellcheck disable=SC2034,SC2015
set -o errexit -o errtrace -o functrace -o nounset -o pipefail

ex=0

while read -r file; do
  goimports-reviser -list-diff -set-exit-status -output stdout -company-prefixes "go.farcloser.world" "$file" || {
    ex=$?
    >&2 printf "Imports are not listed properly in %s. Consider calling make lint-fix-imports.\n" "$file"
  }
done < <(find ./ -type f -name '*.go')

exit "$ex"
