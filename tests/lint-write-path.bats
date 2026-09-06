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
  # Pure mktemp root: no .git above, no docs/write-path-matrix.md (with table rows) anywhere.
  # Skip if any ancestor already has a valid matrix (would invalidate the premise on this host).
  TMPMOD7="$(mktemp -d)"
  _check="$TMPMOD7"
  _found_valid_matrix=0
  while true; do
    if [ -f "$_check/docs/write-path-matrix.md" ] && grep -q '^\|' "$_check/docs/write-path-matrix.md" 2>/dev/null; then
      _found_valid_matrix=1; break
    fi
    _up="$(dirname "$_check")"
    [ "$_up" = "$_check" ] && break
    _check="$_up"
  done
  if [ "$_found_valid_matrix" -eq 1 ]; then
    rm -rf "$TMPMOD7"
    skip "ancestor path already has a valid docs/write-path-matrix.md — WP7 premise not testable on this host"
  fi
  mkdir -p "$TMPMOD7/src/com/x"
  printf '@NiagaraProperty(name="setpoint", flags=Flags.OPERATOR)\npublic class BT extends BComponent {}\n' \
    > "$TMPMOD7/src/com/x/BT.java"
  run "$LW" "$TMPMOD7"
  rm -rf "$TMPMOD7"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]]
  [[ "$output" == *"no write-path-matrix.md"* ]]
}

@test "WP8: matrix two levels up in <repo-root>/docs/ is found; module-root run reports only truly uncovered slots" {
  # Simulates: <repo>/.git + <repo>/docs/write-path-matrix.md (shared matrix with setpoint row).
  # Module root: <repo>/Paccadia/ColdRoomPan/src with OPERATOR slots fanMode (not in matrix) + setpoint (in matrix).
  # Walk-up finds the shared matrix; only fanMode should FAIL.
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

@test "WP9a: module root with two profile dirs (both have OPERATOR slots) -> rows for both profiles required" {
  # Simulates Paccadia/ColdRoomPan with ColdRoomPan-rt/ and ColdRoomPan-wb/ inside.
  # The shared matrix covers setpoint (in -rt). fanMode (-rt) and wbParam (-wb) are uncovered.
  TMPREPO9="$(mktemp -d)"
  git init -q "$TMPREPO9"
  mkdir -p "$TMPREPO9/docs"
  printf '# Write-Path Matrix\n| Slot | Writer | Timing | Test |\n|---|---|---|---|\n| setpoint | Dashboard | mid-cycle | w1 |\n' \
    > "$TMPREPO9/docs/write-path-matrix.md"
  # Profile 1: ColdRoomPan-rt
  mkdir -p "$TMPREPO9/ColdRoomPan/ColdRoomPan-rt/src/com/x"
  printf '@NiagaraProperty(name="setpoint", flags=Flags.OPERATOR)\n@NiagaraProperty(name="fanMode", flags=Flags.OPERATOR)\npublic class BCrp extends BComponent {}\n' \
    > "$TMPREPO9/ColdRoomPan/ColdRoomPan-rt/src/com/x/BCrp.java"
  # Profile 2: ColdRoomPan-wb (writeback — also has an OPERATOR slot)
  mkdir -p "$TMPREPO9/ColdRoomPan/ColdRoomPan-wb/src/com/x"
  printf '@NiagaraProperty(name="wbParam", flags=Flags.OPERATOR)\npublic class BWb extends BComponent {}\n' \
    > "$TMPREPO9/ColdRoomPan/ColdRoomPan-wb/src/com/x/BWb.java"
  MOD9="$TMPREPO9/ColdRoomPan"
  run "$LW" "$MOD9"
  rm -rf "$TMPREPO9"
  # fanMode (-rt) and wbParam (-wb) uncovered; setpoint (-rt) covered
  [ "$status" -eq 1 ]
  [[ "$output" == *"ColdRoomPan-rt"* ]]
  [[ "$output" == *"fanMode"* ]]
  [[ "$output" == *"ColdRoomPan-wb"* ]]
  [[ "$output" == *"wbParam"* ]]
  [[ "$output" != *"setpoint"* ]]
}

@test "WP9b: module root with no src anywhere -> ERROR exit 3" {
  # Module root has profile-like subdirs but none has a src/; ERROR + exit 3.
  TMPREPO9b="$(mktemp -d)"
  git init -q "$TMPREPO9b"
  mkdir -p "$TMPREPO9b/docs"
  printf '# Write-Path Matrix\n| Slot | Writer | Timing | Test |\n|---|---|---|---|\n| setpoint | Dashboard | mid-cycle | w1 |\n' \
    > "$TMPREPO9b/docs/write-path-matrix.md"
  mkdir -p "$TMPREPO9b/ColdRoomPan/ColdRoomPan-rt"   # no src/
  MOD9b="$TMPREPO9b/ColdRoomPan"
  run "$LW" "$MOD9b"
  rm -rf "$TMPREPO9b"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]]
  [[ "$output" == *"no src found"* ]]
}
