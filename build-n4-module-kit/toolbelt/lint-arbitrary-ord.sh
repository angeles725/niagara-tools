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
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: AO2 -- require a leading quote and AO2 stops warning on BOrd.make(variable)
# Mutation: AO4 -- ignore the reviewed marker and AO4 WARNs again on the reviewed call site
# Mutation: AO6 -- accept a bare marker without a reason and AO6 stops warning
# Mutation: AO8 -- split code/comment at the first // regardless of quotes and AO8/AO9 stop warning
# Mutation: AO12 -- no block-comment skip in code_part and the apostrophe in /* it's */ hides the marker
# Mutation: AO-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: AO-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-arbitrary-ord raw 'usage: lint-arbitrary-ord.sh <src-root>' "$@"
awk_lint_list -name '*.java'

# One awk per file (was a printf|grep pipeline per source line plus a marker_state subshell per hit).
#
# marker_state(text): "reason" when the text carries the reviewed marker followed by a reason,
# "bare" when it carries the marker with nothing after it, "" when there is no marker.
#
# code_part(s): the line up to its // line comment. A "//" inside a double-quoted string or a
# single-quoted char literal is not a comment start, so it neither hides a call nor supplies a reviewed
# marker; an inline /* ... */ block comment is skipped (kept in the code so it stays a prefix of the
# line), so an apostrophe inside it opens no char literal. Block-comment state is not carried across
# lines. Sibling string-aware strippers: lint-delays.sh strip_comment (awk), lint-write-path.sh audit
# pass (awk). [polish-2026-10-02 P2a/P2b, #199 WU6a]
# shellcheck disable=SC2016  # a literal awk program: $0, $n are awk fields
awk_lint_each '
    function marker_state(text) {
        if (text ~ /lint-arbitrary-ord:[[:space:]]*reviewed[[:space:]]+[^[:space:]]/) return "reason"
        if (text ~ /lint-arbitrary-ord:[[:space:]]*reviewed/) return "bare"
        return ""
    }
    function code_part(s,    i, n, c, q, rest, e) {
        if (!index(s, "//")) return s
        if (!index(s, "\"") && !index(s, "\047") && !index(s, "/*")) return substr(s, 1, index(s, "//") - 1)
        n = length(s); q = ""
        for (i = 1; i <= n; i++) {
            c = substr(s, i, 1)
            if (q != "") {
                if (c == "\\") i++; else if (c == q) q = ""
                continue
            }
            if (c == "\"" || c == "\047") q = c
            else if (c == "/") {
                if (substr(s, i + 1, 1) == "/") return substr(s, 1, i - 1)
                if (substr(s, i + 1, 1) == "*") {
                    rest = substr(s, i + 2); e = index(rest, "*/")
                    if (!e) return s   # unclosed block comment: no // comment on this line
                    i += e + 2         # the loop i++ lands just past the */
                }
            }
        }
        return s
    }
    {
        code = code_part($0)
        # BOrd.make( <lowercase ident> ) — a bare variable, not a "literal" and not a CONST
        if (code ~ /BOrd\.make\([[:space:]]*[a-z][A-Za-z0-9_]*[[:space:]]*\)/) {
            t = $0; sub(/^[[:space:]]+/, "", t)
            st = marker_state(substr($0, length(code) + 1))
            if (st == "" && prev ~ /^[[:space:]]*\/\//) st = marker_state(prev)
            if (st == "bare")
                printf "WARN  lint-arbitrary-ord  %s:%d  reviewed marker without a reason (add one): %s\n", ENVIRON["AL_FILE"], FNR, t
            else if (st != "reason")
                printf "WARN  lint-arbitrary-ord  %s:%d  BOrd.make from a variable: %s\n", ENVIRON["AL_FILE"], FNR, t
        }
        prev = $0
    }'

awk_lint_exit 0
