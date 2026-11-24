#!/usr/bin/env bash
# pr-size-waiver.sh — the step of .github/workflows/pr-size.yml that decides the PR size check: runs the sibling
# scripts/pr-size.sh and waives its exit 1 (over the limit) for a pull request that carries the `size:exception` label.
#
# Usage:  HAS_EXCEPTION=true|false pr-size-waiver.sh <base> <head>   (arguments go to pr-size.sh as they are)
# Env:    HAS_EXCEPTION   exactly "true" when the PR has the size:exception label; anything else (unset, empty) is no label
# Exits:  pr-size.sh's exit, except exit 1 with the label, which becomes 0 after a `::warning` line. Exit 3 (usage or
#         environment: the size was not measured) is never waived, label or not.
# tests/pr-size.bats PS15 drives it against a stub pr-size.sh for each case.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
rc=0
bash "$here/pr-size.sh" "$@" || rc=$?
if [ "$rc" -eq 1 ] && [ "${HAS_EXCEPTION:-}" = "true" ]; then
  echo "::warning title=PR size::over the size limit, allowed by the size:exception label"
  exit 0
fi
exit "$rc"
