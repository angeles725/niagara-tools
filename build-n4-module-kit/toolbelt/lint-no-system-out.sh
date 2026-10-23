#!/usr/bin/env bash
# lint-no-system-out.sh — flags System.out / System.err print calls in module Java.
#
# System.out is captured by StdoutManager -> stdout.txt / spy:/stdout; it bypasses
# java.util.logging entirely: no logger name, no severity classification, and no entry
# in BLogHistoryService (so it is NOT BQL-queryable). A station operator never sees it in
# the Logging service. Module diagnostics MUST use java.util.logging.Logger.
# See types/observability.md §1. [ev: retro 2026-09-19-observability]
#
# Implementation: go/internal/lint/rules/nosystemout on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-no-system-out <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_each did).
#
# Usage:  lint-no-system-out.sh <src-root>
#   Scans *.java under <src-root>, dot-dirs pruned (D9b). Line // comments are stripped
#   before matching, so commented-out code does not trigger.
#   Row:   FAIL  lint-no-system-out  <file>:<line>  <trimmed source>
#   Exit:  0  no FAIL (clean) · 1  any FAIL · 3  usage/env or a sub-directory find cannot enter
#
# Known limitation: a `//` inside a string literal on the same line as a System.out call
# can hide it (rare). VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# Mutation: NSO2 -- drop the System.out.print case and NSO2 stops flagging System.out.println
# Mutation: NSO-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: NSO-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-no-system-out: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-no-system-out "$@"
