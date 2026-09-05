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
