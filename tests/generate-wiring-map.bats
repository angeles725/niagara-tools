#!/usr/bin/env bats
# Tests for generate-wiring-map.sh (Δ6, retro live-commissioning-verification-gaps).
#
# Verifies that the script:
#   - Exits 3 on missing argument or non-directory.
#   - Emits Table 1 (OPERATOR config → control) with the expected slot.
#   - Emits Table 2 (SUMMARY display ← control) with the expected slot.
#   - Emits the per-instance commissioning checklist.
#   - Writes to --output <file> when requested.
#   - Exits 0 on an empty source directory (no slots found, scaffold still emitted).

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SCRIPT="$KIT/toolbelt/generate-wiring-map.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/generate-wiring-map"
}

# ---------------------------------------------------------------------------
# Usage / env errors
# ---------------------------------------------------------------------------

@test "GWM-usage: no argument -> exit 3" {
  run "$SCRIPT"
  [ "$status" -eq 3 ]
}

@test "GWM-notdir: non-existent path -> exit 3" {
  run "$SCRIPT" "/nonexistent/__gwm_xyz__"
  [ "$status" -eq 3 ]
}

@test "GWM-unknown-opt: unknown option -> exit 3" {
  run "$SCRIPT" "$FX" --unknown-flag
  [ "$status" -eq 3 ]
}

# ---------------------------------------------------------------------------
# Slot extraction
# ---------------------------------------------------------------------------

@test "GWM-operator: BRoomFacade.java OPERATOR slot -> Table 1 contains temperatureSetpoint" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Table 1"* ]]
  [[ "$output" == *"temperatureSetpoint"* ]]
}

@test "GWM-summary: BRoomFacade.java SUMMARY slot -> Table 2 contains roomTemperature" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Table 2"* ]]
  [[ "$output" == *"roomTemperature"* ]]
}

@test "GWM-operator-not-in-table2: temperatureSetpoint (OPERATOR) must NOT appear under Table 2 header" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  # Table 2 section starts after Table 1; temperatureSetpoint is OPERATOR only -> Table 1 only.
  # Split at Table 2 and confirm temperatureSetpoint is absent from that half.
  table2="${output#*Table 2}"
  [[ "$table2" != *"temperatureSetpoint"* ]]
}

# ---------------------------------------------------------------------------
# Commissioning checklist
# ---------------------------------------------------------------------------

@test "GWM-checklist: output contains commissioning checklist heading" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  [[ "$output" == *"commissioning checklist"* ]]
}

@test "GWM-checklist-items: output contains key checklist items" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  [[ "$output" == *"triage-console"* ]]
  [[ "$output" == *"bog-audit"* ]]
  [[ "$output" == *"obix-nav"* ]]
}

# ---------------------------------------------------------------------------
# --output file
# ---------------------------------------------------------------------------

@test "GWM-output-file: --output writes file and exits 0" {
  local tmpout="$BATS_TEST_TMPDIR/wiring-map.md"
  run "$SCRIPT" "$FX" --output "$tmpout"
  [ "$status" -eq 0 ]
  [ -f "$tmpout" ]
  grep -q "temperatureSetpoint" "$tmpout"
}

@test "GWM-output-stdout: without --output, content goes to stdout" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Wiring Map"* ]]
}

# ---------------------------------------------------------------------------
# Empty source directory (no Java files)
# ---------------------------------------------------------------------------

@test "GWM-empty-src: empty directory -> exits 0, scaffold emitted with no-slots placeholder" {
  local emptydir="$BATS_TEST_TMPDIR/empty_gwm"
  mkdir -p "$emptydir"
  run "$SCRIPT" "$emptydir"
  [ "$status" -eq 0 ]
  [[ "$output" == *"no OPERATOR slots found"* ]]
}
