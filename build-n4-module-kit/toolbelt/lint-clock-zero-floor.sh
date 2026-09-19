#!/usr/bin/env bash
# lint-clock-zero-floor.sh — flags the naive Math.max(0, delay) floor near Clock.schedule.
#
# Clock / EngineManager rejects a schedule time <= 0 the SAME as a negative — so the
# common defensive floor Math.max(0L, delay) STILL throws when delay computes to 0. The
# floor must be 1 ms: Math.max(1L, delay). A self-firing timer floored at 0 silently
# dies. See types/issues-and-gotchas.md B4. [ev: retro 2026-09-19-gotchas-tier-a]
#
# Usage:  lint-clock-zero-floor.sh <src-root>
#   Only files that also reference Clock.schedule are scanned (reduces false positives).
#   Row:  WARN  lint-clock-zero-floor  <file>:<line>  zero-floor: <source>
#   Exit: 0  always (advisory) · 3  usage/env
# VCS-free by design.
# Mutation: CZF2 -- drop the Math.max(0 match and CZF2 stops warning on a zero-floor near Clock.schedule
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-clock-zero-floor.sh <src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-clock-zero-floor: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

while IFS= read -r f; do
    # only consider files that schedule on the Clock
    grep -q 'Clock\.schedule' "$f" 2>/dev/null || continue
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=${line%%//*}
        if printf '%s' "$code" | grep -Eq 'Math\.max\(\s*0[Ll]?\s*,'; then
            trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
            printf 'WARN  lint-clock-zero-floor  %s:%s  zero-floor: %s\n' "$f" "$ln" "$trimmed"
        fi
    done < "$f"
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print 2>/dev/null)

exit 0
