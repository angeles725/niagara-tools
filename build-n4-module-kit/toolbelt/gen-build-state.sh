#!/usr/bin/env bash
# gen-build-state.sh — generates the per-change parts of the kit block of BUILD-STATE.md from ledger.d/.
#
# The kit block's `last_change:` line, its `open_issues:` items and its session log (the lines after
# `last_session:`) used to be hand-edited by every kit PR at the same spots, so parallel sessions
# conflicted there and "keep both sides" resolutions lost or duplicated entries. Each change now adds
# ONE file, ledger.d/<YYYY-MM-DD>-<slug>.md, and these three parts are DERIVED from those files; on a
# merge conflict in BUILD-STATE.md, keep either side and regenerate. [ev: retro ledger-fragments]
#
# Fragment contract (line based; any other line is a violation naming the file and line):
#   <!-- ledger: kit -->                   required
#   <!-- date: YYYY-MM-DD -->              required; equals the filename date
#   <!-- last-change-note: <text> -->      optional; the `# comment` of last_change
#   log: YYYY-MM-DD · <text>               exactly one; the session-log line (date = the header date)
#   open_issue: <text>                     zero or more; each becomes an `  - <text>` item
# last_change = the newest date (tie: later filename); open_issues = newest fragment first; session log =
# oldest first. ledger.d/0000-archive-pre-2026-10-23.md holds the ledger as it stood before fragments.
# Module (non-kit) blocks and the other kit fields stay hand-kept.
#
# Implementation: go/internal/buildstate, built on demand by go/go-tool.sh `go_tool_bin` (as
# gen-retro-index.sh). Needs `go` on PATH to build. Mutation pins (GBS-stale, GBS-order, GBS-last,
# GBS-tail) live in the Go source and tests.
#
# Usage:  gen-build-state.sh [--check] [<kit-root>]     (default kit root: this script's ../)
# Exits:  0 written / fresh · 1 stale under --check, fragment or BUILD-STATE structure violation
#         (nothing written) · 3 usage/env, including no Go toolchain or a failed build
# VCS-free by design (kit-links L2).
set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'gen-build-state: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" gen-build-state)" || exit 3
GEN_BUILD_STATE_DEFAULT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
