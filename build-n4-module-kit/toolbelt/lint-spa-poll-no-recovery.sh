#!/usr/bin/env bash
# lint-spa-poll-no-recovery.sh — flags an rc/ SPA poll loop (setInterval/setTimeout) whose
# catch path only marks data stale and never reloads/re-authenticates after the underlying
# session dies (e.g. a station restart kills the web session but the kiosk keeps polling the
# dead one forever).
#
# Shape (Δ1, PANCCADIA 2026-09-23): `setInterval(poll, N4.pollMs)` registers a poll function
# whose `catch` branch only sets a stale flag and repaints an "error" state
# (`DashboardPan-ux/src/rc/index.html` 2.4.2 `poll()` — `data.forEach(d => d.st = "stale");
# paint("error");`). The 10" Honeywell kiosk has no page-level auto-refresh, so after a station
# restart kills the web session the SPA shows dead values until a human power-cycles the panel.
# The fix (shipped 2.4.3, same evidence): a failure-count watchdog in the SAME catch path that,
# after N consecutive poll failures, probes the same origin and calls `location.reload()` once
# the server answers again (a fresh session, not a stale one). See types/dashboard.md "ux —
# servlet + SPA" checklist and corpus B1064 (REST-poll 5s pattern this lint pairs with).
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ1]
# [ev: corpus DashboardPan-ux index.html poll() 2.4.2 -> 2.4.3]
#
# Usage:  lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>
#   Scans *.html and *.js under <ux-src-root> (dot-dirs pruned; typically an rc/ tree, but the
#   root itself is not required to be named rc/). Comments are line-stripped (`//...`, naive —
#   see limitations). Per file:
#     1. the FIRST `setInterval(<name>, ...)` or `setTimeout(<name>, ...)` call names a poll
#        function <name>;
#     2. its declaration (`function <name>(` or `async function <name>(`) is located and its
#        body is brace-extracted forward from that line;
#     3. that body must contain a `catch (` / `catch(` clause — no catch, no shape to flag;
#     4. the WHOLE FILE is then checked for a recovery signal: `location.reload(` or
#        `location.href =` ANYWHERE. Absent -> WARN at the setInterval/setTimeout call line.
#   Row:  WARN  lint-spa-poll-no-recovery  <file>:<line>  setInterval/setTimeout poll loop
#         '<name>()' has a catch path with no location.reload(/location.href recovery anywhere
#         in the file -- kiosk/HMI will stay blank after a station restart; add a
#         failure-count watchdog -> same-origin probe -> location.reload()
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env
#
# Known limitations (advisory heuristic, documented per kit style):
#   - `location.reload(`/`location.href =` ANYWHERE in the file suppresses the WARN, even if it
#     is unrelated to the poll loop's catch path (e.g. a manual "reload" button elsewhere) --
#     a false-NEGATIVE bias, not a false-positive risk (mirrors lint-session-store-lazy-evict's
#     file-wide co-occurrence rule).
#   - only the FIRST setInterval/setTimeout call in a file is checked; a file registering a
#     second, unrelated interval after the poll loop is not separately evaluated.
#   - a poll function recovering via something OTHER than a full navigation reload (e.g. tearing
#     down and re-creating the app's own session state in place) is a false positive; this lint
#     only recognises the reload-based recipe named in the retro.
#   - naive `//` comment stripping can mis-treat a `//` inside a string/URL as a comment start;
#     cheap to accept per kit style (rc-scan.sh / lint-jasmine-ux.sh precedent).
# VCS-free by design (kit-links L2). LC_ALL=C.
# Mutation: SPR2 -- drop the location.reload(/location.href recovery check so a compliant watchdog file still WARNs
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-spa-poll-no-recovery: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-spa-poll-no-recovery.sh [--strict] <ux-src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-spa-poll-no-recovery: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  for (i = 1; i <= NR; i++) { ln = lines[i]; sub(/\/\/.*$/, "", ln); lines[i] = ln }

  # 1. First setInterval(<name>, ...) or setTimeout(<name>, ...) call.
  fn = ""; call_line = 0
  for (i = 1; i <= NR; i++) {
    if (match(lines[i], /(setInterval|setTimeout)\([[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*,/)) {
      seg = substr(lines[i], RSTART, RLENGTH)
      match(seg, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*,$/)
      fn = substr(seg, RSTART, RLENGTH)
      gsub(/[[:space:]]*,$/, "", fn)
      call_line = i
      break
    }
  }
  if (fn == "") exit 0

  # 2. Locate the function declaration and brace-extract its body forward.
  decl_line = 0
  for (i = 1; i <= NR; i++) {
    if (match(lines[i], "function[[:space:]]+" fn "[[:space:]]*\\(")) { decl_line = i; break }
  }
  if (decl_line == 0) exit 0

  depth = 0; started = 0; body_end = NR
  for (i = decl_line; i <= NR; i++) {
    ln = lines[i]
    for (ci = 1; ci <= length(ln); ci++) {
      c = substr(ln, ci, 1)
      if (c == "{") { depth++; started = 1 }
      else if (c == "}") depth--
    }
    if (started && depth == 0) { body_end = i; break }
  }

  # 3. The body must contain a catch clause -- otherwise this is not the flagged shape.
  has_catch = 0
  for (i = decl_line; i <= body_end; i++) {
    if (match(lines[i], /catch[[:space:]]*\(/)) { has_catch = 1; break }
  }
  if (!has_catch) exit 0

  # 4. Whole-file recovery signal.
  has_recovery = 0
  for (i = 1; i <= NR; i++) {
    if (index(lines[i], "location.reload(") > 0 || match(lines[i], /location\.href[[:space:]]*=/)) {
      has_recovery = 1; break
    }
  }
  if (has_recovery) exit 0

  printf "WARN  lint-spa-poll-no-recovery  %s:%d  setInterval/setTimeout poll loop '%s()' has a" \
    " catch path with no location.reload(/location.href recovery anywhere in the file --" \
    " kiosk/HMI will stay blank after a station restart; add a failure-count watchdog ->" \
    " same-origin probe -> location.reload()\n", FILE, call_line, fn
}
AWKEOF

had_warn=0
while IFS= read -r f; do
  out=$(awk -v FILE="$f" -f "$_TMP/main.awk" "$f" 2>/dev/null)
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    had_warn=1
  fi
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f \( -name '*.html' -o -name '*.js' \) -print | LC_ALL=C sort)

if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
