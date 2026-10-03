#!/usr/bin/env bash
# lint-persist-hot-write.sh — flags a persisted (non-transient) property SETTER called from
# changed() or any of its one-hop callees with no cadence guard nearby the call site.
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
#   limitation). A setter CALL (not its method DECLARATION) inside changed()'s body or ANY
#   zero-arg method changed() calls (one hop, every callee) is a candidate. A cadence guard is a
#   `Clock.millis(` comparison or a `%` (modulo/counter) test within 5 lines above the call,
#   in the SAME method.
#   Row:  WARN  lint-persist-hot-write  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: PHW2 -- drop the cadence-guard lookback so a guarded setter call still false-WARNs
# Mutation: PHW6 -- keeping only the last callee drops a hot write in an earlier callee
# Mutation: PHW-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: PHW-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
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

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"

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

  # 2. One-hop callees: EVERY zero-arg method invoked from changed() (identifier immediately
  #    followed by "()" and ";", not a keyword) -- each one's body is scanned, not only one.
  n_kw = split("if for while switch catch try finally return new else do super this Clock Sys", KW, " ")
  for (k = 1; k <= n_kw; k++) kw[KW[k]] = 1
  n_callee = 0
  for (i = changed_line; i <= changed_end; i++) {
    rest = lines[i]
    while (match(rest, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([[:space:]]*\)[[:space:]]*;/)) {
      cand = substr(rest, RSTART, RLENGTH)
      rest = substr(rest, RSTART + RLENGTH)
      sub(/[[:space:]]*\([[:space:]]*\)[[:space:]]*;.*/, "", cand)
      if (!(cand in kw) && !(cand in seen_callee)) { seen_callee[cand] = 1; n_callee++; callee_name[n_callee] = cand }
    }
  }
  # A declaration is distinguished from a call by what follows "()" on the line: a call ends
  # in ';' right after the parens (possibly with trailing whitespace); a declaration is either
  # bare (Allman '{' on the next line) or a one-liner "name() { ... }".
  n_scope = 1
  for (c = 1; c <= n_callee; c++) {
    callee_line = 0
    for (i = 1; i <= NR; i++) {
      probe = " " lines[i]
      if (match(probe, "[^A-Za-z0-9_]" callee_name[c] "[[:space:]]*\\([[:space:]]*\\)")) {
        trailing = substr(probe, RSTART + RLENGTH)
        gsub(/^[[:space:]]*/, "", trailing)
        if (substr(trailing, 1, 1) != ";") { callee_line = i; break }
      }
    }
    if (callee_line == 0) continue
    depth = 0; started = 0; callee_end = 0
    for (i = callee_line; i <= NR; i++) {
      ln = lines[i]
      for (ci = 1; ci <= length(ln); ci++) {
        ch = substr(ln, ci, 1)
        if (ch == "{") { depth++; started = 1 }
        else if (ch == "}") depth--
      }
      if (started && depth == 0) { callee_end = i; break }
    }
    if (callee_end > 0) { n_scope++; scope_start[n_scope] = callee_line; scope_end[n_scope] = callee_end }
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

  # 4. Scan changed()'s body and every one-hop callee's body for setter CALLS (not
  #    declarations) to any non-transient property, with no cadence guard within 5 lines above
  #    (same scope).
  scope_start[1] = changed_line; scope_end[1] = changed_end

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
had_err=0
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.java'; then
  printf 'lint-persist-hot-write: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
  had_err=1
fi
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -f "$_TMP/main.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-persist-hot-write: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
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
