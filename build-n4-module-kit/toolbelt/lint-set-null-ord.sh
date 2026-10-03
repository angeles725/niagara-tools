#!/usr/bin/env bash
# lint-set-null-ord.sh — flags a getSlotPathOrd() result passed to a setter / set() with no null guard: the ORD is
# null on an unmounted component and BComplex.set NPEs on a null value.
#
# BComponent.getSlotPathOrd() returns null when getSlotPath() is null, i.e. the component is not mounted in a
# component space yet (a point a -wb manager just created). Passed to a BOrd property setter it reaches
# BComplex.set(Property, BValue, Context), which dereferences value.getSlotMap() and throws a NullPointerException
# from inside the framework. Fix: guard `if (ord == null || ord.isNull())`, build the ORD from the resolved parent
# folder + slot name, mount the component first. See types/issues-and-gotchas.md §H3.
# [ev: retro wb-mapping-ord-npe-and-wsl-windows-jdk Δ1]
# [ev: code javax/baja/sys/BComponent.java:321-324] [ev: code com/tridium/sys/schema/ComponentSlotMap.java:171-173]
# [ev: code javax/baja/sys/BComplex.java:386-390]
#
# Structural rules (comments blanked by lib/method-boundary.sh mb_strip; scope = one method from mb_parse):
#   SNO1 direct  — `set<Name>(… x.getSlotPathOrd())` / `set(<prop>, x.getSlotPathOrd(), …)`: the getSlotPathOrd()
#                  call is itself an argument (followed by `,` or `)`). Guarded when the call line, or an earlier
#                  line of the same method, compares the SAME receiver's getSlotPathOrd() with null (`!= null` /
#                  `== null`); a check on `a.` never guards `b.getSlotPathOrd()`.
#   SNO2 local   — `[BOrd] v = <expr>getSlotPathOrd();` then a later `set…(…v…)` in the same method, with no
#                  `v == null`, `v != null`, `null == v`, `null != v`, `v.isNull()` or `requireNonNull(v` on any
#                  line from the assignment to the call (a ternary on the call line counts).
#   Scope limits (false negatives, never false WARNs): a value carried through a field, a return value or another
#   method; a signature wrapped over several lines before `{` (mb_parse does not see that method).
#   Not overlapping lint-null-context-write: that lint flags a null CONTEXT argument, this one a null VALUE.
#
# Usage:  lint-set-null-ord.sh [--strict] <src-root>
#   Row:  WARN  lint-set-null-ord  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: SNO1 -- dropping the direct-argument rule passes setTargetOrd(point.getSlotPathOrd()) clean
# Mutation: SNO1-recv -- a receiver-agnostic guard silences b.getSlotPathOrd() after a check on a
# Mutation: SNO2-guard -- ignoring the null guard WARNs a local that is checked before the set
# Mutation: SNO2-scope -- matching the variable outside its own method WARNs a same-named local in a sibling method
# Mutation: SNO-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: SNO-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
LC_ALL=C
export LC_ALL

USAGE='usage: lint-set-null-ord.sh [--strict] <src-root>'
STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-set-null-ord: unknown flag: %s\n%s\n' "$1" "$USAGE" >&2; exit 3 ;;
    *) break ;;
  esac
done
if [ $# -ne 1 ]; then
  printf '%s\n' "$USAGE" >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-set-null-ord: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"

cat > "$_TMP/main.awk" << 'AWKEOF'
function setter_name(s,    t) {
  # the LEFTMOST `set<Name>(` on the line (a chained or nested setter line names its first setter)
  if (match(s, /(^|[^A-Za-z0-9_])set[A-Za-z0-9_]*[[:space:]]*\(/)) {
    t = substr(s, RSTART, RLENGTH); sub(/^[^s]*/, "", t); sub(/[[:space:]]*\($/, "", t)
    return t
  }
  return ""
}
# guard_recvs(s): record in guarded[] every receiver (`a.`, `x.y.`, or "" for this) whose getSlotPathOrd() result
# is compared with null on line s
function guard_recvs(s,    t, r) {
  t = s
  while (match(t, /[A-Za-z0-9_.]*getSlotPathOrd\(\)[[:space:]]*[!=]=[[:space:]]*null/)) {
    r = substr(t, RSTART, RLENGTH); sub(/getSlotPathOrd\(\).*$/, "", r); guarded[r] = 1
    t = substr(t, RSTART + RLENGTH)
  }
  t = s
  while (match(t, /null[[:space:]]*[!=]=[[:space:]]*[A-Za-z0-9_.]*getSlotPathOrd\(\)/)) {
    r = substr(t, RSTART, RLENGTH); sub(/^null[[:space:]]*[!=]=[[:space:]]*/, "", r); sub(/getSlotPathOrd\(\)$/, "", r)
    guarded[r] = 1
    t = substr(t, RSTART + RLENGTH)
  }
}
# unguarded_direct(s): 1 when some `<recv>getSlotPathOrd()` argument on s (followed by `,` or `)`) has a receiver
# that no null compare in this method (or on this line) has guarded
function unguarded_direct(s,    t, r) {
  t = s
  while (match(t, /[A-Za-z0-9_.]*getSlotPathOrd\(\)[[:space:]]*[,)]/)) {
    r = substr(t, RSTART, RLENGTH); sub(/getSlotPathOrd\(\).*$/, "", r)
    if (!(r in guarded)) return 1
    t = substr(t, RSTART + RLENGTH)
  }
  return 0
}
function guards_var(s, v,    re) {
  re = "(^|[^A-Za-z0-9_])" v "[[:space:]]*[!=]=[[:space:]]*null"
  if (s ~ re) return 1
  re = "null[[:space:]]*[!=]=[[:space:]]*" v "([^A-Za-z0-9_]|$)"
  if (s ~ re) return 1
  re = "(^|[^A-Za-z0-9_])" v "\\.isNull\\("
  if (s ~ re) return 1
  re = "requireNonNull\\([[:space:]]*" v "([^A-Za-z0-9_]|$)"
  if (s ~ re) return 1
  return 0
}
{ raw[NR] = $0 }
END {
  n = NR
  mb_strip(raw, n, code)
  cnt = mb_parse(code, n, ms, me, mn)
  for (k = 0; k < cnt; k++) {
    nv = 0; split("", guarded)
    for (i = ms[k]; i <= me[k]; i++) {
      c = code[i]
      guard_recvs(c)
      # SNO1: getSlotPathOrd() is itself the argument of a set...( call on this line; guarded per receiver
      if (c ~ /(^|[^A-Za-z0-9_])set[A-Za-z0-9_]*[[:space:]]*\([^;]*getSlotPathOrd\(\)[[:space:]]*[,)]/) {
        if (unguarded_direct(c))
          printf "WARN  lint-set-null-ord  %s:%d  getSlotPathOrd() passed straight to %s(...) -- null on an unmounted component and BComplex.set NPEs; assign it, guard `if (ord == null || ord.isNull())` (types/issues-and-gotchas.md §H3)\n", FILE, i, setter_name(c)
      }
      # SNO2: a local assigned from getSlotPathOrd(), then used as a set argument
      if (match(c, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*[^=;][^;]*getSlotPathOrd\(\)[[:space:]]*;/)) {
        s = substr(c, RSTART, RLENGTH); match(s, /^[A-Za-z_][A-Za-z0-9_]*/)
        nv++; vname[nv] = substr(s, 1, RLENGTH); vline[nv] = i; vguard[nv] = 0
        continue
      }
      for (j = 1; j <= nv; j++) {
        if (vguard[j]) continue
        if (guards_var(c, vname[j])) { vguard[j] = 1; continue }
        re = "(^|[^A-Za-z0-9_])set[A-Za-z0-9_]*[[:space:]]*\\(([^;]*[,[:space:]])?" vname[j] "[[:space:]]*[,)]"
        if (c ~ re) {
          printf "WARN  lint-set-null-ord  %s:%d  '%s' (getSlotPathOrd() at line %d) reaches %s(...) with no null guard -- null on an unmounted component and BComplex.set NPEs; guard `if (%s == null || %s.isNull())` (types/issues-and-gotchas.md §H3)\n", FILE, i, vname[j], vline[j], setter_name(c), vname[j], vname[j]
          vguard[j] = 1
        }
      }
    }
  }
}
AWKEOF

had_warn=0
had_err=0
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.java'; then
  printf 'lint-set-null-ord: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  had_err=1
fi
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -f "$_TMP/method-boundary.awk" -f "$_TMP/main.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-set-null-ord: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
    had_err=1
    continue
  fi
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    had_warn=1
  fi
done < "$_TMP/files"

[ "$had_err" -eq 0 ] || exit 3
if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
