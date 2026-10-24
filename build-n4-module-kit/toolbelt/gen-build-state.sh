#!/usr/bin/env bash
# gen-build-state.sh — generates BUILD-STATE.md from BUILD-STATE.src.md + ledger.d/, or validates them.
#
# The kit block's `last_change:` line, its `open_issues:` items and its session log (the lines after
# `last_session:`) used to be hand-edited by every kit PR at the same spots; then (ledger-fragments) they
# were derived from one ledger.d/<YYYY-MM-DD>-<slug>.md per change but the derived file was still
# committed, so every PR still conflicted on it. Now BUILD-STATE.md is GENERATED and NOT committed
# (.gitignore): BUILD-STATE.src.md holds every hand-kept field with those three parts empty, and this
# tool fills them in. Readers run toolbelt/kit-state.sh build-state. [ev: retro ledger-generated-not-committed]
#
# Fragment contract (line based; any other line is a violation naming the file and line):
#   <!-- ledger: kit -->                   required
#   <!-- date: YYYY-MM-DD -->              required; equals the filename date
#   <!-- last-change-note: <text> -->      optional; the `# comment` of last_change
#   log: YYYY-MM-DD · <text>               exactly one; the session-log line (date = the header date)
#   open_issue: <text>                     zero or more; each becomes an `  - <text>` item
# last_change = the newest date (tie: later filename); open_issues = newest fragment first; session log =
# oldest first. ledger.d/0000-archive-pre-2026-10-23.md holds the ledger as it stood before fragments.
# Module (non-kit) blocks and the other kit fields stay hand-kept in BUILD-STATE.src.md; --check fails a
# source whose kit envelope carries generated content (it would be dropped silently).
#
# Implementation: go/internal/buildstate, built on demand by go/go-tool.sh `go_tool_bin` (as
# gen-retro-index.sh). Needs `go` on PATH to build. Mutation pins (GBS-stale, GBS-order, GBS-last,
# GBS-tail, GBS-source) live in the Go source and tests.
#
# Usage:  gen-build-state.sh [--check] [<kit-root>]     (default kit root: this script's ../)
# Exits:  0 written / inputs valid under --check · 1 fragment or BUILD-STATE.src.md violation (nothing
#         written) · 3 usage/env, including a missing source, no Go toolchain or a failed build
# VCS-free by design (kit-links L2).
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'gen-build-state: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" gen-build-state)" || exit 3
GEN_BUILD_STATE_DEFAULT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
