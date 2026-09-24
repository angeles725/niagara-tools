#!/usr/bin/env bash
# lint-changed-hot-write.sh — flags a hot changed() callback that re-runs a heavy control
# method at INPUT RATE and writes a persisted (non-transient) slot on every call.
#
# Shape (RUN8, B1158): a `changed(Property p, Context cx)` override with N>=6 `p == <prop>`
# branches guarding a call to a heavy `execute()`/control method, with NO rate guard (no
# `Clock.millis()` delta gate, no `Clock.schedule` coalescing ticket) IN THE CHANGED() BODY,
# where the class also writes >=1 NON-TRANSIENT @NiagaraProperty slot (declared without
# Flags.TRANSIENT) reachable from changed() (directly, or from its execute() callee).
# Under a churning field bus (a flapping IO-34 relay drove ~18 msg/s), this ran execute()
# 25-90x over its intended tick rate on the single engine thread, marking the station config
# dirty on every call — a tenured-heap growth path distinct from the timer/subscriber/null-
# Context leak shapes the kit already lints. Fix: leading-edge time debounce on the callback
# path (Clock.millis() gate), or move periodic writes to a tick and keep changed() input-only.
# See types/issues-and-gotchas.md §I1. [ev: retro live-diagnosis-hardening-deltas Δ1]
# [ev: corpus B1158; BCompressorControl.java:1850-1871,2019-2021]
#
# Usage:  lint-changed-hot-write.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned) for a `changed(Property <name>, ...)`
#   override. Only the raw slot-o-matic declaration form
#   `Property <name> = newProperty(<flags>, ...)` is recognized for the non-transient check
#   (the @NiagaraProperty annotation form is not parsed by this lint — false-negative, not
#   false-positive, documented limitation). Reachability is one hop: changed()'s own body,
#   or a zero-arg `execute()` method it calls.
#   Row:  WARN  lint-changed-hot-write  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env
# VCS-free by design (kit-links L2).
# Mutation: CHW2 -- drop the branch-count threshold (N>=6) so a 1-branch changed() false-WARNs
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-changed-hot-write: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-changed-hot-write.sh [--strict] <src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-changed-hot-write.sh [--strict] <src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-changed-hot-write: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  # Strip // line comments (block comments not handled -- advisory heuristic, documented).
  for (i = 1; i <= NR; i++) { ln = lines[i]; sub(/\/\/.*$/, "", ln); lines[i] = ln }

  # 1. Locate a `changed(Property <name>` override signature.
  changed_line = 0; pname = ""
  for (i = 1; i <= NR; i++) {
    if (match(lines[i], /changed[[:space:]]*\([[:space:]]*Property[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
      seg = substr(lines[i], RSTART, RLENGTH)
      sub(/^.*Property[[:space:]]+/, "", seg)
      pname = seg
      changed_line = i
      break
    }
  }
  if (changed_line == 0) exit 0

  # 2. Brace-depth extract the changed() body.
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
  changed_body = ""
  for (i = changed_line; i <= changed_end; i++) changed_body = changed_body "\n" lines[i]

  # 3. Count `<pname> ==` branches (word-boundary via a leading non-identifier char).
  cnt = 0; tmp = changed_body
  while (match(tmp, "[^A-Za-z0-9_]" pname "[[:space:]]*==")) {
    cnt++
    tmp = substr(tmp, RSTART + RLENGTH)
  }
  if (cnt < 6) exit 0

  # 4. Must call a heavy execute() control method.
  if (index(changed_body, "execute()") == 0) exit 0

  # 5. No rate guard in the changed() body.
  if (index(changed_body, "Clock.millis(") > 0) exit 0
  if (index(changed_body, "Clock.schedule") > 0) exit 0

  # 6. Locate an execute() method DECLARATION (zero-arg). Distinguished from a CALL by what
  #    immediately follows the "()" on the line: a call is "execute();" (trailing ';'); a
  #    declaration is either bare (Allman '{' on the next line) or a one-liner "execute() { ... }".
  exec_line = 0
  for (i = 1; i <= NR; i++) {
    probe = " " lines[i]
    if (match(probe, /[^A-Za-z0-9_]execute[[:space:]]*\([[:space:]]*\)/)) {
      trailing = substr(probe, RSTART + RLENGTH)
      gsub(/^[[:space:]]*/, "", trailing)
      if (substr(trailing, 1, 1) != ";") { exec_line = i; break }
    }
  }
  exec_body = changed_body
  if (exec_line > 0) {
    depth = 0; started = 0; exec_end = 0
    for (i = exec_line; i <= NR; i++) {
      ln = lines[i]
      for (ci = 1; ci <= length(ln); ci++) {
        c = substr(ln, ci, 1)
        if (c == "{") { depth++; started = 1 }
        else if (c == "}") depth--
      }
      if (started && depth == 0) { exec_end = i; break }
    }
    if (exec_end > 0) {
      exec_body = ""
      for (i = exec_line; i <= exec_end; i++) exec_body = exec_body "\n" lines[i]
    }
  }

  # 7. Collect NON-transient property names: `Property <name> = newProperty(<flags>, ...)`
  #    where <flags> does not mention TRANSIENT (raw slot-o-matic form only, D-limitation).
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
      if (index(rest, "TRANSIENT") == 0 && nm != "") {
        n_nt++; nt_name[n_nt] = nm
      }
    }
  }
  if (n_nt == 0) exit 0

  # 8. A non-transient property's setter reachable from changed() or its execute() callee.
  wprop = ""
  for (k = 1; k <= n_nt; k++) {
    cap = toupper(substr(nt_name[k], 1, 1)) substr(nt_name[k], 2)
    setter = "set" cap "("
    if (index(exec_body, setter) > 0 || index(changed_body, setter) > 0) { wprop = nt_name[k]; break }
  }
  if (wprop == "") exit 0

  printf "WARN  lint-changed-hot-write  %s:%d  changed(%s) has %d '%s ==' branches guarding execute()," \
    " no Clock.millis()/schedule rate guard, writes non-transient slot '%s' -- debounce the callback" \
    " (leading-edge Clock.millis() gate) or move the periodic write to a tick\n", \
    FILE, changed_line, pname, cnt, pname, wprop
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
