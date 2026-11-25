#!/usr/bin/env bash
# pr-size-base.sh — put the BASE revision's PR size gate scripts in a directory, so .github/workflows/pr-size.yml runs
# the gate as it is on the base branch and a pull request cannot weaken its own gate (issue #979).
#
# Usage:  pr-size-base.sh <base> <dest-dir>     (run inside the checkout; <dest-dir> is created if missing)
# Writes: <dest-dir>/scripts/<file> for each gate script (GATE_FILES below), read with `git show <base>:scripts/<file>`.
#         The scripts run from there as they run from scripts/: pr-size-waiver.sh finds pr-size.sh beside itself.
# New gate: a file absent at <base> (the PR that adds it) is copied from the checkout instead, after a `::notice` line.
# Lib:    a gate script that sources another file must list it in GATE_FILES (tests/pr-size.bats PS19 fails on a
#         `source` in either script today).
# Exits:  0 done · 3 usage/environment: bad arguments, an unknown <base>, a file at neither <base> nor the checkout
set -euo pipefail

GATE_FILES=(scripts/pr-size.sh scripts/pr-size-waiver.sh)

die() { printf 'pr-size-base: %s\n' "$1" >&2; exit 3; }

[ "$#" -eq 2 ] || die "usage: pr-size-base.sh <base> <dest-dir>"
base="$1"
dest="$2"
top="$(git rev-parse --show-toplevel 2>/dev/null)" || die "not inside a git repository"
git rev-parse --verify --quiet "$base^{commit}" >/dev/null || die "unknown base: $base"

for f in "${GATE_FILES[@]}"; do
  mkdir -p "$dest/$(dirname "$f")"
  if git cat-file -e "$base:$f" 2>/dev/null; then
    git show "$base:$f" > "$dest/$f"
  elif [ -f "$top/$f" ]; then
    echo "::notice title=PR size::$f is not at the base revision, using the head's"
    cp "$top/$f" "$dest/$f"
  else
    die "$f is at neither the base revision nor the checkout"
  fi
done
