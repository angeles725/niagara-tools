#!/usr/bin/env bash
# lint-clock-zero-floor.sh — flags the naive Math.max(0, delay) floor near Clock.schedule.
#
# Clock / EngineManager rejects a schedule time <= 0 the SAME as a negative — so the
# common defensive floor Math.max(0L, delay) STILL throws when delay computes to 0. The
# floor must be 1 ms: Math.max(1L, delay). A self-firing timer floored at 0 silently
# dies. See types/issues-and-gotchas.md B4. [ev: retro 2026-09-19-gotchas-tier-a]
#
# Implementation: go/internal/lint/rules/clockzerofloor on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-clock-zero-floor <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_each did).
#
# Usage:  lint-clock-zero-floor.sh <src-root>
#   Only files that also call Clock.schedule are reported (reduces false positives). Both the call
#   and the floor are matched in the code only: a comment, Javadoc or string never counts, and a
#   `//` inside a string never hides the rest of a line.
#   Row:  WARN  lint-clock-zero-floor  <file>:<line>  zero-floor: <source>
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: CZF2 -- drop the Math.max(0 match and CZF2 stops warning on a zero-floor near Clock.schedule
# Mutation: CZF-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: CZF4 -- matching the raw line (no comment/literal blanking) WARNs a commented or string zero floor
# Mutation: CZF-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-clock-zero-floor: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-clock-zero-floor "$@"
