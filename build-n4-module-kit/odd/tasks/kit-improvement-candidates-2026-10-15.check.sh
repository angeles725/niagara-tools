#!/usr/bin/env bash
# No-loss proof for kit-improvement-candidates-2026-10-15.md.
# Extracts every delta ID from the 2026-10-15 retros and checks that each appears exactly once in the
# register's Appendix A, that Appendix A holds no unknown ID, and that every mapped C-id has a heading.
# Read-only. Exit 0 = missing=0, extra=0, dup=0, shape ok, every C-id defined.
set -euo pipefail
export LC_ALL=C.UTF-8

here=$(cd "$(dirname "$0")" && pwd)
kit=$(cd "$here/../.." && pwd)
reg="$here/kit-improvement-candidates-2026-10-15.md"
retros="$kit/retros"

src=$(mktemp)
app=$(mktemp)
trap 'rm -f "$src" "$app"' EXIT

files=0
shape_bad=0
for f in "$retros"/2026-10-15-*.md; do
  files=$((files + 1))
  stem=$(basename "$f" .md)
  slug=${stem#2026-10-15-}
  declared=$(sed -n 's/^\*\*Delta count\*\*: *\([0-9][0-9]*\).*/\1/p' "$f")
  ids=$(grep -E '^\| *(Δ)?[0-9]+ *\|' "$f" | sed -E 's/^\| *(Δ)?([0-9]+) *\|.*/\2/' || true)
  n=$(printf '%s\n' "$ids" | grep -c . || true)
  if [ "$n" != "$declared" ]; then
    echo "SHAPE $stem: parsed $n rows, declared ${declared:-none}"
    shape_bad=1
  fi
  printf '%s\n' "$ids" | grep . | sed "s/^/${slug}#Δ/" >> "$src"
done
sort -o "$src" "$src"

# Appendix A rows: first cell only, from the Appendix A heading to the next level-2 heading.
# shellcheck disable=SC2016  # the backticks are literal Markdown, not a command substitution
awk '/^## Appendix A/{on=1; next} on && /^## /{on=0} on' "$reg" \
  | sed -n 's/^| `\([a-z0-9-]*#Δ[0-9]*\)` |.*/\1/p' | sort > "$app"

total=$(wc -l < "$src")
mapped=$(comm -12 "$src" <(sort -u "$app") | wc -l)
missing=$(comm -23 "$src" <(sort -u "$app"))
extra=$(comm -13 "$src" <(sort -u "$app"))
dups=$(uniq -d "$app")

# Every C-id named in Appendix A must have a '### Cnn' heading, and its Appendix A tier must equal the
# tier section ('## P0 …' … '## P3 …') that heading sits in.
undef=""
tiermismatch=""
heading_tiers=$(awk '/^## P[0-3] /{t=$2} /^### C[0-9]+ /{print $2, t}' "$reg")
while read -r c t; do
  ht=$(printf '%s\n' "$heading_tiers" | awk -v c="$c" '$1 == c {print $2}')
  if [ -z "$ht" ]; then undef="$undef $c"; elif [ "$ht" != "$t" ]; then tiermismatch="$tiermismatch $c($t≠$ht)"; fi
done < <(awk '/^## Appendix A/{on=1; next} on && /^## /{on=0} on' "$reg" | grep -E '^\| `' \
           | awk -F'|' '{gsub(/ /, "", $5); gsub(/ /, "", $6); print $5, $6}' | sort -u)

# Per-tier source-delta counts (column 6 of Appendix A).
tiers=$(awk '/^## Appendix A/{on=1; next} on && /^## /{on=0} on' "$reg" | grep -E '^\| `' \
          | awk -F'|' '{gsub(/ /, "", $6); print $6}' | sort | uniq -c | awk '{printf "%s=%s ", $2, $1}')
citems=$(grep -cE '^### C[0-9]+ ' "$reg")

echo "retro files: $files"
echo "source deltas: $total"
echo "mapped: $mapped"
echo "missing: $(printf '%s' "$missing" | grep -c . || true)"
echo "extra (in appendix, not in retros): $(printf '%s' "$extra" | grep -c . || true)"
echo "duplicate appendix rows: $(printf '%s' "$dups" | grep -c . || true)"
echo "undefined C-ids:${undef:- none}"
echo "tier mismatches:${tiermismatch:- none}"
echo "C-items defined: $citems"
echo "source deltas per tier: $tiers"
[ -z "$missing" ] || { echo "MISSING:"; printf '%s\n' "$missing"; }
[ -z "$extra" ] || { echo "EXTRA:"; printf '%s\n' "$extra"; }
[ -z "$dups" ] || { echo "DUP:"; printf '%s\n' "$dups"; }

if [ -z "$missing" ] && [ -z "$extra" ] && [ -z "$dups" ] && [ -z "$undef" ] && [ -z "$tiermismatch" ] && [ "$shape_bad" -eq 0 ]; then
  echo "RESULT: PASS (missing=0)"
else
  echo "RESULT: FAIL"
  exit 1
fi
