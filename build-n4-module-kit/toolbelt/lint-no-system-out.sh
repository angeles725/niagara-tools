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
#   Exit:  0  no FAIL (clean) · 1  any FAIL · 3  usage/env
#
# Known limitation: a `//` inside a string literal on the same line as a System.out call
# can hide it (rare). VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# Mutation: NSO2 -- drop the System.out.print case and NSO2 stops flagging System.out.println
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-no-system-out.sh <src-root>\n' >&2
    exit 3
fi

ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-no-system-out: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

fail=0
while IFS= read -r f; do
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=${line%%//*}
        case "$code" in
            *System.out.print*|*System.err.print*)
                trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
                printf 'FAIL  lint-no-system-out  %s:%s  %s\n' "$f" "$ln" "$trimmed"
                fail=1
                ;;
        esac
    done < "$f"
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print 2>/dev/null)

exit "$fail"
