#!/usr/bin/env bash
# lint-size.sh — advisory size smell: a Java class whose hand-written code (slotomatic regions excluded) exceeds
# ~800 lines, or a method of a PURE class (no Baja type) that spans more than ~80 lines.
#
# A BComponent that grows past ~800 hand-written lines, or a pure control method past ~80, is where mixed concerns
# hide: the decision-logic decomposition retro traced a live defect to a monolithic step() whose counts used
# different membership rules. Size is a REVIEW QUESTION for new modules, never a hard gate, so every row is WARN.
# Doctrine: types/logic.md § Composition & organization, METHODOLOGY.md § Conformance rules (size smell); deployed
# modules are not restructured to meet it (METHODOLOGY.md § Schema / upgrade safety).
# [ev: retro decision-logic-decomposition Δ2]
#
# Structural rules (decided up front; METHODOLOGY.md § Conformance rules, heuristic parsers):
#   - Hand-written lines = non-blank lines after comments are blanked (lib/method-boundary.sh mb_strip), outside
#     every slotomatic region. A region opens on a line containing `BEGIN BAJA AUTO GENERATED CODE` and closes on
#     the next line containing `END BAJA AUTO GENERATED CODE` (the marker text slot-o-matic emits; matched on the
#     RAW line because the marker sits inside a comment). [ev: corpus B711] [ev: code docDeveloper slot-o-matic.html:421,459]
#   - A region that opens and never closes cannot be classified: WARN `unclosed-region`, and its lines count as
#     hand-written (fail closed: never shrink the count on an unparsed shape). A BEGIN inside an open region is a
#     WARN `nested-region` (the earlier region counts as hand-written); an END with no open region is a WARN
#     `stray-end`.
#   - Pure class = a file with no `@NiagaraType` annotation and no `extends B<Upper>` declaration. Only pure classes
#     get the method rule (the BComponent shell is measured by the class rule). Methods come from mb_parse (a
#     signature whose `(...)` closes on the `{` line, or a `{` alone after it; a signature wrapped over several
#     lines before `{` is not seen — a false negative, never a false WARN).
#   - One row per file for the class rule, named after the first top-level `class` declaration.
#
# Usage:  lint-size.sh [--strict] [--max-class <n>] [--max-method <n>] <src-root>
#   Defaults: --max-class 800, --max-method 80 (strictly greater than the limit WARNs).
#   Row:  WARN  lint-size  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: LSZ2 -- counting the slotomatic region lines as hand-written WARNs a small class with a large generated block
# Mutation: LSZ4 -- applying the method rule to a Baja (BComponent) class WARNs its long callback
# Mutation: LSZ5 -- dropping the unclosed-region row lets an unterminated BEGIN marker pass silently
# Mutation: LSZ8 -- ignoring a nested BEGIN lets a stray END close an unterminated region and hide its lines
# Mutation: LSZ-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: LSZ-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
LC_ALL=C
export LC_ALL

USAGE='usage: lint-size.sh [--strict] [--max-class <n>] [--max-method <n>] <src-root>'
STRICT=0
MAXC=800
MAXM=80
is_count() { case "$1" in ''|*[!0-9]*) return 1 ;; *) return 0 ;; esac; }
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --max-class|--max-method)
      if [ $# -lt 2 ] || ! is_count "$2"; then
        printf 'lint-size: %s needs a non-negative integer\n%s\n' "$1" "$USAGE" >&2
        exit 3
      fi
      if [ "$1" = --max-class ]; then MAXC="$2"; else MAXM="$2"; fi
      shift 2 ;;
    --) shift; break ;;
    -*) printf 'lint-size: unknown flag: %s\n%s\n' "$1" "$USAGE" >&2; exit 3 ;;
    *) break ;;
  esac
done
if [ $# -ne 1 ]; then
  printf '%s\n' "$USAGE" >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-size: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"

cat > "$_TMP/main.awk" << 'AWKEOF'
{ raw[NR] = $0 }
END {
  n = NR
  mb_strip(raw, n, code)
  # Pass 1: slotomatic regions on the RAW lines (the markers live inside comments).
  in_gen = 0; open_line = 0
  for (i = 1; i <= n; i++) {
    gen[i] = 0
    is_begin = index(raw[i], "BEGIN BAJA AUTO GENERATED CODE") > 0
    is_end = index(raw[i], "END BAJA AUTO GENERATED CODE") > 0
    if (is_begin) {
      if (in_gen) {
        # A BEGIN inside an open region: the earlier region never closed, so count it (fail closed).
        for (k = open_line; k < i; k++) gen[k] = 0
        printf "WARN  lint-size  %s:%d  nested-region: BEGIN BAJA AUTO GENERATED CODE inside the region opened at line %d -- that region is counted as hand-written\n", FILE, i, open_line
      }
      in_gen = 1; open_line = i; gen[i] = 1; continue
    }
    if (is_end && !in_gen) {
      printf "WARN  lint-size  %s:%d  stray-end: END BAJA AUTO GENERATED CODE with no open region\n", FILE, i
      continue
    }
    if (in_gen) {
      gen[i] = 1
      if (is_end) in_gen = 0
    }
  }
  if (in_gen) {
    # Unclassifiable: count the unterminated region as hand-written and say so.
    for (i = open_line; i <= n; i++) gen[i] = 0
    printf "WARN  lint-size  %s:%d  unclosed-region: BEGIN BAJA AUTO GENERATED CODE with no END marker -- its lines are counted as hand-written\n", FILE, open_line
  }
  # Pass 2: hand-written line count, class name, pure-class test.
  hand = 0; cname = ""; cline = 0; baja = 0
  for (i = 1; i <= n; i++) {
    c = code[i]
    if (index(c, "@NiagaraType") > 0) baja = 1
    if (match(c, /extends[[:space:]]+B[A-Z][A-Za-z0-9_]*/)) baja = 1
    if (cname == "" && match(c, /(^|[[:space:]])class[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
      s = substr(c, RSTART, RLENGTH); sub(/^[[:space:]]*class[[:space:]]+/, "", s)
      cname = s; cline = i
    }
    if (gen[i]) continue
    t = c; gsub(/[[:space:]]/, "", t)
    if (t != "") hand++
  }
  if (cname == "") { cname = "?"; cline = 1 }
  if (hand > MAXC)
    printf "WARN  lint-size  %s:%d  class %s has %d hand-written lines (> %d, slotomatic regions excluded) -- split by concern into pure classes / child components (types/logic.md § Composition)\n", FILE, cline, cname, hand, MAXC
  if (baja) exit 0
  # Pass 3: method spans of a pure class.
  cnt = mb_parse(code, n, ms, me, mn)
  for (k = 0; k < cnt; k++) {
    span = me[k] - ms[k] + 1
    if (span > MAXM)
      printf "WARN  lint-size  %s:%d  method %s() in pure class %s spans %d lines (> %d) -- split into named phase methods (types/logic.md § Pure-class extraction)\n", FILE, ms[k], mn[k], cname, span, MAXM
  }
}
AWKEOF

had_warn=0
had_err=0
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.java'; then
  printf 'lint-size: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  had_err=1
fi
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -v MAXC="$MAXC" -v MAXM="$MAXM" -f "$_TMP/method-boundary.awk" -f "$_TMP/main.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-size: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
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
