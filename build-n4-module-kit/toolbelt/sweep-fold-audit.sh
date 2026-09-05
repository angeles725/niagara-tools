#!/usr/bin/env bash
# sweep-fold-audit.sh — Audit folded retros for [ev: retro] citation coverage.
#
# Every folded row in INDEX.md must have at least one [ev: retro <token>]
# citation in the core kit corpus. A missing citation indicates that the
# retro's lesson was never folded into a kit file — this audit makes that
# visible so it becomes a PR7 follow-up item, not a silent gap.
#
# Usage: sweep-fold-audit.sh [--strict] <INDEX.md> <kit-root>
# exit:  0 clean or WARN-only · 1 uncited rows AND --strict · 3 usage/env
#
# Corpus: every *.md under <kit-root> excluding */retros/* and INDEX.md.
# Token harvest: grep -ohE '\[ev: retro <token>' across corpus files.
# Stem: filename minus leading YYYY-MM-DD- and trailing .md.
# Token match (hyphen-segment aligned, floor 6 chars):
#   case "-$stem-" in *"-$T-"*)
# This script is VCS-free by design — version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*)
      echo "fold-audit: unknown flag: $1" >&2
      echo "usage: sweep-fold-audit.sh [--strict] <INDEX.md> <kit-root>" >&2
      exit 3
      ;;
    *) break ;;
  esac
done

[ $# -eq 2 ] || {
  echo "usage: sweep-fold-audit.sh [--strict] <INDEX.md> <kit-root>" >&2
  exit 3
}

INDEX="$1"
KITROOT="$2"

[ -f "$INDEX" ]   || { echo "fold-audit: INDEX not found: $INDEX" >&2;    exit 3; }
[ -d "$KITROOT" ] || { echo "fold-audit: kit-root not found: $KITROOT" >&2; exit 3; }

# Harvest citation tokens from corpus (VCS-free, pure file scan).
# retros/ excluded: self-citations must not mask genuinely uncited entries.
# INDEX.md excluded: cross-links in the index are not proof of folded content.
# Token format: [ev: retro <token>] where token is [A-Za-z0-9][A-Za-z0-9._-]+
# The 6-char floor removes template placeholders like [ev: retro X].
TOKENS=$(find "$KITROOT" -name '*.md' \
  ! -path "*/retros/*" \
  ! -name 'INDEX.md' \
  -exec grep -ohE '\[ev: retro [A-Za-z0-9][A-Za-z0-9._-]*' {} + 2>/dev/null \
  | sed 's/\[ev: retro //' \
  | sort -u)

NFOLDED=0; NCITED=0; NUNCITED=0; HAS_WARN=0

while IFS= read -r line; do
  # Only pipe-delimited data rows containing a dated retro filename
  case "$line" in \|*) : ;; *) continue ;; esac
  printf '%s\n' "$line" | grep -qE '[0-9]{4}-[0-9]{2}-[0-9]{2}[^|]*\.md' || continue
  fname=$(printf '%s\n' "$line" | grep -oE '[A-Za-z0-9._-]+\.md' | grep -vx 'INDEX.md' | head -1)
  [ -n "$fname" ] || continue
  rstatus=$(printf '%s\n' "$line" | awk -F'|' '{
    for(i=1;i<=NF;i++){gsub(/^ +| +$/,"",$i); if($i=="pending"||$i=="folded"){print $i; exit}}
  }')
  [ "$rstatus" = "folded" ] || continue
  NFOLDED=$((NFOLDED + 1))

  # Stem: strip leading YYYY-MM-DD- date prefix and trailing .md
  base="${fname%.md}"
  stem=$(printf '%s' "$base" | sed -E 's/^[0-9]{4}-[0-9]{2}-[0-9]{2}-//')

  # Check each harvested token for hyphen-segment alignment with this stem.
  # A token T credits the stem when "-T-" appears as a segment in "-stem-".
  # Example: token "5rooms" credits stem "dashboardpan-5rooms" because
  #          "-dashboardpan-5rooms-" contains the segment "-5rooms-".
  # Counter-example: token "ender-doors" does NOT credit "detail-render-doors"
  #          because "-detail-render-doors-" has no segment "-ender-doors-"
  #          ("ender" is mid-segment inside "render").
  matched=0
  match_count=0
  matched_tokens=""
  while IFS= read -r T; do
    [ -n "$T" ] || continue
    [ "${#T}" -ge 6 ] || continue
    case "-${stem}-" in
      *"-${T}-"*)
        matched=1
        match_count=$((match_count + 1))
        matched_tokens="${matched_tokens} ${T}"
        ;;
    esac
  done <<< "$TOKENS"

  if [ "$matched" -eq 0 ]; then
    printf 'fold-audit: WARN %s folded with no [ev: retro …] citation\n' "$fname"
    NUNCITED=$((NUNCITED + 1))
    HAS_WARN=1
  else
    NCITED=$((NCITED + 1))
    if [ "$match_count" -gt 1 ]; then
      printf 'fold-audit: NOTE ambiguous citation token credits %s:%s\n' "$fname" "$matched_tokens"
    fi
  fi
done < "$INDEX"

printf 'fold-audit: %d folded, %d cited, %d uncited\n' "$NFOLDED" "$NCITED" "$NUNCITED"

if [ "$HAS_WARN" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
