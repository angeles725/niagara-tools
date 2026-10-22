#!/usr/bin/env bash
# lint-null-context-write.sh — flags component writes/invokes made with a null Context.
#
# A write with a null Context — `parent.set(prop, val, null)` or `comp.invoke(act, arg, null)` —
# is a DOUBLE hazard: (1) it skips the AuditHistory record (no who-changed-what), and (2) it
# grants BPermissions.all, bypassing RBAC. A servlet/dashboard write must carry the invoking
# user's Context: `new BasicContext(BUser.getUserFromSubject(...))`. See types/security.md §1.
# Advisory (WARN): some framework-internal writes legitimately pass null. [ev: retro 2026-09-19-security-model]
#
# A bql: ORD resolved with no caller Context — `BOrd.make("…bql:…").get(base)`, `.resolve(base)`, a bare
# `.get()`/`.resolve()` or a `null` Context — runs the query with no user, so it sees every component regardless
# of permissions (BOrd.java:90-107 pass null; corpus B1237 §1237.7 rule 4). The window starts at the string
# literal holding `bql:` and ends at the first `;` or `{`; only the get/resolve called right on that ORD
# (`"…bql:…" [+ …]) .get(…)`) is judged, so another one-argument call in the statement is not. A query string
# held in a variable is not followed (owed Go lint G4). -wb parts are skipped by this rule only (client code,
# the session carries the user); the null-Context write rule scans every part.
# See types/logic-authoring.md § BQL from code. [ev: retro n4-course-drivers-bql-ords-files Δ1]
# Java is read through lib/method-boundary.sh `mb_strip` (comments out, a // or /* inside a literal is code),
# then literal contents are blanked for matching; the bql literal is found before blanking.
#
# Usage:  lint-null-context-write.sh <src-root>
#   Row:  WARN  lint-null-context-write  <file>:<line>  null-Context write: <source>
#   Row:  WARN  lint-null-context-write  <file>:<line>  bql: ORD resolved without the caller's Context (pass get(base, cx)): <source>
#   Exit: 0  always (advisory) · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: NCW2 -- drop the ,null match and NCW2 stops warning on set(...,null)
# Mutation: NCW-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: NCW-bql1 -- dropping the one-argument get/resolve match passes get(Sys.getStation()) clean
# Mutation: NCW-bql2 -- checking only the bql: line misses .get(base, null) on a later line
# Mutation: NCW-literal -- matching the unblanked mb_strip text WARNs on a write quoted inside a string literal
# Mutation: NCW-wb-write -- skipping -wb files for both rules drops the -wb null-Context write row
# Mutation: NCW-bql-unrelated -- matching a one-argument get( anywhere in the window flags params.get("x") beside a clean get(base, cx)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
awk_lint_parse lint-null-context-write raw 'usage: lint-null-context-write.sh <src-root>' "$@"
printf '%s\n' "$MB_AWK" > "$AL_TMP/method-boundary.awk"
# shellcheck disable=SC2034  # read by awk_lint_run (lib/awk-lint.sh)
AL_AWK_ARGS=(-f "$AL_TMP/method-boundary.awk")

cat > "$AL_TMP/main.awk" << 'AWKEOF'
# dst = src with the contents of "...", '...' and """...""" literals blanked (quotes kept, \ escapes honoured,
# text-block state carried across lines), as lint-driver-scaffold does after mb_strip. bcol[i] = column in
# dst[i] of the opening quote of the first literal on line i whose text holds "bql:" (0 when none).
function blank(src, n, dst,   i, ln, out, j, c, q, tb, lit, at) {
  for (i = 1; i <= n; i++) {
    ln = src[i]; out = ""; q = ""; bcol[i] = 0
    for (j = 1; j <= length(ln); j++) {
      c = substr(ln, j, 1)
      if (tb) { if (c == "\\") j++; else if (substr(ln, j, 3) == "\"\"\"") { out = out "\"\"\""; tb = 0; j += 2 } }
      else if (q != "") {
        if (c == "\\") { lit = lit substr(ln, j, 2); j++ }
        else if (c == q) { out = out q; if (q == "\"" && !bcol[i] && index(lit, "bql:")) bcol[i] = at; q = "" }
        else lit = lit c
      }
      else if (substr(ln, j, 3) == "\"\"\"") { out = out "\"\"\""; tb = 1; j += 2 }
      else { if (c == "\"" || c == "\047") { q = c; lit = ""; at = length(out) + 1 } out = out c }
    }
    dst[i] = out
  }
}
function src(s) { sub(/^[[:space:]]+/, "", s); return s }
{ raw[NR] = $0 }
END {
  mb_strip(raw, NR, keep); blank(keep, NR, code)
  for (i = 1; i <= NR; i++)
    if (code[i] ~ /\.(set|setInt|setBoolean|setDouble|invoke)\(.*,[[:space:]]*null[[:space:]]*\)/)
      printf "WARN  lint-null-context-write  %s:%d  null-Context write: %s\n", FILE, i, src(raw[i])
  if (FILE ~ /-wb\//) exit
  for (i = 1; i <= NR; i++) {
    if (!bcol[i]) continue
    w = substr(code[i], bcol[i]); k = i
    while (!match(w, /[;{]/) && k < NR) w = w " " code[++k]
    if (match(w, /[;{]/)) w = substr(w, 1, RSTART - 1)
    # the ORD literal, an optional paren-free concatenation, make's ")", then the call on that ORD
    if (!match(w, /^""[^()]*\)[[:space:]]*\.(get|resolve)\(/)) continue
    args = substr(w, RSTART + RLENGTH)
    if (args ~ /^[[:space:]]*\)/ || args ~ /^([^(),]|\([^()]*\))+\)/ || args ~ /^[^;]*,[[:space:]]*null[[:space:]]*\)/)
      printf "WARN  lint-null-context-write  %s:%d  bql: ORD resolved without the caller's Context (pass get(base, cx)): %s\n", FILE, i, src(raw[i])
  }
}
AWKEOF

awk_lint_run -name '*.java'
