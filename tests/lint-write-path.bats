#!/usr/bin/env bats
# RED-FIRST pins for lint-write-path.sh (campaign 8 PR19, wave3 R19). Every @NiagaraProperty with
# Flags.OPERATOR under <module-root>/src must have a ROW in docs/write-path-matrix.md naming it; a
# missing row FAILs naming the slot. Row-presence only (D16: it does not validate the test name, and
# CHECK12 is bog-audit's concern). A comment mention does NOT count as a row.
#
# SURFACE: lint-write-path.sh <module-root> [--bog <config.bog>]
#   Row `<check>  FAIL  <slot>  no write-path-matrix row`; exit 0 all covered / 1 any uncovered / 3 usage.
#
# RED today: lint-write-path.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (post-green, R19.6): remove a covered slot's matrix row -> that slot appears in FAIL.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LW="$KIT/toolbelt/lint-write-path.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-write-path"
}

@test "WP1: every OPERATOR slot has a matrix row -> exit 0 (no FAIL)" {
  run "$LW" "$FX/covered"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "WP2: an OPERATOR slot with no matrix row FAILs naming it (hoaMode), exit 1" {
  run "$LW" "$FX/uncovered"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"hoaMode"* ]]
}

@test "WP3: a slot mentioned only in a comment does NOT count as covered (still FAIL)" {
  # hoaMode appears in an HTML comment in uncovered's matrix but not as a table row.
  run "$LW" "$FX/uncovered"
  [[ "$output" == *"hoaMode"* ]]                 # comment mention did not satisfy the row requirement
  [[ "$output" != *"setpoint"* ]]                # setpoint HAS a real row -> not flagged
}

@test "WP4: a non-operator slot (internalState) is not required to have a row" {
  run "$LW" "$FX/uncovered"
  [[ "$output" == *"hoaMode"* ]]                 # anchor: the tool ran and flagged the real uncovered slot
  [[ "$output" != *"internalState"* ]]           # READONLY slot -> not a write path
}

@test "WP-prune: an OPERATOR slot under src/.deploy-baseline/ is NOT counted (D9b dot-dir prune)" {
  run "$LW" "$FX/uncovered"
  [[ "$output" == *"hoaMode"* ]]                 # anchor: the scan ran (hoaMode flagged) ...
  [[ "$output" != *"staleKnob"* ]]               # ... but the stale baseline copy is pruned, never required
}

@test "WP-usage: no <module-root> argument -> exit 3" {
  run "$LW"
  [ "$status" -eq 3 ]
}

@test "WP-smoke: client repo deed38c (22 dashboard-writable vs 13 matrix rows) FAILs uncovered (SKIP if absent)" {
  ROOT="${C8_WRITEPATH_ROOT:-}"
  [ -n "$ROOT" ] && [ -d "$ROOT" ] || skip "no client module root (set C8_WRITEPATH_ROOT to a deed38c ColdRoomPan/CompPan root)"
  run "$LW" "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}
