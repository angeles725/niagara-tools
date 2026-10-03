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

# polish-2026-10-02 P2e (#199, P2d review fail-open): Trip / Alarm / Fault also name cutout setpoints, so
# they are not status suffixes; a setpoint-shaped lpCutoutTrip / lowLimitAlarm / cutoutFault = 0 beside a
# nonzero floor still WARNs. Named mutation LCS-floor-trip (Trip|Alarm|Fault back in the status
# suffixes) -> LCS-floor-trip flips.
@test "LCS-floor-trip: comp2MinOn=1 beside lpCutoutTrip / lowLimitAlarm / cutoutFault =0 -> CS4 WARN" {
  local v
  for v in lpCutoutTrip lowLimitAlarm cutoutFault; do
    only FloorCamelNames.java
    sed -i "s/\"lpCutout\"/\"$v\"/" "$ONE/FloorCamelNames.java"
    run "$LCS" "$ONE"
    [ "$status" -eq 0 ]
    [[ "$output" == *"CS4: floor \"comp2MinOn\"=1 with LP cutout \"$v\"=0"* ]] || { echo "$v -> $output"; return 1; }
  done
}

@test "LCS-usage: no argument -> exit 3 (usage)" {
  run "$LCS"
  [ "$status" -eq 3 ]
}

# audit-2026-10-03 A1 — CS1 compares durations in ONE unit (ms). It used to compare the raw numbers of
# BRelTime.makeSeconds (s) and BRelTime.make (ms), and to skip makeMinutes/makeHours/makeDays and make(0L)
# silently. Factories: javax/baja/sys/BRelTime.java:56-83 (make(long ms), makeDays/Hours/Minutes/Seconds(int),
# make(d,h,m,s)); constants DEFAULT/SECOND/MINUTE/HOUR/DAY :34-48.
_cs1() { # _cs1 <Class> <interval-default> <duration-default>
  rm -f "$ONE"/*.java
  cat > "$ONE/$1.java" <<JAVA
package demo;
import javax.baja.sys.*;
@NiagaraType
public final class $1 extends BComponent {
  @NiagaraProperty(name = "defrostInterval", defaultValue = "$2")
  private BRelTime defrostInterval;
  @NiagaraProperty(name = "defrostDuration", defaultValue = "$3")
  private BRelTime defrostDuration;
}
JAVA
}

@test "CS1-units: makeHours(4) interval vs make(1800000L) duration is SANE (4h > 30m) — no raw-number compare (A1)" {
  # Mutation: CS1-units -- comparing raw factory arguments (4 <= 1800000) false-FAILs a sane pair.
  _cs1 U1 'BRelTime.makeHours(4)' 'BRelTime.make(1800000L)'
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  if [[ "$output" == *"CS1"* ]]; then return 1; fi
}

@test "CS1-mixed: make(30000) interval (30 s) vs makeSeconds(60) duration FAILs — the old raw compare passed it (A1)" {
  # Mutation: CS1-mixed -- reading make(ms) and makeSeconds(s) as the same unit compares 30000 <= 60 and passes.
  _cs1 U2 'BRelTime.make(30000)' 'BRelTime.makeSeconds(60)'
  run "$LCS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CS1"*"defrostInterval"* ]]
}

@test "CS1-minutes: makeMinutes(10) interval vs makeMinutes(45) duration FAILs — makeMinutes is parsed (A1)" {
  # Mutation: CS1-minutes -- skipping makeMinutes silently passes a 10 min cycle with a 45 min defrost.
  _cs1 U3 'BRelTime.makeMinutes(10)' 'BRelTime.makeMinutes(45)'
  run "$LCS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"CS1"* ]]
}

@test "CS1-zero: make(0L) interval FAILs against any positive duration (A1)" {
  _cs1 U4 'BRelTime.make(0L)' 'BRelTime.makeSeconds(60)'
  run "$LCS" "$ONE"
  [ "$status" -eq 1 ]
}

@test "CS1-unreadable: a BRelTime default the parser cannot read is a WARN row, never a silent skip (A1)" {
  # Mutation: CS1-unreadable -- dropping the unreadable row skips the pair silently again.
  _cs1 U5 'BRelTime.makeMinutes(DEFAULT_MIN)' 'BRelTime.makeMinutes(45)'
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"*"CS1"*"unreadable"*"defrostInterval"* ]]
}

@test "LCS-finderr: an unreadable sub-directory is an env error (exit 3), never a clean pass (A1)" {
  # Mutation: LCS-finderr -- ignoring the find status skips the locked directory's classes and exits 0.
  only IntervalBad.java
  mkdir -p "$ONE/locked"; mv "$ONE/IntervalBad.java" "$ONE/locked/"
  printf 'class A {}\n' > "$ONE/A.java"
  chmod 000 "$ONE/locked"
  if [ -r "$ONE/locked" ]; then chmod 755 "$ONE/locked"; skip "running as root"; fi
  run "$LCS" "$ONE"
  chmod 755 "$ONE/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}

@test "CS1-bothzero: BRelTime.DEFAULT interval AND duration (a display mirror) is unset, not a FAIL (A1)" {
  # Mutation: CS1-bothzero -- comparing a 0/0 (DEFAULT) pair FAILs every display-mirror panel.
  _cs1 U6 'BRelTime.DEFAULT' 'BRelTime.DEFAULT'
  run "$LCS" "$ONE"
  [ "$status" -eq 0 ]
  if [[ "$output" == *"CS1"* ]]; then return 1; fi
}

@test "CS1-wrapped: a factory inside a wrapper (make(BRelTime.class, BRelTime.makeSeconds(300))) is read, not skipped (A1)" {
  # Mutation: CS1-wrapped -- anchoring the factory to the whole defaultValue reads the wrapped form as unreadable.
  _cs1 U7 'make(BRelTime.class, BRelTime.makeSeconds(300))' 'make(BRelTime.class, BRelTime.makeMinutes(10))'
  run "$LCS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"CS1"*"5m"*"10m"* ]]
}
