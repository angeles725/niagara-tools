#!/usr/bin/env bash
# lint-arbitrary-ord.sh — flags BOrd.make() resolved from a non-literal (client-supplied) value.
#
# BOrd.make(someVariable) that comes from client input lets a caller resolve ANY station ORD
# (e.g. a foreign service, a credential store). Validate/allowlist a client-supplied ORD prefix
# before resolving. See types/security.md §4. Advisory (WARN), FP-prone: many BOrd.make(var)
# calls take a trusted internal value. [ev: retro 2026-09-19-security-model]
#
# Usage:  lint-arbitrary-ord.sh <src-root>
#   Flags BOrd.make(<lowercase-identifier>) — skips string literals and UPPER_CASE constants.
#   Row:  WARN  lint-arbitrary-ord  <file>:<line>  BOrd.make from a variable: <source>
#   Exit: 0  always (advisory) · 3  usage/env
# VCS-free by design.
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-arbitrary-ord.sh <src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-arbitrary-ord: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

while IFS= read -r f; do
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=${line%%//*}
        # BOrd.make( <lowercase ident> ) — a bare variable, not a "literal" and not a CONST
        if printf '%s' "$code" | grep -Eq 'BOrd\.make\([[:space:]]*[a-z][A-Za-z0-9_]*[[:space:]]*\)'; then
            trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
            printf 'WARN  lint-arbitrary-ord  %s:%s  BOrd.make from a variable: %s\n' "$f" "$ln" "$trimmed"
        fi
    done < "$f"
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print 2>/dev/null)

exit 0
