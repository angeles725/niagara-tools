#!/usr/bin/env bats
# C11 T3 RED — concept-drift, the inverse of the S25 STALE advisory. S25 flags a matrix
# row whose backtick slot name is NOT in the covered set (and not [concept]-marked). T3
# flags the opposite drift: a [concept]-MARKED row whose slot name IS now in the covered
# set — the [concept] exemption has gone stale, because a real source slot with that name
# now exists, so the row should be a normal covered row, not a concept. STATUS-first DRIFT
# grammar, advisory by default (exit 0), promoted to exit 1 under --strict (like STALE).
#
# RED on dab0807: lint-write-path has STALE but no DRIFT class — a [concept] row is simply
# skipped (`case ... *[concept]* -> continue`), so no DRIFT is ever emitted. GREEN once T3
# adds the inverse pass. [ev: companero C11 T3]

setup() { LW="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/lint-write-path.sh"; }

_mk() {  # $1 dir ; source has a COVERED slot `setpoint` (OPERATOR); matrix from stdin
  local d="$1"; mkdir -p "$d/src/com/x" "$d/docs"
  cat > "$d/src/com/x/BThing.java" <<'JAVA'
package com.x;
@NiagaraProperty(name="setpoint", flags=Flags.SUMMARY | Flags.OPERATOR)
public class BThing extends BComponent {}
JAVA
  cat > "$d/docs/write-path-matrix.md"
}

@test "WP-drift-neg: a [concept] row whose slot IS covered emits DRIFT, exit stays 0 (advisory)" {
  d="$BATS_TEST_TMPDIR/driftneg"; _mk "$d" <<'MD'
# M
| Slot | Writer | Timing | Test |
|--|--|--|--|
| `setpoint` [concept] | D | mid | w1 |
MD
  run "$LW" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" == *"DRIFT"* && "$output" == *"setpoint"* ]]
}

@test "WP-drift-strict: --strict promotes the [concept]-covered DRIFT to exit 1" {
  d="$BATS_TEST_TMPDIR/driftstrict"; _mk "$d" <<'MD'
# M
| Slot | Writer | Timing | Test |
|--|--|--|--|
| `setpoint` [concept] | D | mid | w1 |
MD
  run "$LW" --strict "$d"
  [ "$status" -eq 1 ]
  [[ "$output" == *"DRIFT"* && "$output" == *"setpoint"* ]]
}

@test "WP-drift-true-concept: a [concept] row whose slot is NOT covered stays silent (a real concept — no DRIFT, no STALE)" {
  d="$BATS_TEST_TMPDIR/drifttrue"; _mk "$d" <<'MD'
# M
| Slot | Writer | Timing | Test |
|--|--|--|--|
| `hoaMode` [concept] | Engine link | n/a | — |
MD
  run "$LW" --strict "$d"
  [[ "$output" != *"DRIFT"* ]] && [[ "$output" != *"STALE"* ]]
}

@test "WP-drift-decoy: [concept] only inside a <!-- --> comment does NOT mark the row -> a covered slot stays silent (DRIFT keys on the comment-stripped [concept]; flips if the strip is dropped)" {
  d="$BATS_TEST_TMPDIR/driftdecoy"; _mk "$d" <<'MD'
# M
| Slot | Writer | Timing | Test |
|--|--|--|--|
| `setpoint` | D | mid <!-- [concept] --> | w1 |
MD
  run "$LW" --strict "$d"
  [[ "$output" != *"DRIFT"* ]]
}
