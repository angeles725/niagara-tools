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
#   a `Clock.schedule*` call whose scheduled body (one hop: the `do<Action>()` or `<action>()`
#   method named by an argument of the call) calls `<field>.remove(` (scheduled purge). A `remove(` reachable only from an unrelated method (explicit logout,
#   or a query-time expiry check) does NOT count as a guard -- that is exactly the shape
#   this lint flags.
#   Row:  WARN  lint-session-store-lazy-evict  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env or an unscannable source file
# VCS-free by design (kit-links L2).
# Mutation: SSL2 -- drop the `static` requirement so an instance-scope map (e.g. ConfigSession)
# false-WARNs
# Mutation: SSL5 -- file-wide co-occurrence (any Clock.schedule + any remove) hides the logout-only shape
# Mutation: SSL-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
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

  # 2. Scheduled-purge check, one hop: every identifier on a Clock.schedule* call (up to the
  #    closing ';', at most 3 lines) is a candidate action; its scheduled body is the method
  #    declared as do<Action>() or <action>(). A field counts as purged only when such a body
  #    calls <field>.remove( -- a remove() elsewhere (explicit logout, query-time expiry) next to
  #    an unrelated schedule (a refresh tick) is NOT a purge.
  has_scheduled_purge_for = ""
  for (i = 1; i <= NR; i++) {
    if (index(lines[i], "Clock.schedule") == 0) continue
    call = ""
    for (i2 = i; i2 <= i + 2 && i2 <= NR; i2++) {
      call = call " " lines[i2]
      if (index(lines[i2], ";") > 0) break
    }
    sub(/^.*Clock\.schedule[A-Za-z]*[[:space:]]*\(/, "", call)
    sub(/;.*$/, "", call)
    rest = call
    while (match(rest, /[A-Za-z_][A-Za-z0-9_]*/)) {
      act = substr(rest, RSTART, RLENGTH)
      rest = substr(rest, RSTART + RLENGTH)
      doname = "do" toupper(substr(act, 1, 1)) substr(act, 2)
      for (j = 1; j <= NR; j++) {
        probe = " " lines[j]
        if (!match(probe, "[^A-Za-z0-9_](" doname "|" act ")[[:space:]]*\\([^;]*\\)[[:space:]]*(\\{|$|throws)")) continue
        depth = 0; started = 0; send = 0
        for (j2 = j; j2 <= NR; j2++) {
          ln = lines[j2]
          for (ci = 1; ci <= length(ln); ci++) {
            ch = substr(ln, ci, 1)
            if (ch == "{") { depth++; started = 1 }
            else if (ch == "}") depth--
          }
          if (started && depth == 0) { send = j2; break }
        }
        for (j2 = j; j2 <= send; j2++)
          for (k = 1; k <= n_f; k++)
            if (index(lines[j2], field[k] ".remove(") > 0)
              has_scheduled_purge_for = has_scheduled_purge_for " " field[k]
        break
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
    # No enclosing signature found (a static initializer): the scope is the put line alone,
    # never the whole file -- a remove() anywhere else must not count (fail closed).
    noscope = (msig == 0)
    if (noscope) msig = put_line

    depth = 0; started = 0; mend = (noscope ? put_line : NR)
    for (i = msig; i <= NR && !noscope; i++) {
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
had_err=0
while IFS= read -r f; do
  # An awk failure (unreadable file, awk error) is an env error, never a clean pass (fail closed).
  if ! out=$(awk -v FILE="$f" -f "$_TMP/main.awk" "$f" 2>"$_TMP/awk.err"); then
    printf 'lint-session-store-lazy-evict: cannot scan %s: %s\n' "$f" "$(head -n 1 "$_TMP/awk.err")" >&2
    had_err=1
    continue
  fi
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    had_warn=1
  fi
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print | LC_ALL=C sort)

[ "$had_err" -eq 0 ] || exit 3
if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
