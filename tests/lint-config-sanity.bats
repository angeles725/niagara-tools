#!/usr/bin/env bats
# RED-FIRST pins for lint-config-sanity.sh (Wave 3, LR3).
# Detects three unsafe @NiagaraProperty default patterns:
#   CS1 FAIL: *Interval default <= *Duration default (physical impossibility: interval must exceed duration)
#   CS2 FAIL: *Setpoint/*Set OPERATOR slot with default=0 and no min>0 facet (zero setpoint on cooling unit)
#   CS3 WARN: boolean flag combo enforced only in a comment (heuristic; see script header for limitation)
#   CS4 WARN: nonzero permanent-minimum floor default (*MinStagesOn*/*MinOn*, not a BRelTime) beside a
#             disabled (0) *LowLimit*/*Cutout* default in the same class [ev: retro panccadia-commissioning-lessons Δ3]
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

@test "LCS-floor: FloorNoCutout (minStagesOn default 1 + suctionLowLimit default 0) -> CS4 WARN exit 0" {
  only FloorNoCutout.java
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"CS4"* ]]
  [[ "$output" == *"minStagesOn"* ]] && [[ "$output" == *"suctionLowLimit"* ]]
}

@test "LCS-floor-neg: an enabled cutout (FloorWithCutout) or a BRelTime minOn timer (TimerNotFloor) -> no CS4" {
  only FloorWithCutout.java
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"CS4"* ]]
  only TimerNotFloor.java
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"CS4"* ]]
}

# polish-2026-10-02 P2b (#199 WU6b): CS4 matches names on camelCase word boundaries, not substrings. A
# floor is ...MinOn / ...MinStagesOn (not a *Time/*Delay/*Sec timer); an LP cutout floor ends in LowLimit
# or Cutout. Named mutation LCS-floor-name (back to the substring match) -> LCS-floor-name flips.
@test "LCS-floor-name: substring decoys (adminOnline, minOnTime, cutoutDelay) -> no CS4" {
  only FloorNameDecoys.java
  cp "$FX/FloorNoCutout.java" "$ONE/"; sed -i -e 's/FloorNoCutout/FloorDecoyPair/' -e 's/"minStagesOn"/"stageCount"/' "$ONE/FloorNoCutout.java"
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"CS4"* ]]
  # a real floor beside a cutoutDelay of 0 (a delay, not a disabled cutout) -> still no CS4
  only FloorCamelNames.java
  sed -i 's/"lpCutout"/"cutoutDelay"/' "$ONE/FloorCamelNames.java"
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"CS4"* ]]
}

@test "LCS-floor-camel: comp2MinOn=1 beside lpCutout=0 -> CS4 WARN naming both" {
  only FloorCamelNames.java
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"CS4: floor \"comp2MinOn\"=1 with LP cutout \"lpCutout\"=0"* ]]
}

# polish-2026-10-02 P2c (#199, P2b review advisories): a cutout name may carry a unit or qualifier
# suffix (lpCutoutPsi, lowLimitBar); only timer suffixes (Delay/Time/Sec...) are excluded.
# Named mutation LCS-floor-suffix (anchor the cutout token to the name end again) -> LCS-floor-suffix flips.
@test "LCS-floor-suffix: comp2MinOn=1 beside lpCutoutPsi=0 or lowLimitBar=0 -> CS4 WARN" {
  local v
  for v in lpCutoutPsi lowLimitBar; do
    only FloorCamelNames.java
    sed -i "s/\"lpCutout\"/\"$v\"/" "$ONE/FloorCamelNames.java"
    run "$LCS" "$ONE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"CS4: floor \"comp2MinOn\"=1 with LP cutout \"$v\"=0"* ]] || { echo "$v -> $output"; return 1; }
  done
}

# polish-2026-10-02 P2d (#199, P2c review advisories): a cutout status or counter slot (cutoutCount,
# cutoutActive, lowLimitReached, cutoutTripCount) is not a cutout floor, and a non-numeric default
# (false, BBoolean.FALSE) is never read as a disabled cutout. Unit suffixes (lpCutoutPsi) still count.
# Named mutation LCS-floor-status (drop the status/counter suffix exclusion) -> LCS-floor-status flips.
@test "LCS-floor-status: comp2MinOn=1 beside cutoutCount / cutoutActive / lowLimitReached =0 -> no CS4" {
  local v
  for v in cutoutCount cutoutActive lowLimitReached cutoutTripCount lpCutoutState; do
    only FloorCamelNames.java
    sed -i "s/\"lpCutout\"/\"$v\"/" "$ONE/FloorCamelNames.java"
    run "$LCS" "$ONE"
    [ "$status" -eq 0 ]
    [[ "$output" != *"CS4"* ]] || { echo "$v -> $output"; return 1; }
  done
  for v in 'false' 'BBoolean.FALSE'; do
    only FloorCamelNames.java
    sed -i -e 's/"lpCutout", type = "double", defaultValue = "0.0"/"lpCutout", type = "boolean", defaultValue = "'"$v"'"/' "$ONE/FloorCamelNames.java"
    grep -q "defaultValue = \"$v\"" "$ONE/FloorCamelNames.java"
    run "$LCS" "$ONE"
    [ "$status" -eq 0 ]
    [[ "$output" != *"CS4"* ]] || { echo "default $v -> $output"; return 1; }
  done
  # the suffixed real cutout is still a cutout floor
  only FloorCamelNames.java
  sed -i 's/"lpCutout"/"lpCutoutPsi"/' "$ONE/FloorCamelNames.java"
  run "$LCS" "$ONE"
  [[ "$output" == *"CS4: floor \"comp2MinOn\"=1 with LP cutout \"lpCutoutPsi\"=0"* ]]
}

@test "LCS-usage: no argument -> exit 3 (usage)" {
  run "$LCS"
  [ "$status" -eq 3 ]
}
