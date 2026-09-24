#!/usr/bin/env bash
# lint-inert-coordination.sh — flags a coordinator whose unit list is STRUCTURALLY singleton
# (e.g. `units()` returns `Collections.singletonList(getParent())` or a one-element
# `Arrays.asList(...)`) while the SAME class still declares queue/token/stagger coordination
# state -- the machinery cross-unit-serializes over exactly one unit and is dead code.
#
# Shape (Δ5, PANCCADIA 2026-09-23): PR1c moved `BDefrostController` under each
# `BEvaporatorUnit`. `units()` now returns a single-element list, so the token/queue/
# `staggerDelay` machinery never engages -- a room with two electric resistances can defrost
# both at once, exactly the coordination the class exists to prevent. The class's own comment
# admits it ("staggerDelay is inert: with one unit in the list the waitingQueue is always
# empty") but nothing flags it as a defect. See types/issues-and-gotchas.md §J1 and
# types/logic.md "RT control logic" for the fix (coordinate at the parent that actually
# aggregates multiple units, or drop the dead machinery).
# [ev: retro panccadia-defrost-sequencing-hmi-reload-deltas Δ5]
# [ev: corpus BDefrostController.units() ~990-1001; staggerDelay ~309-330; waitingQueue ~1060]
#
# Usage:  lint-inert-coordination.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned). Run on any -rt profile with Java sources.
#   Comments are line-stripped (`//...`, naive -- see limitations). Per file:
#     1. locate a `<ret-type> units()` method declaration and brace-extract its body forward;
#     2. the body must return a STRUCTURALLY singleton list: `singletonList(` anywhere in the
#        body, OR `Arrays.asList(` with no comma between the call's opening paren and the end
#        of its line (single-argument heuristic -- see limitations);
#     3. AND the class (whole file) must declare at least one coordination-state identifier:
#        a `Deque`/`Queue` typed field, or an identifier containing `Token` or `Stagger`
#        (case-sensitive substrings, matching Java's camelCase convention: `waitingQueue`,
#        `acquireDefrostToken`, `staggerDelay`).
#   All true -> WARN at the units() declaration line.
#   Row:  WARN  lint-inert-coordination  <file>:<line>  units() returns a structurally singleton
#         list while <ids> (queue/token/stagger state) are still declared -- the coordination
#         machinery is dead code after the per-unit move; coordinate at the parent that
#         aggregates multiple units, or drop the unused fields
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env
#
# Known limitations (advisory heuristic, documented per kit style):
#   - the singleton-return check only looks for `singletonList(`/`Arrays.asList(` text ANYWHERE
#     in the body, not that it is the actual `return` expression -- a comment or an unrelated
#     local variable using the same call shape is a false-positive risk, cheap to accept per
#     kit style (mirrors lint-wb-external-ord-value's same-method text-presence checks).
#   - the `Arrays.asList(` single-argument check is a same-line no-comma-to-end-of-line
#     heuristic (no real paren-depth tracking): a multi-line call, or a two-argument call whose
#     first argument itself contains no comma but a later unrelated comma appears further on
#     the SAME line, can mis-classify -- rare in this shape (a coordinator's argument is always
#     `getParent()`/a single component reference).
#   - `Token`/`Stagger` substring matching is case-sensitive on the capitalized form used by
#     Java identifiers after the first word (e.g. `defrostToken`, `getStaggerDelay`); an
#     all-lowercase or ALL-CAPS identifier is a false-negative risk.
#   - a `Deque`/`Queue` field used for something UNRELATED to unit coordination (rare in a
#     control component) would false-positive; only fires when a singleton units() ALSO exists,
#     which narrows the risk considerably.
# VCS-free by design (kit-links L2). LC_ALL=C.
# Mutation: ICO2 -- drop the coordination-state (queue/token/stagger) requirement so any singleton units() WARNs even with no dead machinery
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-inert-coordination: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-inert-coordination.sh [--strict] <src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-inert-coordination.sh [--strict] <src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-inert-coordination: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT

cat > "$_TMP/main.awk" << 'AWKEOF'
{ lines[NR] = $0 }

END {
  for (i = 1; i <= NR; i++) { ln = lines[i]; sub(/\/\/.*$/, "", ln); lines[i] = ln }

  # 1. Locate a `units()` method declaration.
  decl_line = 0
  for (i = 1; i <= NR; i++) {
    if (match(lines[i], /[A-Za-z_][A-Za-z0-9_<>\[\], ]*[[:space:]]units\(\)/)) { decl_line = i; break }
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
  if (!started) exit 0   # declaration only (interface/abstract) -- nothing to check

  # 2. Structurally-singleton return shape.
  is_singleton = 0
  for (i = decl_line; i <= body_end; i++) {
    ln = lines[i]
    if (index(ln, "singletonList(") > 0) { is_singleton = 1; break }
    p = index(ln, "Arrays.asList(")
    if (p > 0) {
      tail = substr(ln, p + length("Arrays.asList("))
      # Heuristic (documented limitation): no comma from the call open to end-of-line means a
      # single argument -- a nested no-comma call (e.g. `(Unit) getParent()`) still matches.
      if (index(tail, ",") == 0) { is_singleton = 1; break }
    }
  }
  if (!is_singleton) exit 0

  # 3. Coordination-state identifiers anywhere in the file: Deque/Queue typed field, or an
  #    identifier containing Token or Stagger.
  found = ""
  for (i = 1; i <= NR; i++) {
    ln = lines[i]
    if (match(ln, /(Deque|Queue)[[:space:]]*</)) {
      m = match(ln, /[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=/)
      name = ""
      if (m) { seg = substr(ln, RSTART, RLENGTH); sub(/[[:space:]]*=$/, "", seg); split(seg, parts, " "); name = parts[length(parts)] }
      if (name == "") name = "(Deque/Queue field)"
      if (index(found, name) == 0) found = found (found == "" ? "" : ", ") name
    }
    if (match(ln, /[A-Za-z_][A-Za-z0-9_]*Token[A-Za-z0-9_]*/)) {
      seg = substr(ln, RSTART, RLENGTH)
      if (index(found, seg) == 0) found = found (found == "" ? "" : ", ") seg
    }
    if (match(ln, /[A-Za-z_][A-Za-z0-9_]*Stagger[A-Za-z0-9_]*/)) {
      seg = substr(ln, RSTART, RLENGTH)
      if (index(found, seg) == 0) found = found (found == "" ? "" : ", ") seg
    }
  }
  if (found == "") exit 0

  printf "WARN  lint-inert-coordination  %s:%d  units() returns a structurally singleton list" \
    " while %s (queue/token/stagger state) are still declared -- the coordination machinery is" \
    " dead code after the per-unit move; coordinate at the parent that aggregates multiple" \
    " units, or drop the unused fields\n", FILE, decl_line, found
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
