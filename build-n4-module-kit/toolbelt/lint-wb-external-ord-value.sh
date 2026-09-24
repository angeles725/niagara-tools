#!/usr/bin/env bash
# lint-wb-external-ord-value.sh — flags a -wb manager/view method that resolves a per-row
# ORD to a target OUTSIDE the view's subscribed subtree, reads that target's live
# value/status, but never leases a refresh for that specific external target.
#
# Shape (Δ3, corpus B1140-G1): subject-subtree `registerForComponentEvents(subject, depth)`
# only ever reaches DESCENDANTS of `subject`. A manager column that resolves a per-row
# `BOrd` property to an ARBITRARY external station component (e.g.
# `BApillmImportMap.targetOrd` -> any point outside the importer subtree, or
# `BApillmExport.ord` -> any export target) and then reads that resolved target's live
# value/status is reading a component the subject-subtree subscription never covers.
# It stays at its resolve-time snapshot (or shows "unresolved" on the first, async,
# resolve) no matter how the subject-subtree depth is tuned. Fix: for each resolved
# external target, call `target.loadSlots()` once (forces the synchronous first read)
# and `registerForComponentEvents(target, 0)` (leases future refreshes for that target
# specifically) -- guard repeat resolves with `isRegisteredForComponentEvents(target)`.
# See types/wb-widgets.md "view refresh rules" (Δ3 bullet) for the full rule and the
# depth rule it pairs with (Δ2).
# [ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ3]
# [ev: corpus B1140-G1; Apillm BApillmImporterManager.java rowFor()]
#
# Usage:  lint-wb-external-ord-value.sh [--strict] <wb-src-root>
#   Scans *.java under <wb-src-root> (dot-dirs pruned). Comments are stripped and method
#   boundaries are found with the shared PEAK-depth parser (toolbelt/lib/method-boundary.sh
#   mb_strip/mb_parse -- fragment rule: edit the shared fragment, never re-implement a
#   parallel parser here). Per METHOD:
#     1. a candidate ORD-resolve line: `<ident>Ord()<opt-space>.get(` (e.g.
#        `getTargetOrd().get(`) or `<ident>[Oo]rd<opt-space>.get(` (e.g. `folderOrd.get(`);
#     2. AND, anywhere in the SAME method, a live-value/status read off the resolved
#        object: `.get("out")`, `getOutStatusValue(`, or `BStatusValue`;
#     3. AND the SAME method has NO `registerForComponentEvents(` call anywhere in its
#        body.
#   All three true -> WARN at the resolve line.
#   Row:  WARN  lint-wb-external-ord-value  <file>:<line>  <method>() resolves an external
#         ORD and reads its value/status with no registerForComponentEvents(target,0)
#         lease in scope -- subject-subtree depth does not cover it; add loadSlots() +
#         registerForComponentEvents(target,0) for the resolved target
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env
#
# Known limitations (advisory heuristic, documented per kit style):
#   - a resolve line and its value-read split across TWO methods (helper extraction) is
#     a false NEGATIVE -- this lint scans one method's body at a time.
#   - a `registerForComponentEvents(` call ANYWHERE in the same method suppresses the
#     WARN even if it targets an unrelated component -- this lint does not track which
#     BComponent variable is passed to which call (a rare false NEGATIVE, not a
#     false-positive risk).
#   - an ORD resolved purely to navigate a CONTAINER (e.g. `folderOrd.get(...)` used only
#     to find a folder to add a new point into) is exempted naturally: it only WARNs when
#     the SAME method also reads a live value/status off the resolved object.
# VCS-free by design (kit-links L2). LC_ALL=C.
# Mutation: WEO2 -- drop the registerForComponentEvents(-absence check so a compliant method still WARNs
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/method-boundary.sh"
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-wb-external-ord-value: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-wb-external-ord-value.sh [--strict] <wb-src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-wb-external-ord-value.sh [--strict] <wb-src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-wb-external-ord-value: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

printf '%s\n' "$MB_AWK" > "$_TMP/method-boundary.awk"
cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  mb_strip(lines, NR, slines)

  n_meth = mb_parse(slines, NR, meth_start, meth_end, meth_name)

  for (mi = 0; mi < n_meth; mi++) {
    ms = meth_start[mi]; me = meth_end[mi]; mn = meth_name[mi]

    # 1. Locate an ORD-resolve line inside this method.
    resolve_line = 0
    for (i = ms; i <= me; i++) {
      ln = slines[i]
      if (match(ln, /[A-Za-z_][A-Za-z0-9_]*Ord\(\)[[:space:]]*\.get\(/) ||
          match(ln, /[A-Za-z_][A-Za-z0-9_]*[Oo]rd[[:space:]]*\.get\(/)) {
        resolve_line = i
        break
      }
    }
    if (resolve_line == 0) continue

    # 2. Same method must also read a live value/status off the resolved object.
    read_live = 0
    for (i = ms; i <= me; i++) {
      ln = slines[i]
      if (index(ln, ".get(\"out\")") > 0 || index(ln, "getOutStatusValue(") > 0 ||
          index(ln, "BStatusValue") > 0) { read_live = 1; break }
    }
    if (!read_live) continue

    # 3. Same method must have NO registerForComponentEvents( call anywhere.
    has_register = 0
    for (i = ms; i <= me; i++) {
      if (index(slines[i], "registerForComponentEvents(") > 0) { has_register = 1; break }
    }
    if (has_register) continue

    printf "WARN  lint-wb-external-ord-value  %s:%d  %s() resolves an external ORD and reads its" \
      " value/status with no registerForComponentEvents(target,0) lease in scope -- subject-subtree" \
      " depth does not cover it; add loadSlots() + registerForComponentEvents(target,0) for the" \
      " resolved target\n", FILE, resolve_line, mn
  }
}
AWKEOF

had_warn=0
while IFS= read -r f; do
  out=$(awk -f "$_TMP/method-boundary.awk" -f "$_TMP/main.awk" -v FILE="$f" "$f" 2>/dev/null)
  if [ -n "$out" ]; then
    printf '%s\n' "$out"
    had_warn=1
  fi
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print | LC_ALL=C sort)

if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
