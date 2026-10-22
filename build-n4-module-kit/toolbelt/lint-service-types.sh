#!/usr/bin/env bash
# lint-service-types.sh — flags a service whose getServiceTypes() registers no type (an empty array or null)
# and WARNs when the returned array does not hold the service's own TYPE.
#
# ServiceManager.register iterates exactly the array getServiceTypes() returns: an empty array or null registers
# the service under no type, so Sys.getService(Type) never finds it. IntelliJ's "Implement methods" stub returns
# `new Type[0]`, which is how it ships by accident. Real services return `new Type[] { TYPE }` (BAlarmService) or
# add their interface type (`{ TYPE, BILoggingService.TYPE }`). See METHODOLOGY.md § rt.
# [ev: corpus B1419 §1419.1] [ev: code com/tridium/sys/service/ServiceManager.java:93-99]
#
# Structural rules (comments blanked by lib/method-boundary.sh mb_strip):
#   Scope     — a `Type[] getServiceTypes(...)` declaration with a body; an abstract/interface declaration
#               (`;` before `{`) and a file without one are out of scope.
#   Returned  — every `return <expr>;` of the body, the worst verdict wins (empty > other > unresolved > own).
#               A return inside a nested scope — a block lambda, an anonymous class, a local class — belongs to
#               that scope, not to getServiceTypes(), and is skipped; a switch-rule block (`case A, B -> {`,
#               `default -> {`, a guarded `case T t when (…) -> {`) is not one, a lambda on a case line is.
#               String/char/text-block contents are blanked first, so no literal opens a scope or holds a return.
#               <expr> is an array literal (`new Type[] { … }`), `null`, `new Type[0]`, or a field (`a`, `Own.a`,
#               `(Type[]) a.clone()`) resolved through every `a = …;` assignment in the same file (a static
#               initializer counts), read from the assignment itself, not from the start of its line.
#   FAIL      — the returned array is empty (`new Type[0]`, `new Type[] {}`, `{}`) or null.
#   Own type  — `TYPE`, `this.TYPE`, `<OwnClass>.TYPE` or `getType()`/`this.getType()` (own class = first `class X`
#               in the file); `<OwnClass>.<field>` resolves like a bare field; outer parentheses are dropped.
#   WARN      — the array holds no own type; or the return cannot be resolved in this file (a super call, an
#               inherited field, a method).
#
# Usage:  lint-service-types.sh [--strict] <src-root>
#   Row:  FAIL|WARN  lint-service-types  <file>:<line>  <detail>     (<line> = the getServiceTypes declaration)
#   Exit: 0  no FAIL (WARN only without --strict) · 1  any FAIL, or any WARN under --strict · 3  usage/env, an
#         unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: ST1 -- dropping the empty-array check passes the IntelliJ 'Implement methods' stub new Type[0]
# Mutation: ST2 -- requiring the array on the return line flags a multi-line initializer that holds TYPE
# Mutation: ST3 -- not resolving a returned field passes a serviceTypes field initialized to new Type[0]
# Mutation: ST4 -- accepting any X.TYPE as the own type passes a service registered only under another type
# Mutation: ST5 -- treating an unresolved return as clean passes return super.getServiceTypes()
# Mutation: ST9 -- resolving only the declaration line misses a field assigned in a static block to new Type[0]
# Mutation: ST10 -- checking only the first return passes a getServiceTypes() whose fallback branch returns null
# Mutation: ST11 -- reading the initializer from the line start stops at an earlier ';' and misses new Type[0]
# Mutation: ST8 -- dropping getType() from the own-type set WARNs the vendor form new Type[] { this.getType() }
# Mutation: ST12 -- counting nested-scope returns FAILs a lambda that returns null inside a method returning { TYPE }
# Mutation: ST13 -- ranking other below unresolved reports a foreign-only array as merely unresolved
# Mutation: ST14 -- classifying any '->' on a case line as a switch rule FAILs a lambda's return null on that line
# Mutation: ST15 -- reading literal contents lets "class A" in a condition hide the method's return null
# Mutation: ST6 -- matching inside comments flags a commented-out return new Type[0]
# Mutation: ST-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: ST-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-service-types one 'usage: lint-service-types.sh [--strict] <src-root>' "$@"
printf '%s\n' "$MB_AWK" > "$AL_TMP/method-boundary.awk"

cat > "$AL_TMP/main.awk" << 'AWKEOF'
# raw text from column `col` of line `from` up to (not including) the first `stop` character, across lines
function until_char(from, col, n, stop,    i, j, ch, out) {
  out = ""
  for (i = from; i <= n; i++) {
    for (j = (i == from ? col : 1); j <= length(code[i]); j++) {
      ch = substr(code[i], j, 1)
      if (ch == stop) return out
      out = out ch
    }
  }
  return out
}
# 1 when `seg` (the statement text before a `->` that precedes `{`) is a switch-rule label: `default`, or `case …`
# whose labels hold no other `->` (that is a lambda on a case line), no assignment `=` (an old `case X:` label then
# `r = () -> {`) and balanced parentheses (`case X: run(() -> {`); a guard (`case T t when (…)`) is a label
function switch_rule(seg,    t, k, d, ch) {
  sub(/^[[:space:]]+/, "", seg); sub(/[[:space:]]*->$/, "", seg)
  if (seg == "default") return 1
  if (seg !~ /^case[[:space:]]/ || index(seg, "->")) return 0
  t = seg; gsub(/==|!=|<=|>=/, "", t); if (index(t, "=")) return 0
  d = 0
  for (k = 1; k <= length(seg); k++) {
    ch = substr(seg, k, 1)
    if (ch == "(") d++
    else if (ch == ")" && --d < 0) return 0
  }
  return d == 0
}
# 1 when the `{` that follows text `pre` (literal contents already blanked by block_body) opens a nested scope
# whose returns are not the method's: a block lambda (`-> {` that is not a switch rule), an anonymous class
# (`new X(…) {`) or a local class/interface/enum
function nested_opener(pre,    seg, k, d, ch) {
  sub(/[[:space:]]+$/, "", pre)
  seg = pre; sub(/^.*[;{}]/, "", seg)
  if (pre ~ /->$/) return !switch_rule(seg)
  if (seg ~ /(^|[^A-Za-z0-9_])(class|interface|enum)[[:space:]]+[A-Za-z_]/) return 1
  if (pre !~ /\)$/) return 0
  d = 0
  for (k = length(pre); k > 0; k--) {
    ch = substr(pre, k, 1)
    if (ch == ")") d++
    else if (ch == "(" && --d == 0) break
  }
  return (substr(pre, 1, k - 1) ~ /(^|[^A-Za-z0-9_])new[[:space:]]+[A-Za-z_][A-Za-z0-9_.]*[[:space:]]*(<[^;{}()]*>)?[[:space:]]*$/)
}
# body text of the brace block that opens on line `from`; a nested scope (nested_opener) is replaced by `{}` so its
# returns never count as the method's. mb_strip keeps literal contents, so here every character inside a "…", '…'
# or """…""" literal becomes `_`: no brace, arrow, keyword or `return` is read from a literal. An unterminated
# "…"/'…' ends at its line end (as in mb_strip); a text block carries across lines.
function block_body(from, n,    i, j, ch, d, started, out, skip, q, tb) {
  d = 0; started = 0; out = ""; skip = 0; tb = 0
  for (i = from; i <= n; i++) {
    q = ""
    for (j = 1; j <= length(code[i]); j++) {
      ch = substr(code[i], j, 1)
      if (tb || q != "") {
        if (ch == "\\") { j++; ch = "__" }
        else if (tb && substr(code[i], j, 3) == "\"\"\"") { tb = 0; j += 2; ch = "\"\"\"" }
        else if (!tb && ch == q) q = ""
        else ch = "_"
      } else if (substr(code[i], j, 3) == "\"\"\"") { tb = 1; j += 2; ch = "\"\"\"" }
      else if (ch == "\"" || ch == "\047") q = ch
      else if (skip) {
        if (ch == "{") skip++
        else if (ch == "}" && --skip == 0) out = out "{}"
        continue
      }
      else if (ch == "{") {
        if (!started) { d = 1; started = 1; continue }
        if (nested_opener(out)) { skip = 1; continue }
        d++
      }
      else if (ch == "}") { d--; if (started && d == 0) return out }
      if (started && !skip) out = out ch
    }
    if (started) out = out "\n"
  }
  return out
}
# classify a whitespace-free array expression: "empty", "own", "other" or "unresolved"
function classify(e,    inner, k, parts, cnt) {
  while (e ~ /^\(.*\)$/) e = substr(e, 2, length(e) - 2)
  if (e == "null" || e == "{}" || e == "newType[0]" || e == "newType[]{}") return "empty"
  if (e ~ /^newType\[\]\{.*\}$/) inner = substr(e, 11, length(e) - 11)
  else if (e ~ /^\{.*\}$/) inner = substr(e, 2, length(e) - 2)
  else return "unresolved"
  cnt = split(inner, parts, ",")
  for (k = 1; k <= cnt; k++)
    if (parts[k] ~ /^(this\.)?(TYPE|getType\(\))$/ || parts[k] == own ".TYPE") return "own"
  return "other"
}
# a returned field (`a`, `Own.a`, `this.a`, `(Type[]) a.clone()`) -> the verdict of every `a = …;` in the file:
# any empty -> empty, any other -> other, all own -> own, no assignment found -> unresolved
function resolve(e, n,    f, init, v, out, seen) {
  sub(/^\(Type\[\]\)/, "", e); sub(/\.clone\(\)$/, "", e)
  if (own != "" && index(e, own ".") == 1) e = substr(e, length(own) + 2)
  sub(/^this\./, "", e)
  if (e !~ /^[A-Za-z_][A-Za-z0-9_]*$/) return "unresolved"
  out = "own"; seen = 0
  for (f = 1; f <= n; f++) {
    if (!match(code[f], ("(^|[^A-Za-z0-9_])" e "[[:space:]]*=([^=]|$)"))) continue
    init = until_char(f, RSTART, n, ";"); gsub(/[[:space:]]/, "", init)
    sub("^[^=]*=", "", init)
    v = classify(init); seen = 1
    if (v == "empty") return "empty"
    if (v != "own") out = (v == "other" || out == "other") ? "other" : "unresolved"
  }
  return seen ? out : "unresolved"
}
{ raw[NR] = $0 }
END {
  rank[""] = 0; rank["own"] = 1; rank["unresolved"] = 2; rank["other"] = 3; rank["empty"] = 4
  n = NR
  mb_strip(raw, n, code)
  own = ""
  for (i = 1; i <= n && own == ""; i++)
    if (match(code[i], /(^|[^A-Za-z0-9_])class[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
      own = substr(code[i], RSTART, RLENGTH); sub(/.*class[[:space:]]+/, "", own)
    }
  for (i = 1; i <= n; i++) {
    if (code[i] !~ /Type[[:space:]]*\[[[:space:]]*\][[:space:]]+getServiceTypes[[:space:]]*\(/) continue
    head = until_char(i, 1, n, "{")
    if (index(head, ";") > 0) continue                     # abstract / interface declaration
    b = block_body(i, n); gsub(/[[:space:]]+/, " ", b)
    # every `return <expr>;` of the method counts (nested scopes are blanked by block_body); the worst wins:
    # empty > other > unresolved > own > "" (rank 0 = no return seen yet; no return at all = unresolved)
    verdict = ""; rest = b
    while (match(rest, /(^|[^A-Za-z0-9_])return[[:space:]][^;]*;/)) {
      e = substr(rest, RSTART, RLENGTH); rest = substr(rest, RSTART + RLENGTH)
      sub(/.*return[[:space:]]/, "", e); sub(/;$/, "", e); gsub(/[[:space:]]/, "", e)
      v = classify(e)
      if (v == "unresolved") v = resolve(e, n)
      if (rank[v] > rank[verdict]) verdict = v
    }
    if (verdict == "") verdict = "unresolved"
    if (verdict == "empty")
      printf "FAIL  lint-service-types  %s:%d  getServiceTypes() registers no type (empty array or null) -- the service is never registered; return new Type[] { TYPE } (METHODOLOGY.md § rt)\n", FILE, i
    else if (verdict == "other")
      printf "WARN  lint-service-types  %s:%d  getServiceTypes() does not return the service's own TYPE -- Sys.getService(%s.TYPE) will not find it; add TYPE unless that is intended\n", FILE, i, own
    else if (verdict == "unresolved")
      printf "WARN  lint-service-types  %s:%d  getServiceTypes() return not resolved in this file -- check it returns a non-empty array holding TYPE\n", FILE, i
  }
}
AWKEOF

awk_lint_list -name '*.java'
fail=0
warn=0
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -f "$AL_TMP/method-boundary.awk" -f "$AL_TMP/main.awk" "$f" 2>"$AL_TMP/awk.err"); then
    printf '%s: cannot scan %s: %s\n' "$AL_NAME" "$f" "$(head -n 1 "$AL_TMP/awk.err")" >&2
    # shellcheck disable=SC2034  # read by awk_lint_exit (lib/awk-lint.sh)
    AL_ERR=1
    continue
  fi
  [ -n "$out" ] || continue
  printf '%s\n' "$out"
  case "$out" in *FAIL\ \ lint-service-types*) fail=1 ;; esac
  case "$out" in *WARN\ \ lint-service-types*) warn=1 ;; esac
done < "$AL_TMP/files"
if [ "$fail" -eq 1 ] || { [ "$warn" -eq 1 ] && [ "$AL_STRICT" -eq 1 ]; }; then awk_lint_exit 1; fi
awk_lint_exit 0
