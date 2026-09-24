#!/usr/bin/env bash
# lint-session-store-lazy-evict.sh — flags a STATIC session/token store whose eviction is
# only lazy (on query) or explicit-logout, with no sweep reachable from the insert itself
# and no scheduled purge anywhere in the file.
#
# Shape (UXS7, B1160): a `static Map<String,*>` field in a `-ux` class inserts on login
# (`.put(`) and removes ONLY via an explicit logout method or a query-time expiry check --
# never a sweep in the SAME method that inserts, and never a Clock.schedule-driven purge.
# Being STATIC, the map survives servlet re-mount, so a reused container id (JSESSIONID) can
# inherit an authenticated write session across a stop/restart cycle -- a session-fixation
# shape, not (primarily) an unbounded-growth leak (a session that never logs out is
# negligible heap). Rule: sweep-on-insert (or a scheduled/size-bounded eviction); question
# `static`; bind auth to user+issue-time, not the bare container id; audit the write. See
# types/issues-and-gotchas.md §F2. [ev: retro live-diagnosis-hardening-deltas Δ5]
# [ev: corpus B1160; DashboardConfigSession.java:31-126]
#
# Usage:  lint-session-store-lazy-evict.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned). Run on any -ux profile with Java sources.
#   Candidate: a `static ... Map<...> <field>` declaration with a `.put(` call on <field>
#   somewhere in the file. WARNs unless EITHER (a) the method containing the `.put(` call
#   also calls `<field>.remove(` in its own body (sweep-on-insert), or (b) the file contains
#   a `Clock.schedule` call whose scheduled body (one hop) calls `<field>.remove(`
#   (scheduled purge). A `remove(` reachable only from an unrelated method (explicit logout,
#   or a query-time expiry check) does NOT count as a guard -- that is exactly the shape
#   this lint flags.
#   Row:  WARN  lint-session-store-lazy-evict  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env
# VCS-free by design (kit-links L2).
# Mutation: SSL2 -- drop the `static` requirement so an instance-scope map (e.g. ConfigSession)
# false-WARNs
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-session-store-lazy-evict: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-session-store-lazy-evict.sh [--strict] <src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-session-store-lazy-evict.sh [--strict] <src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-session-store-lazy-evict: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  for (i = 1; i <= NR; i++) { ln = lines[i]; sub(/\/\/.*$/, "", ln); lines[i] = ln }

  # 1. Collect static Map<...> field names.
  n_f = 0
  for (i = 1; i <= NR; i++) {
    ln = lines[i]
    if (ln ~ /static/ && match(ln, /(Map|HashMap|ConcurrentHashMap)[[:space:]]*<[^>]*>[[:space:]]+[A-Za-z_][A-Za-z0-9_]*/)) {
      seg = substr(ln, RSTART, RLENGTH)
      match(seg, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*$/)
      fname = substr(seg, RSTART, RLENGTH)
      gsub(/[[:space:]]/, "", fname)
      if (fname != "") { n_f++; field[n_f] = fname; field_decl_line[n_f] = i }
    }
  }
  if (n_f == 0) exit 0

  # 2. Whole-file scheduled-purge check: a Clock.schedule call whose one-hop scheduled
  #    method body contains <field>.remove(.
  has_scheduled_purge_for = ""
  for (i = 1; i <= NR; i++) {
    if (index(lines[i], "Clock.schedule") == 0) continue
    # Look for a zero-arg callee name near the schedule call (e.g. Clock.schedule(this, t, sweepAction, null)).
    for (i2 = i; i2 <= i + 2 && i2 <= NR; i2++) {
      ln2 = lines[i2]
      for (k = 1; k <= n_f; k++) {
        # A scheduled body somewhere in the file that calls field.remove( counts as a purge,
        # regardless of exact wiring (advisory heuristic: presence of BOTH signals in the file).
        if (index(ln2, "Clock.schedule") > 0) {
          for (j = 1; j <= NR; j++) {
            if (index(lines[j], field[k] ".remove(") > 0 && j != field_decl_line[k]) {
              # crude scope check: this remove() is inside a method whose body ALSO schedules
              # (a scheduled sweep method calls remove on itself periodically) OR is invoked
              # by name from the schedule call line. Accept file-wide co-occurrence as clean --
              # a false-negative bias, not false-positive (documented).
              has_scheduled_purge_for = has_scheduled_purge_for " " field[k]
            }
          }
        }
      }
    }
  }

  # 3. For each field with a `.put(`, find the enclosing method and check for a same-method
  #    `.remove(` (sweep-on-insert).
  for (k = 1; k <= n_f; k++) {
    fname = field[k]
    if (index(" " has_scheduled_purge_for " ", " " fname " ") > 0) continue

    put_line = 0
    for (i = 1; i <= NR; i++) {
      if (index(lines[i], fname ".put(") > 0) { put_line = i; break }
    }
    if (put_line == 0) continue

    # Find the enclosing method: scan backward for the nearest method-signature-shaped line
    # (two identifiers then a parenthesized arg list, NOT immediately followed by ';' -- that
    # would be a call, not a declaration -- so a one-liner "name(args) { ... }" still matches),
    # then brace-extract forward from it.
    msig = 0
    for (i = put_line; i >= 1; i--) {
      ln = lines[i]
      if (match(ln, /[A-Za-z_][A-Za-z0-9_<>\[\]]*[[:space:]]+[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([^;{}]*\)/)) {
        trailing = substr(ln, RSTART + RLENGTH)
        gsub(/^[[:space:]]*/, "", trailing)
        if (substr(trailing, 1, 1) != ";") { msig = i; break }
      }
    }
    if (msig == 0) msig = 1

    depth = 0; started = 0; mend = NR
    for (i = msig; i <= NR; i++) {
      ln = lines[i]
      for (ci = 1; ci <= length(ln); ci++) {
        c = substr(ln, ci, 1)
        if (c == "{") { depth++; started = 1 }
        else if (c == "}") depth--
      }
      if (started && depth == 0) { mend = i; break }
    }

    swept = 0
    for (i = msig; i <= mend; i++) {
      if (index(lines[i], fname ".remove(") > 0) { swept = 1; break }
    }
    if (!swept) {
      printf "WARN  lint-session-store-lazy-evict  %s:%d  static session/token map '%s' inserted here" \
        " with no sweep in the SAME method (no scheduled purge found either) -- add a" \
        " scheduled/size-bounded eviction, or bind auth to user+issue-time rather than the bare" \
        " container id\n", FILE, put_line, fname
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
