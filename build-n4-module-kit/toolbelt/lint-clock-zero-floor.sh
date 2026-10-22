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
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: CZF2 -- drop the Math.max(0 match and CZF2 stops warning on a zero-floor near Clock.schedule
# Mutation: CZF-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: CZF-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-clock-zero-floor raw 'usage: lint-clock-zero-floor.sh <src-root>' "$@"
awk_lint_list -name '*.java'

# One awk per file (was a grep per file + a printf|grep per source line). Hits are held until END
# because only a file that references Clock.schedule anywhere (comments included) is reported.
# shellcheck disable=SC2016  # a literal awk program: $0, $n are awk fields
awk_lint_each '
    /Clock\.schedule/ { sched = 1 }
    { code = $0; i = index(code, "//"); if (i) code = substr(code, 1, i - 1) }
    code ~ /Math\.max\([[:space:]]*0[Ll]?[[:space:]]*,/ {
        t = $0; sub(/^[[:space:]]+/, "", t)
        rows = rows sprintf("WARN  lint-clock-zero-floor  %s:%d  zero-floor: %s\n", ENVIRON["AL_FILE"], FNR, t)
    }
    END { if (sched) printf "%s", rows }'

awk_lint_exit 0
