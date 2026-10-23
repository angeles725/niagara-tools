#!/usr/bin/env bash
# gen-lint-index.sh — generates toolbelt/INDEX.md, the lint index, from each toolbelt/lint-*.sh header.
#
# BUILD-LOOP §5/§6.b used to enumerate every lint inline; each new lint edited that enumeration
# (a merge-conflict hotspot) and the prose drifted. The index is now DERIVED: one row per
# toolbelt/lint-*.sh with its header description, Usage:, Exit contract, [ev: ...] tags and
# whether report-module.sh runs it automatically. Prose never enumerates lints or counts them.
# [ev: retro kit-meta-hygiene-2026-10-01 Δ3]
#
# Header contract (the leading run of '#' and blank lines of every lint-*.sh, no line cap):
#   line 2      # <script-name> — <what it checks>   (continuation lines joined until a blank '#',
#               a keyword line or line 5: the description spans header lines 2-5 at most, so a
#               longer explanation belongs below the blank '#' and stays out of the index)
#   Usage line  # Usage: <script-name> [flags] <arg>
#   Exit line   # Exit: ... | # Exits: ... | # Exit <code> ...
# Usage and Exit lines are read from the header block only (the leading run of '#' and blank lines): a
# `# Exit ...` code comment below the first non-comment line never joins the cell (#226).
# Every Usage and Exit line is kept, in order (joined with <br> in the cell). A '#' line whose text
# starts at or right of the value column of the Usage/Exit line above it is a wrapped continuation
# and is joined with a space; a less-indented line (an explanation) or a 'Word:' line is not.
# A lint missing any of the three is a contract violation (exit 1, nothing written).
# Auto = yes only when a non-comment line of report-module.sh invokes "$TOOLBELT/<script-name>" in
# command position (line start, after $( ( ` ; & | { or a command keyword); a comment, a message
# string, a printf argument, an assignment or a longer script name does not count (#226).
#
# Implementation: the Go module at <repo>/go (go/internal/lintindex). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/gen-lint-index when the binary is missing or older
# than a Go source, then this script execs it. Needs `go` on PATH to build. [ev: retro go-gen-lint-index]
# Mutation pins (GLI-stale, GLI-auto, GLI-wrap-exit, GLI-multi, GLI-exit-block, GLI-awkfail) live in the Go source and tests.
#
# Usage:  gen-lint-index.sh [--check] [<kit-root>]     (default kit root: this script's ../)
# Exits:  0 written / index fresh · 1 stale or missing index under --check, or header-contract
#         violation · 3 usage/env, an unreadable lint header, no Go toolchain or a failed build
#         (nothing written)
# VCS-free by design (kit-links L2). Deterministic: C byte order, no timestamps.
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'gen-lint-index: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" gen-lint-index)" || exit 3
GEN_LINT_INDEX_DEFAULT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
