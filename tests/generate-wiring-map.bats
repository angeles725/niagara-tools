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

# ---------------------------------------------------------------------------
# Workbench display-name column (fold-2026-10-02-pending-retros WU6b)
# [ev: retro panccadia-commissioning-lessons Δ9]: a commissioning link table must carry the
# Workbench display name (from module.lexicon) next to the internal slot name and the full ord,
# or the operator cannot find the slots in the live tree.
# Named mutation GWM-display: drop the lexicon lookup -> every row shows the no-key placeholder.
# ---------------------------------------------------------------------------
@test "GWM-display: --lexicon fills the Workbench display name column (Type.slot beats bare key; no key -> placeholder)" {
  LX="$FX-lex/Facade-rt"
  run "$SCRIPT" "$LX/src" --lexicon "$LX/module.lexicon"
  [ "$status" -eq 0 ]
  [[ "$output" == *"| \`temperatureSetpoint\` | Setpoint de temperatura |"* ]]
  [[ "$output" == *"| \`roomTemperature\` | Temperatura del cuarto |"* ]]
  [[ "$output" == *"| \`doorOpen\` | _(no lexicon key)_ |"* ]]
}

@test "GWM-display-auto: module.lexicon next to <facade-src-dir> is discovered without --lexicon" {
  LX="$FX-lex/Facade-rt"
  run "$SCRIPT" "$LX/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"| \`temperatureSetpoint\` | Setpoint de temperatura |"* ]]
}

@test "GWM-columns: both tables carry internal name + Workbench display name + full ord columns" {
  run "$SCRIPT" "$FX"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '| Facade slot | Workbench display name | .* | Full ord |')" -eq 2 ]
}

@test "GWM-lexicon-missing: --lexicon <nonexistent> -> exit 3" {
  run "$SCRIPT" "$FX" --lexicon "/nonexistent/__gwm__.lexicon"
  [ "$status" -eq 3 ]
}
