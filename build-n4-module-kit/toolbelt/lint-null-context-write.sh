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
# of permissions (BOrd.java:90-107 pass null; corpus B1237 §1237.7 rule 4). The statement is read from the line
# holding the `bql:` literal up to its `;`; a query string held in a variable is not followed. -wb parts are
# skipped (client code, the session carries the user). See types/logic-authoring.md § BQL from code.
# [ev: retro n4-course-drivers-bql-ords-files Δ1]
# Java is read with comments stripped and string/char literals skipped (a "//" or "/*" in a string is not a
# comment); writes are matched with literal contents blanked, the bql: literal with them kept.
#
# Usage:  lint-null-context-write.sh <src-root>
#   Row:  WARN  lint-null-context-write  <file>:<line>  null-Context write: <source>
#   Row:  WARN  lint-null-context-write  <file>:<line>  bql: ORD resolved without the caller's Context: <source>
#   Exit: 0  always (advisory) · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: NCW2 -- drop the ,null match and NCW2 stops warning on set(...,null)
# Mutation: NCW-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: NCW-bql1 -- dropping the one-argument get/resolve match passes get(Sys.getStation()) clean
# Mutation: NCW-bql2 -- checking only the bql: line misses .get(base, null) on a later line
# Mutation: NCW-literal -- stop skipping string literals and "http://" hides the write after it
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-null-context-write raw 'usage: lint-null-context-write.sh <src-root>' "$@"

cat > "$AL_TMP/main.awk" << 'AWKEOF'
# one line with /* */ and // comments removed (`inblk` carries an open block comment): `keep` keeps literal
# contents, `blank` blanks them (\ escapes honoured). Same scanner as lint-driver-scaffold.sh.
function strip(s,   i, n, c, d, q, j) {
  keep = ""; blank = ""; n = length(s); i = 1
  while (i <= n) {
    c = substr(s, i, 1); d = substr(s, i, 2)
    if (inblk) { if (d == "*/") { inblk = 0; i += 2 } else i++; continue }
    if (d == "//") return
    if (d == "/*") { inblk = 1; i += 2; continue }
    if (c == "\"" || c == "'") {
      q = c; j = i + 1
      while (j <= n && substr(s, j, 1) != q) j += (substr(s, j, 1) == "\\") ? 2 : 1
      keep = keep substr(s, i, j - i + 1); blank = blank q q; i = j + 1
      continue
    }
    keep = keep c; blank = blank c; i++
  }
}
function src(s) { sub(/^[[:space:]]+/, "", s); return s }
FILE ~ /-wb\// { exit }
{
  strip($0)
  if (blank ~ /\.(set|setInt|setBoolean|setDouble|invoke)\(.*,[[:space:]]*null[[:space:]]*\)/)
    printf "WARN  lint-null-context-write  %s:%d  null-Context write: %s\n", FILE, FNR, src($0)
  if (stmt == "" && keep ~ /"[^"]*bql:/) { stmt = " "; sline = FNR; stext = src($0) }
  if (stmt != "") {
    stmt = stmt " " blank
    if (index(blank, ";")) {
      if (stmt ~ /\.(get|resolve)\(([^(),]|\([^()]*\))+\)/ || stmt ~ /"[[:space:]]*\)[[:space:]]*\.(get|resolve)\([[:space:]]*\)/ \
          || stmt ~ /\.(get|resolve)\([^;]*,[[:space:]]*null[[:space:]]*\)/)
        printf "WARN  lint-null-context-write  %s:%d  bql: ORD resolved without the caller's Context (pass get(base, cx)): %s\n", FILE, sline, stext
      stmt = ""
    }
  }
}
AWKEOF

awk_lint_run -name '*.java'
