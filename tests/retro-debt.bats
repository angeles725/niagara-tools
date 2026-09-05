#!/usr/bin/env bats
# RED-FIRST pins for E5 retro-debt aging (campaign 6 PR8).
# Contract: openspec/changes/build-n4-module-campaign6/math-models-e5.md (investigador1).
# Mirrors research-sdd's sweep-retros.sh "ESCALATED (aged 31d)" so both kits speak one language.
#
#   retro_debt(rows, today, max_age=30):
#     pending      = rows with status == "pending"     (ONLY pending rows age)
#     age_days(d)  = (today - d).days   from the retro's FILENAME date prefix, NOT the body
#     escalated    = age > max_age      (STRICTLY greater; age == max_age is NOT escalated)
#     total_pending, escalated_count, oldest_age (= "N/A" when 0 pending, distinct from 0)
#   `today` is INJECTED (--today), never the wall clock — so these pins are deterministic.
#
# SURFACE (provisional — the writer may pick `sweep-build-state.sh --age` or a `sweep-retro-debt.sh`;
# I rebase the invocation line, kept in one file):
#   sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age N] <retros-dir> <INDEX.md>
#   -> three prefixed stdout lines: total_pending=<n> / escalated_count=<n> / oldest_age=<n|N/A>
#
# RED today: no --age mode exists; sweep-build-state.sh sees extra args and exits 3 (usage), so no
# `total_pending=` line is printed and every pin FAILS for the right reason. Green once --age lands.
#
# MUTATIONS each pin must catch (post-green):
#   - `age >= max_age` instead of `>`: the 2026-08-06 (age 30) row -> escalated_count 1 != 0 (D2/D4).
#   - date from body/other field not the filename prefix: ages move off their pinned values.
#   - escalation ignores pending state (ages folded rows): the folded-only fixture -> total_pending 1 != 0.
#   - 0-pending -> 0 instead of "N/A": D1/D2 oldest_age.
#   - today read from the wall clock: the fixed --today pins fail on any day but 2026-09-05.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SWEEP="$KIT/toolbelt/sweep-build-state.sh"
  RETRODIR="$BATS_TEST_TMPDIR/retros"
  INDEX="$RETRODIR/INDEX.md"
  mkdir -p "$RETRODIR"
  : > "$INDEX"
  printf '| Retro file | review-status |\n|---|---|\n' > "$INDEX"
}

# add_row <date-stem> <status>: create the dated retro file and its INDEX row.
add_row() {
  printf '<!-- review-status: %s -->\n\n# retro\n' "$2" > "$RETRODIR/$1.md"
  printf '| %s.md | %s |\n' "$1" "$2" >> "$INDEX"
}
age() { run "$SWEEP" --age --today 2026-09-05 "$RETRODIR" "$INDEX"; }

@test "D1: no rows at all -> total_pending=0, escalated_count=0, oldest_age=N/A" {
  age
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "total_pending=0" ]
  [ "${lines[1]}" = "escalated_count=0" ]
  [ "${lines[2]}" = "oldest_age=N/A" ]
}

@test "D2: a FOLDED row never ages -> total_pending=0, oldest_age=N/A (not a big number)" {
  add_row "2026-01-01-old-folded" folded
  age
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "total_pending=0" ]
  [ "${lines[2]}" = "oldest_age=N/A" ]
}

@test "D3: a same-day pending row -> age 0 (total_pending=1, escalated_count=0, oldest_age=0)" {
  add_row "2026-09-05-today" pending
  age
  [ "${lines[0]}" = "total_pending=1" ]
  [ "${lines[1]}" = "escalated_count=0" ]
  [ "${lines[2]}" = "oldest_age=0" ]
}

@test "D4: exactly 30 days (2026-08-06) is NOT escalated (strict >) -> escalated_count=0, oldest_age=30" {
  add_row "2026-08-06-thirty" pending
  age
  [ "${lines[0]}" = "total_pending=1" ]
  [ "${lines[1]}" = "escalated_count=0" ]
  [ "${lines[2]}" = "oldest_age=30" ]
}

@test "D5: 31 days (2026-08-05) IS escalated -> escalated_count=1, oldest_age=31" {
  add_row "2026-08-05-thirtyone" pending
  age
  [ "${lines[0]}" = "total_pending=1" ]
  [ "${lines[1]}" = "escalated_count=1" ]
  [ "${lines[2]}" = "oldest_age=31" ]
}

@test "D6: mixed (2 pending + 1 old folded) -> total_pending=2, escalated_count=1, oldest_age=31" {
  add_row "2026-08-05-esc" pending
  add_row "2020-01-01-ancient" folded
  add_row "2026-09-01-recent" pending
  age
  [ "${lines[0]}" = "total_pending=2" ]
  [ "${lines[1]}" = "escalated_count=1" ]
  [ "${lines[2]}" = "oldest_age=31" ]
}
