#!/usr/bin/env bash
# check-bats-skips.sh — the global bats SKIP allowlist assertion (audit A6; CI bats job).
#
# A bats test that skips reads green. A suite that starts skipping for a NEW reason — a tool install that
# broke, a fixture that went missing, a guard that now always trips — passes CI unseen. This check reads the
# TAP of a bats run and fails when any `# skip <reason>` matches no line of the allowlist.
#
# Usage:  check-bats-skips.sh <bats.tap> <allowlist>
#   allowlist: one extended regular expression per line, matched against the WHOLE reason (grep -xE);
#   `#` lines and blank lines are ignored; a skip with no reason is the empty string. The committed list is
#   tests/bats-skip-allowlist.txt — add a line only for a skip that is legitimate on a CI runner, with a
#   comment saying why.
# Rows:   UNLISTED skip: <ok N test name> — <reason>   (one per unlisted skip, stdout)
# Exit:   0 every skip is allowlisted · 1 an unlisted skip · 3 usage, an unreadable TAP or allowlist, a TAP
#         with no `1..N` plan (bats never ran), or an allowlist line that is not a valid ERE
set -u

die3() { printf 'check-bats-skips: %s\n' "$1" >&2; exit 3; }
[ $# -eq 2 ] || die3 "usage: check-bats-skips.sh <bats.tap> <allowlist>"
tap=$1 allow=$2
for f in "$tap" "$allow"; do
  if [ ! -f "$f" ] || [ ! -r "$f" ]; then die3 "cannot read $f"; fi
done
grep -qE '^1\.\.[0-9]+$' "$tap" || die3 "no 1..N plan in $tap (bats did not run)"

patterns=$(grep -vE '^[[:space:]]*(#|$)' "$allow" || true)
while IFS= read -r p; do
  [ -n "$p" ] || continue
  # Mutation: SK6-bad-ere -- a pattern grep rejects (exit 2) would otherwise never match and hide nothing
  grep -qE -- "$p" </dev/null; [ $? -le 1 ] || die3 "allowlist line is not a valid ERE: $p"
done <<<"$patterns"

skips=0 unlisted=0
# A skip line is `ok N <name> # skip[ <reason>]`; the LAST ' # skip' splits name and reason.
while IFS=$'\t' read -r test reason; do
  skips=$((skips + 1))
  # Mutation: SK3-unanchored -- dropping -x matches a pattern anywhere inside a longer reason
  if [ -z "$patterns" ] || ! grep -qxE -e "$patterns" <<<"$reason"; then
    unlisted=$((unlisted + 1))
    printf 'UNLISTED skip: %s — %s\n' "$test" "${reason:-(no reason)}"
  fi
done < <(awk '/^ok [0-9]+ / {
             i = 0; s = $0
             while ((j = index(substr(s, i + 1), " # skip")) > 0) i += j
             if (i == 0) next
             rest = substr(s, i + 7)
             if (rest != "" && substr(rest, 1, 1) != " ") next
             sub(/^ /, "", rest)
             printf "%s\t%s\n", substr(s, 1, i - 1), rest
           }' "$tap")

if [ "$unlisted" -gt 0 ]; then
  printf 'check-bats-skips: %d of %d skip(s) match no line of %s\n' "$unlisted" "$skips" "$allow" >&2
  exit 1
fi
printf 'check-bats-skips: %d skip(s), all allowlisted\n' "$skips"
exit 0
