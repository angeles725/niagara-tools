#!/usr/bin/env bats
# RED-FIRST pins for lint-link-target-flags.sh (fold-2026-10-02-pending-retros WU6b).
#   LTF1 FAIL: a link-in target slot (comment convention "link-in" / "Linked from" /
#              "Commissioning link" / "written by BLink", or a Table 2 row of --wiring-map)
#              that carries Flags.READONLY — Workbench LinkCheck refuses a READONLY target,
#              so the mirror silently never updates. [ev: retro panccadia-commissioning-lessons Δ1]
#   LTF2 FAIL: an OPERATOR *Mode / *Hoa slot that carries Flags.TRANSIENT — the operator
#              choice reverts to its default on restart and a facade->control link re-pushes it.
#              [ev: retro panccadia-persistent-config-hoa Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LTF="$KIT/toolbelt/lint-link-target-flags.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags"
}

@test "LTF-comment: READONLY slot whose comment names a Commissioning link / BLink -> LTF1 FAIL exit 1" {
  run "$LTF" "$FX/comment/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-link-target-flags"* ]]
  [[ "$output" == *"LTF1"* ]]
  [[ "$output" == *"evap1FreezeActive"* ]]
  [[ "$output" != *"comp1State"* ]]        # SUMMARY-only link-in slot is linkable -> not a subject
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL')" -eq 1 ]
}

@test "LTF-map: READONLY slot listed in --wiring-map Table 2 (control -> facade) -> LTF1 FAIL exit 1" {
  run "$LTF" --wiring-map "$FX/map/docs/wiring-map.md" "$FX/map/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"LTF1"* ]]
  [[ "$output" == *"evap2InDrip"* ]]
  [[ "$output" != *"temperatureSetpoint"* ]]   # Table 1 rows are facade SOURCES, never targets here
}

@test "LTF-map-neg: the SAME READONLY slot without --wiring-map and without a link comment is not a known target -> exit 0" {
  run "$LTF" "$FX/map/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "LTF-transient: OPERATOR *Mode / *Hoa slots with TRANSIENT -> LTF2 FAIL exit 1 (one row each)" {
  run "$LTF" "$FX/transient/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"LTF2"* ]]
  [[ "$output" == *"comp3Mode"* ]]
  [[ "$output" == *"fanHoa"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL')" -eq 2 ]
}

@test "LTF-clean: SUMMARY-only link-in, self-set READONLY anchor, persisted OPERATOR mode, TRANSIENT computed status mode -> exit 0; dot-dir pruned" {
  run "$LTF" "$FX/clean/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"oldFlag"* ]] && [[ "$output" != *"oldMode"* ]]
}

@test "LTF-usage: no argument, a non-directory, a missing --wiring-map file, or no Java sources -> exit 3" {
  run "$LTF"
  [ "$status" -eq 3 ]
  run "$LTF" "/nonexistent/__ltf__"
  [ "$status" -eq 3 ]
  run "$LTF" --wiring-map "/nonexistent/__map__.md" "$FX/map/src"
  [ "$status" -eq 3 ]
  E="$BATS_TEST_TMPDIR/empty"; mkdir -p "$E"
  run "$LTF" "$E"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]]
}
