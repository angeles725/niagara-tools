#!/usr/bin/env bash
# station-snapshot.sh — Local audit-surface snapshot for a Niagara station directory.
# Campaign 8 PR9. Copies config.bog + console*.txt into <out-dir> and writes
# manifest.json with sha256, bytes, and a UTC timestamp, without touching the
# source directory. History/alarm db files are recorded as pointers (path + size)
# only — never copied. Feeds bog-audit and triage-console in the post-deploy flow.
#
# Usage:  station-snapshot.sh <station-dir> <out-dir>
#
#   Copies ONLY:  config.bog  console*.txt  (dot-dirs pruned; D9b)
#   Pointers:     history/**  (relative path + bytes; never copied)
#   Writes:       <out-dir>/manifest.json
#
#   manifest.json shape:
#     { "station": "<name>", "ts": "<UTC-stamp>",
#       "files": [ {"relpath":"…","sha256":"…","bytes":<n>} … ],
#       "pointers": [ {"relpath":"…","bytes":<n>} … ] }
#
#   Row format:  printf '%-4s  %-14s  %s  %s\n' STATUS station-snap relpath detail
#   Exits:       0 ok · 1 a listed file could not be copied · 3 usage/env (K20), including no Go toolchain
#                or a failed build
#
# Source is opened read-only (no write, no chmod, no temp file in station-dir).
# Refuses <out-dir> inside <station-dir> (exit 3).
#
# Implementation: go/internal/stationsnapshot (goldens recorded from the bash script at 4d602a9b by
# go/internal/lint/rules/golden/record/record.sh). go/go-tool.sh `go_tool_bin` builds go/bin/station-snapshot
# when it is missing or older than a Go source, then this script execs it with STATION_SNAPSHOT_SCRIPT = its own
# path (the prefix of the shell-style error messages the bash printed).
# The wrapper needs dirname, readlink and find on PATH (go too when the binary must be rebuilt); the port still
# requires sha256sum on PATH, as the bash did.
# This script is VCS-free by design; kit-links.bats L2 enforces.
# [ev: retro campaign8-station-snapshot] [ev: retro go-station-snapshot]
set -u
# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'station-snapshot: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" station-snapshot)" || exit 3
STATION_SNAPSHOT_SCRIPT="${BASH_SOURCE[0]}" exec "$BIN" "$@"
