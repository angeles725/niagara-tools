#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# RED-FIRST bats for sweep-fold-audit.sh (Campaign 6 T5a.1/T5a.2).
#
# Frozen contract: sweep-fold-audit.sh [--strict] <INDEX.md> <kit-root>
#   exit 0   clean or WARN-only
#   exit 1   uncited rows found AND --strict
#   exit 3   usage/env error
#
# Corpus: every *.md under kit-root excluding */retros/* and INDEX.md.
# A folded row is cited when a token T from the corpus satisfies:
#   ${#T} >= 6 AND case "-$stem-" in *"-$T-"*)
# where stem = filename minus leading YYYY-MM-DD- minus .md.
#
# Named mutations (run post-green, revert each — a listed test changes outcome):
#   F3: drop */retros/* exclusion -> self-ref-only corpus-found -> WARN disappears -> F3 fails
#   F4: exact-stem match instead of hyphen-segment -> 5rooms != dashboardpan-5rooms -> WARN added -> F4 fails
#   F6: plain substring instead of segment-aligned -> ender-doors substring of detail-render-doors -> WARN disappears -> F6 fails
#
# Fixture layout (tests/fixtures/fold-audit/):
#   INDEX.md          — 5 folded + 1 pending row
#   core.md           — corpus: cites rt-hardening, 5rooms, ender-doors
#   retros/self-cite.md — self-citation of self-ref-only (excluded from corpus)

setup() {
  FIXDIR="$(cd "$BATS_TEST_DIRNAME" && pwd)/fixtures/fold-audit"
  SCRIPT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/sweep-fold-audit.sh"
  INDEX="$FIXDIR/INDEX.md"
  KITROOT="$FIXDIR"
}

@test "F1: folded uncited retro emits WARN and exits 0" {
  run "$SCRIPT" "$INDEX" "$KITROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"fold-audit: WARN"* ]]
  [[ "$output" == *"2026-01-01-orphan-report.md"* ]]
}

@test "F2: --strict exits 1 when uncited retros exist" {
  run "$SCRIPT" --strict "$INDEX" "$KITROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"fold-audit: WARN"* ]]
}

@test "F3: citation only in retros/ still produces WARN (corpus excludes retros/)" {
  run "$SCRIPT" "$INDEX" "$KITROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"2026-01-03-self-ref-only.md"* ]]
}

@test "F4: abbreviated token 5rooms credits stem dashboardpan-5rooms (no spurious WARN)" {
  run "$SCRIPT" "$INDEX" "$KITROOT"
  [ "$status" -eq 0 ]
  [[ "$output" != *"2026-01-04-dashboardpan-5rooms"* ]]
}

@test "F5: pending rows are ignored (no WARN for pending-orphan)" {
  run "$SCRIPT" "$INDEX" "$KITROOT"
  [[ "$output" != *"2026-01-05-pending-orphan"* ]]
}

@test "F6: token ender-doors does NOT credit stem detail-render-doors (segment-aligned, not substring)" {
  run "$SCRIPT" "$INDEX" "$KITROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"2026-01-06-detail-render-doors.md"* ]]
}

# --deltas-since <YYYY-MM-DD>: Δ-granular reconcile check (the RECONCILED check of the
# 2026-10-02 fold campaign, committed). Every folded row dated on/after the cutoff must have
# each of its Δ1..Δn cited as `[ev: retro <stem> Δk]` — or explicitly deferred as
# `[deferred: retro <stem> Δk]` — in the corpus (exact stem, exact Δ number).
_delta_fixture() {
  D="$BATS_TEST_TMPDIR/kit"
  mkdir -p "$D/retros"
  {
    printf '| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n'
    printf '| 2026-09-01-old-style.md | kit | 2026-09-01 | folded | 2 |\n'
    printf '| 2026-10-01-full.md | kit | 2026-10-01 | folded | 2 |\n'
    printf '| 2026-10-01-gap.md | kit | 2026-10-01 | folded | 3 |\n'
    printf '| 2026-10-01-deferred.md | kit | 2026-10-01 | folded | 2 |\n'
    printf '| 2026-10-02-campaign-close.md | kit | 2026-10-02 | folded | 1 |\n'
    printf '| 2026-10-02-one.md | kit | 2026-10-02 | folded | 1 |\n'
    printf '| 2026-10-02-later.md | kit | 2026-10-02 | pending | 4 |\n'
  } > "$D/retros/INDEX.md"
  {
    printf 'Doc [ev: retro old-style] cites by stem only (before the cutoff).\n'
    printf 'Rules [ev: retro full Δ1] and [ev: retro full Δ2].\n'
    printf 'Rule [ev: retro gap Δ1] and [ev: retro gap Δ3].\n'
    printf 'Rule [ev: retro deferred Δ1]; out of repo [deferred: retro deferred Δ2].\n'
    printf 'Rule [ev: retro retro-campaign-close Δ1] names another stem.\n'
    printf 'Rule [ev: retro one Δ10] is not Δ1.\n'
  } > "$D/core.md"
}

@test "F7: --deltas-since flags each uncited Δ of a folded row on/after the cutoff; --strict exits 1" {
  _delta_fixture
  run "$SCRIPT" --deltas-since 2026-09-24 "$D/retros/INDEX.md" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" == *"fold-audit: WARN 2026-10-01-gap.md Δ2 not cited"* ]]
  [ "$(printf '%s\n' "$output" | grep -c 'Δ[0-9]* not cited')" -eq 3 ]
  run "$SCRIPT" --strict --deltas-since 2026-09-24 "$D/retros/INDEX.md" "$D"
  [ "$status" -eq 1 ]
}

@test "F8: a [deferred: retro <stem> Δk] marker accounts for Δk; rows before the cutoff and pending rows are not Δ-checked" {
  _delta_fixture
  run "$SCRIPT" --deltas-since 2026-09-24 "$D/retros/INDEX.md" "$D"
  [[ "$output" != *"2026-10-01-deferred.md Δ"* ]]
  [[ "$output" != *"2026-10-01-full.md Δ"* ]]
  [[ "$output" != *"2026-09-01-old-style.md Δ"* ]]
  [[ "$output" != *"2026-10-02-later.md Δ"* ]]
  [[ "$output" == *"deltas: 9 checked, 3 not cited"* ]]
}

@test "F9: the Δ citation needs the exact stem and the exact Δ number (no segment or prefix credit)" {
  # Mutation: F9 -- matching "<stem> Δk" as a substring credits retro-campaign-close Δ1 to
  # campaign-close and Δ10 to Δ1, so both WARN rows disappear.
  _delta_fixture
  run "$SCRIPT" --deltas-since 2026-09-24 "$D/retros/INDEX.md" "$D"
  [[ "$output" == *"fold-audit: WARN 2026-10-02-campaign-close.md Δ1 not cited"* ]]
  [[ "$output" == *"fold-audit: WARN 2026-10-02-one.md Δ1 not cited"* ]]
}

@test "F10: --deltas-since rejects a malformed date (exit 3)" {
  _delta_fixture
  run "$SCRIPT" --deltas-since 2026-9-1 "$D/retros/INDEX.md" "$D"
  [ "$status" -eq 3 ]
}
