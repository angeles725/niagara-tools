#!/usr/bin/env bash
# lint-ws-placement.sh — advisory: a method that adds a freshly created component to a parent and never gives it
# a wire-sheet position (`wsAnnotation`).
#
# A component added from code with no `wsAnnotation` slot gets Workbench's in-memory diagonal stagger (+4, wrap to
# 2, width 8), so every sibling a sync action or a seeding loop creates lands on the same diagonal and overlaps
# (WsController.java:398-405). Workbench itself adds the slot only when absent, with flags 1 = Flags.READONLY,
# and otherwise sets it (WsController.java:189-200). Fix: a grid position from
# templates/ws-placement/src/com/angeles/wsplacement/WsGrid.java, added if absent with Flags.READONLY, never
# re-placed. Doctrine: types/logic-authoring.md §Wire-sheet placement, METHODOLOGY.md § rt (components).
# [ev: corpus B1172] [ev: retro ws-placement]
#
# Structural rules (comments blanked by lib/method-boundary.sh mb_strip; scope = one method from mb_parse):
#   WSP1 an `add(` call (bare `add(` or `<recv>.add(`; `addAll(`, `addLink(` … never match) with AT LEAST TWO
#        top-level arguments whose first argument is not a numeric literal (List.add(int, E) is not a slot add),
#        and whose second argument is a new component:
#          - `new B<Upper>…(` — except the value types that never get a glyph (BStatus*, BWsAnnotation, BFacets,
#            BOrd, BLink, BConversionLink, BAbsTime, BRelTime, BString, BDouble, BFloat, BInteger, BLong,
#            BBoolean, BEnumRange, BFormat, BIcon, BNameMap);
#          - `[<recv>.]create(` / `newInstance(` / `newCopy(`;
#          - a local assigned from one of the above anywhere in the same method (`BFoo f = new BFoo();`, a cast
#            `= (BComponent) create(t)` counts).
#        A leading cast on the argument is ignored. The arguments may wrap over up to 5 lines of the method.
#   Guard: the method mentions `wsAnnotation`, `BWsAnnotation` or `WsGrid` on a code line (a string literal
#        counts, a comment does not). The guard is per method: a sibling method that places nodes does not
#        silence this one.
#   One row per `add(` line. False negatives, never false WARNs on these shapes: a component carried through a
#   field, a return value or a helper method; `getInstance()`; a signature wrapped before `{` (mb_parse does not
#   see that method); a placement delegated to a helper whose name is none of the three guard words (the row
#   then shows and a reviewer confirms — it is WARN).
#
# Usage:  lint-ws-placement.sh [--strict] <src-root>...
#   Row:  WARN  lint-ws-placement  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source
#         file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: WSP1 -- dropping the creation rule passes parent.add(name, create(type)) clean
# Mutation: WSP1-local -- dropping the local tracking passes `BFoo f = new BFoo(); add("f", f);` clean
# Mutation: WSP-guard -- dropping the wsAnnotation guard WARNs a method that already places its children
# Mutation: WSP-scope -- a file-wide guard lets a sibling method's wsAnnotation silence an unplaced add
# Mutation: WSP-list -- reading the last argument of a one-argument add WARNs list.add(new BFoo())
# Mutation: WSP-value -- dropping the value-type exclusion WARNs add("st", new BStatusNumeric())
# Mutation: WSP-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: WSP-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/method-boundary.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-ws-placement each 'usage: lint-ws-placement.sh [--strict] <src-root>...' "$@"
printf '%s\n' "$MB_AWK" > "$AL_TMP/method-boundary.awk"
# shellcheck disable=SC2034  # read by awk_lint_run (lib/awk-lint.sh)
AL_AWK_ARGS=(-f "$AL_TMP/method-boundary.awk")

cat > "$AL_TMP/main.awk" << 'AWKEOF'
function trim(s) { sub(/^[[:space:]]+/, "", s); sub(/[[:space:]]+$/, "", s); return s }
# split_args(s, from): parse the argument list that opens just before index `from` of s; sets NARGS, A[1..NARGS]
# (top-level arguments, untrimmed) and returns 1 when the closing parenthesis was found
function split_args(s, from,    j, c, depth, inq, cur) {
  NARGS = 0; split("", A); depth = 0; inq = ""; cur = ""
  for (j = from; j <= length(s); j++) {
    c = substr(s, j, 1)
    if (inq != "") {
      cur = cur c
      if (c == "\\") { j++; cur = cur substr(s, j, 1); continue }
      if (c == inq) inq = ""
      continue
    }
    if (c == "\"" || c == "'") { inq = c; cur = cur c; continue }
    if (c == "(" || c == "[" || c == "{") { depth++; cur = cur c; continue }
    if (c == ")" || c == "]" || c == "}") {
      if (depth == 0) { if (trim(cur) != "" || NARGS > 0) A[++NARGS] = cur; return 1 }
      depth--; cur = cur c; continue
    }
    if (c == "," && depth == 0) { A[++NARGS] = cur; cur = ""; continue }
    cur = cur c
  }
  return 0
}
function strip_casts(a) {
  a = trim(a)
  while (a ~ /^\([A-Za-z_][A-Za-z0-9_.<>]*\)[[:space:]]*[^[:space:]]/) { sub(/^\([A-Za-z_][A-Za-z0-9_.<>]*\)[[:space:]]*/, "", a) }
  return a
}
# creates(a): what a creation expression at the START of a creates ("" when it is not one)
function creates(a,    t) {
  if (match(a, /^new[[:space:]]+B[A-Z][A-Za-z0-9_]*/)) {
    t = substr(a, RSTART, RLENGTH); sub(/^new[[:space:]]+/, "", t)
    if (t ~ /^(BStatus[A-Za-z0-9_]*|BWsAnnotation|BFacets|BOrd|BLink|BConversionLink|BAbsTime|BRelTime|BString|BDouble|BFloat|BInteger|BLong|BBoolean|BEnumRange|BFormat|BIcon|BNameMap)$/) return ""
    return "new " t "()"
  }
  if (match(a, /^([A-Za-z_][A-Za-z0-9_.]*\.)?(create|newInstance|newCopy)[[:space:]]*\(/)) {
    t = substr(a, RSTART, RLENGTH); sub(/[[:space:]]*\($/, "", t)
    return t "(...)"
  }
  return ""
}
{ raw[NR] = $0 }
END {
  n = NR
  mb_strip(raw, n, code)
  cnt = mb_parse(code, n, ms, me, mn)
  for (k = 0; k < cnt; k++) {
    guarded = 0; split("", made)
    for (i = ms[k]; i <= me[k]; i++) {
      c = code[i]
      if (c ~ /wsAnnotation|BWsAnnotation|WsGrid/) guarded = 1
      # locals assigned from a creation expression: `<name> = [casts] <creation>`
      t = c
      while (match(t, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[^=][^;]*/)) {
        s = substr(t, RSTART, RLENGTH); t = substr(t, RSTART + RLENGTH)
        match(s, /^[A-Za-z_][A-Za-z0-9_]*/); v = substr(s, 1, RLENGTH)
        e = s; sub(/^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*/, "", e)
        w = creates(strip_casts(e))
        if (w != "") made[v] = w
      }
    }
    if (guarded) continue
    for (i = ms[k]; i <= me[k]; i++) {
      c = code[i]
      joined = c
      for (j = i + 1; j <= me[k] && j <= i + 5; j++) joined = joined " " code[j]
      pos = 0; reported = 0
      while (!reported && match(substr(c, pos + 1), /(^|[^A-Za-z0-9_])add[[:space:]]*\(/)) {
        pos = pos + RSTART + RLENGTH - 1
        if (!split_args(joined, pos + 1) || NARGS < 2) continue
        if (trim(A[1]) ~ /^-?[0-9]+[lL]?$/) continue
        a2 = strip_casts(A[2]); what = creates(a2)
        if (what == "" && (a2 in made)) what = a2 " = " made[a2]
        if (what == "") continue
        printf "WARN  lint-ws-placement  %s:%d  add(%s, %s) in %s() never sets a wsAnnotation -- the wire sheet staggers it onto its siblings; add a grid position if absent with Flags.READONLY, never re-place one (types/logic-authoring.md §Wire-sheet placement)\n", FILE, i, trim(A[1]), what, mn[k]
        reported = 1
      }
    }
  }
}
AWKEOF

awk_lint_run -name '*.java'
