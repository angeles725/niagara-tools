#!/usr/bin/env bash
# check-facets-card.sh — diffs the BFacets key reference card (types/logic-authoring.md) against the BFacets
# key constants, in declaration order.
#
# The card was hand-copied once and drifted (makeDouble, 17 keys that are not constants). Pass the 4.15.x
# source (BFacets.java: every `public static final String NAME = "key";`) or a keys file (`NAME=key` lines,
# `#` comments allowed; the kit pins tests/fixtures/facets-card/BFacets-4.15.6.20.keys).
# Card rows are `| \`NAME\` | \`"key"\` | ...` lines from the `## BFacets key reference card` heading to the
# first `**Not` line or the next `## ` heading.
# [ev: retro devguide415-runtime-api-corrections Δ8]
#
# Usage:  check-facets-card.sh <card.md> <BFacets.java | keys-file>
# Output: one line per difference — `MISSING NAME=key` (in the source, not the card), `EXTRA NAME=key`
#         (in the card, not the source), `ORDER card #n NAME=key != source NAME=key` — then a summary line.
# Exit:   0 match · 1 any difference · 3 usage/env (unreadable file, or no keys parsed on either side)
# VCS-free by design (kit-links L2).
set -u
LC_ALL=C
export LC_ALL
[ $# -eq 2 ] || { printf 'usage: check-facets-card.sh <card.md> <BFacets.java | keys-file>\n' >&2; exit 3; }
card="$1" src="$2"
for f in "$card" "$src"; do
  [ -f "$f" ] && [ -r "$f" ] || { printf 'check-facets-card: cannot read %s\n' "$f" >&2; exit 3; }
done
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# shellcheck disable=SC2016  # the backticks are literal Markdown in the awk regex
awk '/^## BFacets key reference card/ { on = 1; next }
     on && (/^\*\*Not/ || /^## /) { exit }
     on && match($0, /^\| `[A-Z_]+` \| `"[A-Za-z]+"` \|/) {
       r = substr($0, RSTART, RLENGTH); gsub(/[|`" ]/, " ", r); split(r, a, " "); print a[1] "=" a[2] }' \
  "$card" > "$tmp/card" || { printf 'check-facets-card: cannot parse %s\n' "$card" >&2; exit 3; }
case "$src" in
  *.java) sed -nE 's/^[[:space:]]*public[[:space:]]+static[[:space:]]+final[[:space:]]+String[[:space:]]+([A-Z_]+)[[:space:]]*=[[:space:]]*"([A-Za-z]+)";.*/\1=\2/p' "$src" ;;
  *)      grep -E '^[A-Z_]+=[A-Za-z]+$' "$src" || true ;;
esac > "$tmp/src"

nc=$(grep -c . "$tmp/card" || true)
ns=$(grep -c . "$tmp/src" || true)
[ "$nc" -gt 0 ] || { printf 'check-facets-card: no card rows parsed from %s\n' "$card" >&2; exit 3; }
[ "$ns" -gt 0 ] || { printf 'check-facets-card: no key constants parsed from %s\n' "$src" >&2; exit 3; }

bad=0
while IFS= read -r k; do grep -qxF "$k" "$tmp/card" || { printf 'MISSING %s\n' "$k"; bad=1; }; done < "$tmp/src"
while IFS= read -r k; do grep -qxF "$k" "$tmp/src" || { printf 'EXTRA %s\n' "$k"; bad=1; }; done < "$tmp/card"
if [ "$bad" -eq 0 ]; then
  i=0
  while IFS= read -r c && IFS= read -r s <&3; do
    i=$((i + 1))
    [ "$c" = "$s" ] || { printf 'ORDER card #%d %s != source %s\n' "$i" "$c" "$s"; bad=1; }
  done < "$tmp/card" 3< "$tmp/src"
fi
if [ "$bad" -eq 0 ]; then printf 'check-facets-card: %d keys match %s\n' "$nc" "$src"; exit 0; fi
printf 'check-facets-card: card %s differs from %s (%d card rows, %d source keys)\n' "$card" "$src" "$nc" "$ns"
exit 1
