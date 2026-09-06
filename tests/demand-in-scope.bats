#!/usr/bin/env bats
# RED-FIRST pins for demand-in-scope.sh (campaign 9 S7, from B820 16b635f0f; concretizes B819 §819.5).
# A SOURCE lint (like lint-delays.sh), WARN-only (--strict -> FAIL). Within a control-decision method
# (one that writes an output/cmd/target/setBool driven by a numeric input), scan the method's PARAMETERS
# or the enclosing class's FIELDS for a demand-shaped input — a name/type in
# {demand*, *call*, enable, *count, loopEnable, an in:BStatusBoolean}. If the method reads a PROCESS
# VARIABLE (suction/pressure/temperature/cv) but has ZERO demand-shaped input in scope -> WARN
# ("pressure without demand": the process can run to defend the variable with nobody calling — the
# "why can't it turn off" failure mode). §820.3: WARN only (statically-decidable absence), never a hard
# FAIL; whether a PRESENT input is really "demand" vs a "modulator" is a review judgment (B819-G1).
#
# Real PASS shape: CompressorControl.step(long now, int demandCount, ... double suction ...) with
#   `if (demandCount <= 0) target = 0;` — demand gates, pressure only modulates. [ev: corpus B820 §820.4]
#
# SURFACE: lint-demand-scope.sh [--strict] <java-src-dir>
#   Row:  WARN  demand-in-scope  <file>:<line>  <method> reads <pv> with no demand-shaped input in scope
#   Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
#
# RED today: lint-demand-scope.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (§820.4, DS2): remove the demandCount param + the `if (demandCount<=0) target=0` gate,
#   leaving staging on suction alone -> the method WARNs ("pressure without demand").

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  DIS="$KIT/toolbelt/lint-demand-scope.sh"
  S="$BATS_TEST_TMPDIR/src"
  ROOT="${C9_CLIENT_ROOT:-/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-ff1b659}"   # RP1: blessed read tree, never the local working copy
  REAL="$ROOT/Compresores/CompPan/CompPan-rt/src/com/angeles/CompPan/CompressorControl.java"
}
only() { rm -rf "$S"; mkdir -p "$S/com/x"; cat > "$S/com/x/$1"; }

# --- DS1: demand in the parameter list + a gate on it -> clean ---
@test "DS1: pass_demandCount_param_gates (demand in scope, reads suction -> no WARN)" {
  only CompressorControl.java <<'JAVA'
package com.x;
public class CompressorControl {
  int step(long now, int demandCount, double suction, double suctionLowLimit) {
    int target = demandCount;
    if (demandCount <= 0) target = 0;                       // demand GATE
    if (suction < suctionLowLimit) target = Math.min(target, demandCount - 1);
    return target;
  }
}
JAVA
  run "$DIS" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- DS2: NAMED MUTATION — demand removed, staging on suction alone -> WARN ---
@test "DS2: fail_demand_removed_mutant (pressure without demand -> WARN naming the method)" {
  only CompressorControl.java <<'JAVA'
package com.x;
public class CompressorControl {
  int step(long now, double suction, double suctionLowLimit) {
    int target = 4;
    if (suction < suctionLowLimit) target = target - 1;     // pressure decides, nobody calls
    return target;
  }
}
JAVA
  run "$DIS" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"demand-in-scope"* ]] && [[ "$output" == *"step"* ]]
}

# --- DS3: no demand param, but a demand-shaped CLASS FIELD in scope -> clean ---
@test "DS3: pass_demand_field_on_class (scope includes enclosing fields -> no WARN)" {
  only CompressorControl.java <<'JAVA'
package com.x;
public class CompressorControl {
  private int demandCount = 0;                              // demand-shaped field in scope
  int step(long now, double suction, double suctionLowLimit) {
    int target = demandCount;
    if (suction < suctionLowLimit) target = Math.min(target, demandCount - 1);
    return target;
  }
}
JAVA
  run "$DIS" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- DS4: writes an output but reads NO process variable -> no WARN (rule does not key on it) ---
@test "DS4: no_flag_no_process_variable (guards against over-triggering)" {
  only ModeSelect.java <<'JAVA'
package com.x;
public class ModeSelect {
  int pick(long now, int mode) {
    int target = 0;
    if (mode == 1) target = 2;                              // no suction/pressure/temp read
    return target;
  }
}
JAVA
  run "$DIS" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- DS5: --strict promotes the WARN to exit 1 ---
@test "DS5: strict_flag_promotes_warn_to_fail" {
  only CompressorControl.java <<'JAVA'
package com.x;
public class CompressorControl {
  int step(long now, double suction, double lowLimit) {
    int target = 4;
    if (suction < lowLimit) target = target - 1;
    return target;
  }
}
JAVA
  run "$DIS" "$S"
  [ "$status" -eq 0 ]
  run "$DIS" --strict "$S"
  [ "$status" -eq 1 ]
}

# --- DS6: no <src-dir> argument -> exit 3 ---
@test "DS6: no src-dir argument -> exit 3" {
  run "$DIS"
  [ "$status" -eq 3 ]
}

# --- DS7: a demand-less method under .deploy-baseline/ is NOT flagged (D9b dot-dir prune) ---
@test "DS7: dot-dir prune (D9b) — a pressure-without-demand method under .deploy-baseline/ not counted" {
  rm -rf "$S"; mkdir -p "$S/com/x" "$S/.deploy-baseline/com/x"
  cat > "$S/com/x/Clean.java" <<'JAVA'
package com.x;
public class Clean {
  int step(long now, int demandCount, double suction) { return demandCount <= 0 ? 0 : demandCount; }
}
JAVA
  cat > "$S/.deploy-baseline/com/x/Stale.java" <<'JAVA'
package com.x;
public class Stale {
  int step(long now, double suction, double lowLimit) { int t=4; if (suction<lowLimit) t--; return t; }
}
JAVA
  run "$DIS" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]] && [[ "$output" != *"Stale"* ]]
}

# --- DS-smoke: the REAL CompressorControl.step (demandCount gate) is clean (SKIP if absent) ---
@test "DS-smoke: real CompressorControl.step has demand in scope -> no WARN for step" {
  [ -f "$REAL" ] || skip "client CompPan tree not on this machine"
  D="$BATS_TEST_TMPDIR/real"; mkdir -p "$D"; cp "$REAL" "$D/"
  run "$DIS" "$D"
  [ "$status" -eq 0 ]                                  # the tool ran (RED now: absent -> 127)
  [ "$(printf '%s\n' "$output" | grep -c 'WARN.*step')" -eq 0 ]   # real step: demand in scope, not flagged
}

# --- DS9: a source dir with NO Java files -> exit 3 + ERROR row, never a silent 0 (K20 / C8 silent-0 lesson; WP9b shape) ---
@test "DS9: no-sources-exit-3 (empty dir and a dir with only a non-Java file -> exit 3 + ERROR row, no WARN)" {
  E="$BATS_TEST_TMPDIR/empty"; rm -rf "$E"; mkdir -p "$E"
  run "$DIS" "$E"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]] && [[ "$output" == *"demand-in-scope"* ]] && [[ "$output" != *"WARN"* ]]
  N="$BATS_TEST_TMPDIR/nojava"; rm -rf "$N"; mkdir -p "$N/com/x"; printf 'not java\n' > "$N/com/x/README.txt"
  run "$DIS" "$N"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]] && [[ "$output" != *"WARN"* ]]
}
