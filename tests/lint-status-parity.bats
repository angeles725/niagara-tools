#!/usr/bin/env bats
# RED-FIRST pins for lint-status-parity.sh (Wave 3, LR2).
# Detects facade classes where the count of per-instance config slots (*Interval*, *Duration*, *Setpoint*;
# N>1) exceeds the count of status slots (*Status, *Active, *Start, *Since).
# Default: WARN exit 0. With --strict: FAIL exit 1.
#
# Real commissioning gap (PANCCADIA): ColdRoomPan had 3 per-evap defrost config slots but 1 shared
# defrostActive status — operators could not tell which evaporator was actively defrosting.
#
# RED today: lint-status-parity.sh does not exist -> every pin fails for the right reason (tool absent).
#
# NAMED MUTATION (post-green): remove the N>M asymmetry detection so all configs pass ->
# LSP-asymmetric stops firing, N=3/M=1 facades silently pass.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LSP="$KIT/toolbelt/lint-status-parity.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-status-parity"
  ONE="$BATS_TEST_TMPDIR/one"; mkdir -p "$ONE"
}
only() { rm -f "$ONE"/*.java; cp "$FX/$1" "$ONE/"; }

@test "LSP-asymmetric: AsymFacade (N=3 Duration, M=1 Active) -> WARN row, exit 0 default" {
  only AsymFacade.java
  run "$LSP" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"N=3"* ]]
}

@test "LSP-strict: AsymFacade with --strict -> FAIL exit 1" {
  only AsymFacade.java
  run "$LSP" --strict "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}

@test "LSP-parity: ParityFacade (N=2 Duration, M=2 Active) -> exit 0, no WARN" {
  only ParityFacade.java
  run "$LSP" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  [[ "$output" != *"FAIL"* ]]
}

@test "LSP-usage: no argument -> exit 3 (usage)" {
  run "$LSP"
  [ "$status" -eq 3 ]
}
