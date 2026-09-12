#!/usr/bin/env bats
# RED-FIRST pins for lint-config-sanity.sh (Wave 3, LR3).
# Detects three unsafe @NiagaraProperty default patterns:
#   CS1 FAIL: *Interval default <= *Duration default (physical impossibility: interval must exceed duration)
#   CS2 FAIL: *Setpoint/*Set OPERATOR slot with default=0 and no min>0 facet (zero setpoint on cooling unit)
#   CS3 WARN: boolean flag combo enforced only in a comment (heuristic; see script header for limitation)
#
# Real commissioning defects (PANCCADIA): interval defaults smaller than duration defaults led to
# "interval already elapsed" immediately on first execute; setpoint=0 on a -20C freezer ran warm.
#
# RED today: lint-config-sanity.sh does not exist -> every pin fails for the right reason (tool absent).
#
# NAMED MUTATION (post-green): remove the interval<=duration comparison ->
# LCS-interval stops firing, CS1 shape silently passes.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LCS="$KIT/toolbelt/lint-config-sanity.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-config-sanity"
  ONE="$BATS_TEST_TMPDIR/one"; mkdir -p "$ONE"
}
only() { rm -f "$ONE"/*.java; cp "$FX/$1" "$ONE/"; }

@test "LCS-interval: IntervalBad (interval=30s <= duration=60s) -> CS1 FAIL exit 1" {
  only IntervalBad.java
  run "$LCS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"CS1"* ]]
}

@test "LCS-setpoint: SetpointZero (Setpoint default=0, OPERATOR, no min>0 facet) -> CS2 FAIL exit 1" {
  only SetpointZero.java
  run "$LCS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"CS2"* ]]
}

@test "LCS-flagcombo: FlagCombo (comment-only flag enforcement) -> CS3 WARN exit 0" {
  only FlagCombo.java
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"CS3"* ]]
}

@test "LCS-sane: SaneConfig (interval>duration, setpoint non-zero with min facet) -> exit 0, no FAIL" {
  only SaneConfig.java
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "LCS-usage: no argument -> exit 3 (usage)" {
  run "$LCS"
  [ "$status" -eq 3 ]
}
