#!/usr/bin/env bats
# RED-FIRST pins for the retro/ticket loop (campaign 8 PR16, wave3 D13/R16). Content shape from
# companero's draft (niagara-research sources/probes/2026-09-05-c8-pr16-retro-loop-draft.md).
#
#   new-retro.sh <module|kit> <slug>  atomically writes retros/<date>-<slug>.md (stub: marker first
#     line, 4 sections, Delta count) + appends the INDEX.md row + sets retro_pending: true in
#     BUILD-STATE.md. Idempotent: a second run with the same slug refuses (file exists) and does NOT
#     duplicate the INDEX row.
#   kit-ticket.sh "<one-line>"  opens a kit issue, or writes retros/tickets/<date>-<slug>.md offline
#     (gh absent/unauthenticated) with a SKIP row and exit 0 — never exit 1 for missing gh.
#
# The scripts write to the kit at $KIT (or cwd); the tests run them via `env -C <tmpkit> KIT=<tmpkit>`
# so they never touch the real kit (a testable-seam requirement for the apply worker to honor).
#
# RED today: new-retro.sh / kit-ticket.sh do not exist -> every pin fails for the right reason (absent).
# NAMED MUTATION (R16.6): omit the INDEX duplicate guard -> RL2 sees a second INDEX row on the 2nd run.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  NR="$KIT/toolbelt/new-retro.sh"; KT="$KIT/toolbelt/kit-ticket.sh"
  TK="$BATS_TEST_TMPDIR/kit"; mkdir -p "$TK/retros"
  printf '| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n' > "$TK/retros/INDEX.md"
  printf '<!-- build-state.v1 -->\n## kit\nretro_required: true\nretro_pending: false\n' > "$TK/BUILD-STATE.md"
  DATE="$(date +%Y-%m-%d)"
}
nr() { run env -C "$TK" KIT="$TK" "$NR" "$@"; }
kt() { run env -C "$TK" KIT="$TK" "$@"; }   # first vararg is the command (for PATH control)

@test "RL1: new-retro.sh writes the stub — line 1 EXACTLY the marker + the 4 sections" {
  nr kit rl1slug
  [ "$status" -eq 0 ]
  F="$TK/retros/$DATE-rl1slug.md"
  [ -f "$F" ]
  [ "$(head -1 "$F")" = "<!-- review-status: pending -->" ]
  grep -qiE 'what happened' "$F" && grep -qiE 'evidence' "$F"
  grep -qiE 'proposed kit deltas' "$F" && grep -qiE 'lessons' "$F"
}

@test "RL2: a second run with the same slug does NOT duplicate the INDEX row (idempotent guard)" {
  nr kit rl2slug
  nr kit rl2slug
  [ "$(grep -c 'rl2slug' "$TK/retros/INDEX.md")" -eq 1 ]
}

@test "RL3: BUILD-STATE.md kit envelope flips retro_pending to true" {
  nr kit rl3slug
  grep -qE 'retro_pending:[[:space:]]*true' "$TK/BUILD-STATE.md"
}

@test "RL4: kit-ticket.sh with gh ABSENT -> exit 0, SKIP row, writes the offline fallback file" {
  NOGH="$TK/nogh"; mkdir -p "$NOGH"
  for t in bash sh env grep sed awk cat date dirname basename mkdir printf; do
    p="$(command -v "$t" 2>/dev/null)"; [ -n "$p" ] && ln -sf "$p" "$NOGH/"
  done
  run env -C "$TK" KIT="$TK" PATH="$NOGH" bash "$KT" "fix the timer leak"
  [ "$status" -eq 0 ]
  [[ "$output" == *"SKIP"* ]]
  ls "$TK"/retros/tickets/"$DATE"-*.md >/dev/null 2>&1
}

@test "RL5: Delta count in the stub == the INDEX deltas column == the deltas table row count" {
  nr kit rl5slug
  F="$TK/retros/$DATE-rl5slug.md"
  dc=$(grep -oiE 'Delta count[^0-9]*[0-9]+' "$F" | grep -oE '[0-9]+' | head -1)
  idx=$(grep 'rl5slug' "$TK/retros/INDEX.md" | awk -F'|' '{gsub(/ /,"",$6); print $6}')
  [ -n "$dc" ] && [ "$dc" = "$idx" ]
}

@test "RL6: the appended INDEX row matches the fixed column format" {
  nr kit rl6slug
  grep -qE "^\|[[:space:]]*$DATE-rl6slug\.md[[:space:]]*\|[[:space:]]*kit[[:space:]]*\|[[:space:]]*$DATE[[:space:]]*\|[[:space:]]*pending[[:space:]]*\|[[:space:]]*[0-9]+[[:space:]]*\|" "$TK/retros/INDEX.md"
}
