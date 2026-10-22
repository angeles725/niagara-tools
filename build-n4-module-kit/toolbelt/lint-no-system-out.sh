#!/usr/bin/env bash
# lint-no-system-out.sh — flags System.out / System.err print calls in module Java.
#
# System.out is captured by StdoutManager -> stdout.txt / spy:/stdout; it bypasses
# java.util.logging entirely: no logger name, no severity classification, and no entry
# in BLogHistoryService (so it is NOT BQL-queryable). A station operator never sees it in
# the Logging service. Module diagnostics MUST use java.util.logging.Logger.
# See types/observability.md §1. [ev: retro 2026-09-19-observability]
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
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-no-system-out raw 'usage: lint-no-system-out.sh <src-root>' "$@"
awk_lint_list -name '*.java'

# One awk per file (was a shell read loop + a sed fork per hit). Line // comments are stripped first.
# shellcheck disable=SC2016  # a literal awk program: $0, $n are awk fields
awk_lint_each '
    { code = $0; i = index(code, "//"); if (i) code = substr(code, 1, i - 1) }
    index(code, "System.out.print") || index(code, "System.err.print") {
        t = $0; sub(/^[[:space:]]+/, "", t)
        printf "FAIL  lint-no-system-out  %s:%d  %s\n", ENVIRON["AL_FILE"], FNR, t
    }'

awk_lint_exit "$AL_ROWS"
