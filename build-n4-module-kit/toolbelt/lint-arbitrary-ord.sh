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
#   Reviewed call site: `// lint-arbitrary-ord: reviewed <reason>` on the same line or the line
#   directly above suppresses that one WARN; the reason is mandatory — a bare marker still WARNs
#   ("reviewed marker without a reason"). [ev: retro panccadia-restart-seq-comp-lockout-hours Δ4]
#   The marker counts only in a real // comment: a "//" inside a string or char literal is code.
#   Exit: 0  always (advisory) · 3  usage/env
# VCS-free by design.
# Mutation: AO2 -- require a leading quote and AO2 stops warning on BOrd.make(variable)
# Mutation: AO4 -- ignore the reviewed marker and AO4 WARNs again on the reviewed call site
# Mutation: AO6 -- accept a bare marker without a reason and AO6 stops warning
# Mutation: AO8 -- split code/comment at the first // regardless of quotes and AO8/AO9 stop warning
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

MARKER='lint-arbitrary-ord:[[:space:]]*reviewed'

# marker_state <text>: "reason" when the text carries the reviewed marker followed by a reason,
# "bare" when it carries the marker with nothing after it, "" when there is no marker.
marker_state() {
    if printf '%s' "$1" | grep -Eq "${MARKER}[[:space:]]+[^[:space:]]"; then
        printf 'reason'
    elif printf '%s' "$1" | grep -Eq "$MARKER"; then
        printf 'bare'
    fi
}

# code_part <line>: the line up to its // line comment. A "//" inside a double-quoted string or a
# single-quoted char literal is not a comment start, so it neither hides a call nor supplies a
# reviewed marker. [polish-2026-10-02 P2a, #199 WU6a]
code_part() {
    local s="$1" i c q="" n
    case "$s" in *//*) ;; *) printf '%s' "$s"; return ;; esac
    case "$s" in *\"*|*\'*) ;; *) printf '%s' "${s%%//*}"; return ;; esac
    n=${#s}
    for ((i = 0; i < n; i++)); do
        c=${s:i:1}
        if [ -n "$q" ]; then
            if [ "$c" = "\\" ]; then i=$((i + 1)); elif [ "$c" = "$q" ]; then q=""; fi
            continue
        fi
        case "$c" in
            \"|\') q=$c ;;
            /) if [ "${s:i+1:1}" = "/" ]; then printf '%s' "${s:0:i}"; return; fi ;;
        esac
    done
    printf '%s' "$s"
}

while IFS= read -r f; do
    ln=0
    prev=""
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=$(code_part "$line")
        # BOrd.make( <lowercase ident> ) — a bare variable, not a "literal" and not a CONST
        if printf '%s' "$code" | grep -Eq 'BOrd\.make\([[:space:]]*[a-z][A-Za-z0-9_]*[[:space:]]*\)'; then
            trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
            st=$(marker_state "${line#"$code"}")
            if [ -z "$st" ] && printf '%s' "$prev" | grep -Eq '^[[:space:]]*//'; then
                st=$(marker_state "$prev")
            fi
            case "$st" in
                reason) : ;;
                bare) printf 'WARN  lint-arbitrary-ord  %s:%s  reviewed marker without a reason (add one): %s\n' "$f" "$ln" "$trimmed" ;;
                *) printf 'WARN  lint-arbitrary-ord  %s:%s  BOrd.make from a variable: %s\n' "$f" "$ln" "$trimmed" ;;
            esac
        fi
        prev=$line
    done < "$f"
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print 2>/dev/null)

exit 0
