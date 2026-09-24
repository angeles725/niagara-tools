#!/usr/bin/env bash
# lint-persist-hot-write.sh — flags a persisted (non-transient) property SETTER called from
# changed() or its one-hop callee with no cadence guard nearby the call site.
#
# Shape (PER8, B1159): a NON-transient @NiagaraProperty slot (declared without Flags.TRANSIENT
# in the raw slot-o-matic form `Property <name> = newProperty(<flags>, ...)`) is correctly
# non-transient (it MUST survive restart, e.g. for fair lead/lag rotation) but its setter is
# called EVERY CYCLE from changed()/execute(), so each call marks the station config dirty and
# fires every link off that slot. Pairs with lint-changed-hot-write.sh (RUN8): that lint flags
# the callback-rate SHAPE (fan-in + no rate guard); this lint flags the SETTER CALL SITE itself,
# independent of branch count, so a single always-executed persisted write still WARNs.
# Rule: integrate in memory; checkpoint the non-transient slot on a periodic tick + in
# stopped(); seed the value once on start (seedHours()-style guard). See
# types/issues-and-gotchas.md §I2. [ev: retro live-diagnosis-hardening-deltas Δ3]
# [ev: corpus B1159; BCompressorControl.java:326,1403,2019]
#
# Usage:  lint-persist-hot-write.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned). Only the raw slot-o-matic declaration
#   `Property <name> = newProperty(<flags>, ...)` is recognized (the @NiagaraProperty
#   annotation form is not parsed -- false-negative, not false-positive, documented
#   limitation). A setter CALL (not its method DECLARATION) inside changed()'s body or a
#   zero-arg method changed() calls (one hop) is a candidate. A cadence guard is a
#   `Clock.millis(` comparison or a `%` (modulo/counter) test within 5 lines above the call,
#   in the SAME method.
#   Row:  WARN  lint-persist-hot-write  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env
# VCS-free by design (kit-links L2).
# Mutation: PHW2 -- drop the cadence-guard lookback so a guarded setter call still false-WARNs
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-persist-hot-write: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-persist-hot-write.sh [--strict] <src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-persist-hot-write.sh [--strict] <src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-persist-hot-write: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  for (i = 1; i <= NR; i++) { ln = lines[i]; sub(/\/\/.*$/, "", ln); lines[i] = ln }

  # 1. Locate changed(Property ...) -- no callback, nothing to check.
  changed_line = 0
  for (i = 1; i <= NR; i++) {
    if (match(lines[i], /changed[[:space:]]*\([[:space:]]*Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
      changed_line = i; break
    }
  }
  if (changed_line == 0) exit 0

  depth = 0; started = 0; changed_end = 0
  for (i = changed_line; i <= NR; i++) {
    ln = lines[i]
    for (ci = 1; ci <= length(ln); ci++) {
      c = substr(ln, ci, 1)
      if (c == "{") { depth++; started = 1 }
      else if (c == "}") depth--
    }
    if (started && depth == 0) { changed_end = i; break }
  }
  if (changed_end == 0) exit 0

  # 2. One-hop callee: a zero-arg method invoked from changed(), found generically
  #    (candidate identifier immediately followed by "()" and not a keyword).
  n_kw = split("if for while switch catch try finally return new else do super this Clock Sys", KW, " ")
  for (k = 1; k <= n_kw; k++) kw[KW[k]] = 1
  callee_line = 0; callee_name = ""
  for (i = changed_line; i <= changed_end && callee_line == 0; i++) {
    ln = lines[i]
    if (match(ln, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([[:space:]]*\)[[:space:]]*;/)) {
      cand = substr(ln, RSTART, RLENGTH)
      sub(/[[:space:]]*\([[:space:]]*\)[[:space:]]*;.*/, "", cand)
      if (!(cand in kw)) callee_name = cand
    }
  }
  # A declaration is distinguished from a call by what follows "()" on the line: a call ends
  # in ';' right after the parens (possibly with trailing whitespace); a declaration is either
  # bare (Allman '{' on the next line) or a one-liner "name() { ... }".
  if (callee_name != "") {
    for (i = 1; i <= NR; i++) {
      probe = " " lines[i]
      if (match(probe, "[^A-Za-z0-9_]" callee_name "[[:space:]]*\\([[:space:]]*\\)")) {
        trailing = substr(probe, RSTART + RLENGTH)
        gsub(/^[[:space:]]*/, "", trailing)
        if (substr(trailing, 1, 1) != ";") { callee_line = i; break }
      }
    }
  }
  callee_end = 0
  if (callee_line > 0) {
    depth = 0; started = 0
    for (i = callee_line; i <= NR; i++) {
      ln = lines[i]
      for (ci = 1; ci <= length(ln); ci++) {
        c = substr(ln, ci, 1)
        if (c == "{") { depth++; started = 1 }
        else if (c == "}") depth--
      }
      if (started && depth == 0) { callee_end = i; break }
    }
  }

  # 3. Collect NON-transient property names (raw slot-o-matic form).
  n_nt = 0
  for (i = 1; i <= NR; i++) {
    ln = lines[i]
    if (match(ln, /Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*newProperty\(/)) {
      nm = substr(ln, RSTART, RLENGTH)
      sub(/[[:space:]]*=.*/, "", nm)
      sub(/^.*Property[[:space:]]+/, "", nm)
      rest = ln
      sub(/^.*newProperty\(/, "", rest)
      sub(/,.*/, "", rest)
      if (index(rest, "TRANSIENT") == 0 && nm != "") { n_nt++; nt_name[n_nt] = nm }
    }
  }
  if (n_nt == 0) exit 0

  # 4. Scan changed()'s body and the one-hop callee's body for setter CALLS (not declarations)
  #    to any non-transient property, with no cadence guard within 5 lines above (same scope).
  n_scope = 0
  scope_start[1] = changed_line; scope_end[1] = changed_end; n_scope = 1
  if (callee_line > 0 && callee_end > 0) {
    n_scope = 2; scope_start[2] = callee_line; scope_end[2] = callee_end
  }

  for (k = 1; k <= n_nt; k++) {
    cap = toupper(substr(nt_name[k], 1, 1)) substr(nt_name[k], 2)
    setter = "set" cap "("
    for (s = 1; s <= n_scope; s++) {
      ms = scope_start[s]; me = scope_end[s]
      for (i = ms; i <= me; i++) {
        ln = lines[i]
        pos = index(ln, setter)
        if (pos == 0) continue
        # Skip a DECLARATION line (has a Java type keyword before the setter name on the line).
        if (ln ~ /(void|public|private|protected|static)[[:space:]]+[A-Za-z_<>\[\],. ]*set[A-Z]/) continue
        # Cadence guard: look back up to 5 lines, same scope, for Clock.millis( or a % test.
        guarded = 0
        for (j = i; j >= ms && j >= i - 5; j--) {
          if (index(lines[j], "Clock.millis(") > 0) { guarded = 1; break }
          if (match(lines[j], /%[[:space:]]*[A-Za-z0-9_]+[[:space:]]*(==|!=)/)) { guarded = 1; break }
        }
        if (!guarded) {
          printf "WARN  lint-persist-hot-write  %s:%d  %s...) writes non-transient slot '%s'" \
            " reached from changed(), no cadence guard (Clock.millis()/counter) within 5 lines --" \
            " integrate in memory, checkpoint on a periodic tick + in stopped(), seed once on start\n", \
            FILE, i, setter, nt_name[k]
        }
      }
    }
  }
}
AWKEOF

had_warn=0
while IFS= read -r f; do
  out=$(awk -v FILE="$f" -f "$_TMP/main.awk" "$f" 2>/dev/null)
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    had_warn=1
  fi
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print | LC_ALL=C sort)

if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
