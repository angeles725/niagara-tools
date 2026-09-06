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

@test "WP7: no write-path-matrix.md found anywhere -> exit 3 + ERROR line" {
  # Module root with OPERATOR slots but no docs/write-path-matrix.md anywhere up to git/fs root.
  # We create a fresh git repo with no docs/ at any level; the walk should error out.
  TMPREPO7="$(mktemp -d)"
  git init -q "$TMPREPO7"
  mkdir -p "$TMPREPO7/mod/src/com/x"
  printf '@NiagaraProperty(name="setpoint", flags=Flags.OPERATOR)\npublic class BT extends BComponent {}\n' \
    > "$TMPREPO7/mod/src/com/x/BT.java"
  # No docs/write-path-matrix.md anywhere in the repo.
  run "$LW" "$TMPREPO7/mod"
  rm -rf "$TMPREPO7"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]]
  [[ "$output" == *"no write-path-matrix.md"* ]]
}

@test "WP8: matrix two levels up in <repo-root>/docs/ is found; module-root run reports only truly uncovered slots" {
  # Simulates: <repo>/.git + <repo>/docs/write-path-matrix.md (shared matrix with setpoint row).
  # Module root: <repo>/Paccadia/ColdRoomPan with OPERATOR slots fanMode (not in matrix) + setpoint (in matrix).
  # BROKEN script (no walk-up): both slots FAIL (matrix empty).
  # FIXED script (walk-up): only fanMode FAIL; setpoint covered by shared matrix.
  TMPREPO8="$(mktemp -d)"
  git init -q "$TMPREPO8"
  mkdir -p "$TMPREPO8/docs"
  printf '# Write-Path Matrix\n| Slot | Writer | Timing | Test |\n|---|---|---|---|\n| setpoint | Dashboard | mid-cycle | w1 |\n' \
    > "$TMPREPO8/docs/write-path-matrix.md"
  mkdir -p "$TMPREPO8/Paccadia/ColdRoomPan/src/com/x"
  printf '@NiagaraProperty(name="fanMode", flags=Flags.OPERATOR)\n@NiagaraProperty(name="setpoint", flags=Flags.OPERATOR)\npublic class BCrp extends BComponent {}\n' \
    > "$TMPREPO8/Paccadia/ColdRoomPan/src/com/x/BCrp.java"
  MOD8="$TMPREPO8/Paccadia/ColdRoomPan"
  run "$LW" "$MOD8"
  rm -rf "$TMPREPO8"
  # fanMode uncovered -> FAIL; setpoint covered in shared matrix -> NOT flagged
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"fanMode"* ]]
  [[ "$output" != *"setpoint"* ]]
}
