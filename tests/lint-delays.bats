#!/usr/bin/env bats
# RED-FIRST pins for lint-delays.sh (campaign 8, from a PRODUCTION bug the kit did not catch).
# ColdRoomPan BDefrostController.armTrigger() schedules Clock.schedule(this, BRelTime.make(d), …) where
# d = Math.max(delayMs, 0L) — floored at 0L, not 1L. When the interval is overdue delayMs=0 -> d=0 ->
# EngineManager throws IllegalArgumentException "time <= 0", logError swallows it, the defrost interval
# dies silently (5x on the PANCCADIA station since 2026-09-02). lint-timers can't see it (it checks
# cancel-in-stopped + discarded tickets only).
#
# SURFACE (provisional): lint-delays.sh <src-dir>
#   FAIL when a Clock.schedule*( delay argument is a variable/expression with NO visible >0 floor.
#   PASS: Math.max(x, 1L) (or any literal >= 1), a positive literal, BRelTime.makeSeconds(<literal>).
#   FAIL: Math.max(x, 0L) (floor is 0, not >0) and a literal BRelTime.make(0).
#   exit 1 on any FAIL, 0 clean. Row: FAIL|PASS  delay  <file>: <detail>.
#
# RED today: lint-delays.sh does not exist -> every pin fails for the right reason (tool absent).
#
# NAMED MUTATION (post-green): accept Math.max(x, 0L) as a valid floor (>=0 instead of >0) -> LD1 and
# LD3 stop failing — the exact real-bug shape would pass, which is the whole point of the check.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LD="$KIT/toolbelt/lint-delays.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-delays"
  ONE="$BATS_TEST_TMPDIR/one"; mkdir -p "$ONE"
}
only() { rm -f "$ONE"/*.java; cp "$FX/$1" "$ONE/"; }   # isolate one fixture

@test "LD1: Math.max(delayMs, 0L) floor -> FAIL, names the class (the real BDefrostController shape)" {
  only Overdue.java
  run "$LD" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"Overdue"* ]]
}
@test "LD2: Math.max(delayMs, 1L) floor -> no FAIL (a >0 floor is safe)" {
  only Floored.java
  run "$LD" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}
@test "LD3: a literal BRelTime.make(0) -> FAIL" {
  only LiteralZero.java
  run "$LD" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}
@test "LD4: a positive literal BRelTime.makeSeconds(5) -> no FAIL" {
  only LiteralPos.java
  run "$LD" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}
@test "LD5: real smoke — the current ColdRoomPan-rt src FAILs naming BDefrostController (SKIP if not present)" {
  CRP="$HOME/modulos_niagara_n4/Cliente/Leon-Guanjuato/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"
  [ -d "$CRP" ] || skip "ColdRoomPan-rt src not on this machine (local-only real smoke)"
  run "$LD" "$CRP"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"BDefrostController"* ]]
}

# B801 [CERT]: Niagara rejects delay/period <= 0 (strictly positive) in Clock.schedule AND
# schedulePeriodically; the throw is runtime, so the lint must flag statically. Rows FAIL|WARN
# <file>:<line> <reason>; exit 1 any FAIL / 0 otherwise / 3 usage. [ev: corpus B801]

@test "LD6: schedulePeriodically period with no >0 floor (Math.max(p,0L)) -> FAIL" {
  only Periodic.java
  run "$LD" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}
@test "LD7: a slot getter delay whose property BFacets.MIN is 0 -> FAIL (the real interval case)" {
  only SlotGetterMinZero.java
  run "$LD" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}
@test "LD8: a slot getter delay whose property BFacets.MIN >= 1 -> WARN, not FAIL (floor is in facets)" {
  only SlotGetterMinPos.java
  run "$LD" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" != *"FAIL"* ]]
}
@test "LD9: no src-dir argument -> exit 3 (usage)" {
  run "$LD"
  [ "$status" -eq 3 ]
}
