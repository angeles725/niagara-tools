#!/usr/bin/env bats
# Pins for commissioning-verify.sh (issue #114, §6.b commissioning punch-list auditor).
# Orchestrates lint-config-sanity.sh, lint-status-parity.sh, lint-recovery-path.sh (per -rt/src),
# and bog-audit.sh CHECK11/CHECK13-19/CHECK20 (when --bog is given) into a single §6.b punch-list.
#
# Row format: STATUS  commissioning  <check>  <detail>
# Exit: 0 clean (zero FAIL) · 1 any FAIL · 3 usage/env.
#
# RED today: commissioning-verify.sh does not exist -> every pin fails for the right reason.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  CV="$KIT/toolbelt/commissioning-verify.sh"
  MINIMAL="$KIT/fixtures/MinimalPan"
  BOG_DIR="$BATS_TEST_DIRNAME/fixtures/bog"
  CV_FX="$BATS_TEST_DIRNAME/fixtures/commissioning"
}

@test "CV-usage: no argument -> exit 3 (usage)" {
  run "$CV"
  [ "$status" -eq 3 ]
}

@test "CV-clean: MinimalPan with no --bog -> exit 0, emits config-sanity/parity/recovery rows + SKIP-bog note + MANUAL footer" {
  run "$CV" "$MINIMAL"
  [ "$status" -eq 0 ]
  # Source check rows present
  [[ "$output" == *"config-sanity"* ]]
  [[ "$output" == *"status-parity"* ]]
  [[ "$output" == *"recovery-path"* ]]
  # Bog SKIP rows present (no --bog given)
  [[ "$output" == *"SKIP"*"proxy-link-safety"* ]]
  [[ "$output" == *"SKIP"*"station-logic"* ]]
  # Manual footer rows present
  [[ "$output" == *"MANUAL"*"hot-reload-console"* ]]
  [[ "$output" == *"MANUAL"*"plant-control"* ]]
  [[ "$output" == *"MANUAL"*"per-instance-values"* ]]
  # Summary present
  [[ "$output" == *"commissioning-verify:"* ]]
  [[ "$output" == *"CLEAN"* ]]
  # No FAIL status rows (summary may contain "0 FAIL" — match the row prefix instead)
  [[ "$output" != *"FAIL  commissioning"* ]]
}

@test "CV-bog-clean: station-logic-CHECK13-clean.bog + --module ColdRoomPan -> exit 0, no FAIL row" {
  run "$CV" "$MINIMAL" --bog "$BOG_DIR/station-logic-CHECK13-clean.bog" --module ColdRoomPan
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL  commissioning"* ]]
  [[ "$output" == *"commissioning-verify:"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "CV-bog-fail: station-logic-CHECK13.bog (CHECK11+CHECK13 FAIL) + --module ColdRoomPan -> exit 1, FAIL row" {
  run "$CV" "$MINIMAL" --bog "$BOG_DIR/station-logic-CHECK13.bog" --module ColdRoomPan
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"commissioning"* ]]
  [[ "$output" == *"ISSUES"* ]]
}

@test "CV-configfail: src fixture with CS1 violation (interval<=duration) -> exit 1, config-sanity FAIL row" {
  run "$CV" "$CV_FX"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"commissioning"*"config-sanity"* ]]
  [[ "$output" == *"CS1"* ]]
  [[ "$output" == *"ISSUES"* ]]
}
