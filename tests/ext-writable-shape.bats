#!/usr/bin/env bats
# RED-FIRST pins for ext-writable-shape.sh (campaign 9 S19, from the B825/B826 slot-type doctrine
# 546271613). A SOURCE lint (like lint-delays.sh): an OPERATOR *complex* property
# (BStatusNumeric/BStatusBoolean/BStatusEnum) with NO @NiagaraAction on the same type is a bad
# external-write target — a bare complex either rejects the PUT ("Cannot translate") or, via the
# wrapped-<obj> shorthand, silently writes a DEFAULT (the live silent-zero: a setpoint set to 0.0 on
# a 200 OK). It must be written through its child-leaf ORD (bare <real>) or carry a writing action.
#   -> WARN (child-leaf note). Plain double/bool/reltime OPERATOR slots are clean; a complex OPERATOR
#   slot WITH an @NiagaraAction that sets it is clean; a SUMMARY-only complex slot is exempt.
#
# Real trigger: DashboardPan BRoomPanel.setpoint (BStatusNumeric, SUMMARY|OPERATOR, NO action) at
# a109249 — the exact 3421b34(double)->3166b8d(StatusNumeric) regression axis. [ev: corpus B825/B826]
#
# SURFACE: ext-writable-shape.sh [--strict] <java-src-dir>
#   Row:  WARN  ext-writable-shape  <file>:<line>  <slot>: OPERATOR <type> with no writing action ...
#   Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
#
# RED today: lint-ext-writable-shape.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (post-green): treat a complex OPERATOR slot as plain (drop the BStatusX recognizer)
#   -> EW1/EW6 stop WARNing (the setpoint_statusnumeric_no_action regression goes unseen).

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  EW="$KIT/toolbelt/lint-ext-writable-shape.sh"
  SRC="$BATS_TEST_TMPDIR/src"; mkdir -p "$SRC/com/x"
  ROOT="${C9_CLIENT_ROOT:-/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-a109249}"   # RP1: blessed read tree, never the local working copy
}
only() { rm -rf "$SRC"; mkdir -p "$SRC/com/x"; cat > "$SRC/com/x/$1"; }

# --- EW1: the real regression shape — StatusNumeric OPERATOR, no action -> WARN naming the slot ---
@test "EW1: warns_on_operator_statusnumeric_without_action (child-leaf note, names setpoint)" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="setpoint", type="BStatusNumeric", defaultValue="new BStatusNumeric(0d)", flags=Flags.SUMMARY|Flags.OPERATOR)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"ext-writable-shape"* ]] && [[ "$output" == *"setpoint"* ]]
}

# --- EW2: plain double OPERATOR -> clean ---
@test "EW2: clean_on_plain_double_operator (no WARN)" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="differentialUp", type="double", defaultValue="1.5", flags=Flags.SUMMARY|Flags.OPERATOR)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- EW3: complex OPERATOR WITH a writing action -> clean (action is the safe write surface) ---
@test "EW3: clean_on_operator_statusnumeric_with_matching_action (no WARN)" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="setpoint", type="BStatusNumeric", defaultValue="new BStatusNumeric(0d)", flags=Flags.SUMMARY|Flags.OPERATOR)
@NiagaraAction(name="setSetpoint", parameterType="BDouble", defaultValue="BDouble.make(0)", flags=Flags.OPERATOR)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- EW4: SUMMARY-only complex (no OPERATOR) -> exempt, clean ---
@test "EW4: ignores_summary_only_statusnumeric (display slot, not a write target)" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="zoneTemp1", type="BStatusNumeric", defaultValue="new BStatusNumeric()", flags=Flags.SUMMARY)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- EW5: BRelTime + boolean OPERATOR (plain) -> clean ---
@test "EW5: clean_on_reltime_and_boolean_operator (plain types)" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="defrostInterval", type="BRelTime", defaultValue="BRelTime.make(1800000)", flags=Flags.SUMMARY|Flags.OPERATOR)
@NiagaraProperty(name="coolOnSensorFault", type="boolean", defaultValue="false", flags=Flags.SUMMARY|Flags.OPERATOR)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# --- EW6: StatusBoolean OPERATOR, no action -> WARN (complex class beyond StatusNumeric) ---
@test "EW6: warns_on_operator_statusboolean_without_action" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="manualEnable", type="BStatusBoolean", defaultValue="new BStatusBoolean()", flags=Flags.SUMMARY|Flags.OPERATOR)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"manualEnable"* ]]
}

# --- EW7: WARN-only never fails the build; --strict promotes to exit 1 ---
@test "EW7: warn_only_never_fails_build; --strict -> exit 1" {
  only BRoom.java <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="setpoint", type="BStatusNumeric", defaultValue="new BStatusNumeric(0d)", flags=Flags.SUMMARY|Flags.OPERATOR)
public class BRoom extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  run "$EW" --strict "$SRC"
  [ "$status" -eq 1 ]
}

# --- EW8: a WARN-shaped class under .deploy-baseline/ is NOT flagged (D9b dot-dir prune) ---
@test "EW8: dot-dir prune — a setpoint under src/.deploy-baseline/ is not counted (D9b)" {
  rm -rf "$SRC"; mkdir -p "$SRC/com/x" "$SRC/.deploy-baseline/com/x"
  cat > "$SRC/com/x/Clean.java" <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="differentialUp", type="double", defaultValue="1.5", flags=Flags.SUMMARY|Flags.OPERATOR)
public class Clean extends BComponent {}
JAVA
  cat > "$SRC/.deploy-baseline/com/x/Stale.java" <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="setpoint", type="BStatusNumeric", defaultValue="new BStatusNumeric(0d)", flags=Flags.SUMMARY|Flags.OPERATOR)
public class Stale extends BComponent {}
JAVA
  run "$EW" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]] && [[ "$output" != *"Stale"* ]]
}

# --- EW9: no <src-dir> argument -> exit 3 (usage) ---
@test "EW9: no src-dir argument -> exit 3" {
  run "$EW"
  [ "$status" -eq 3 ]
}

# --- EW10: EXACT contract at a109249 (C9_CLIENT_ROOT): the only complex OPERATOR slot without an action is BRoomPanel.setpoint ---
@test "EW10: real trees exact — DashboardPan-rt exactly 1 WARN (BRoomPanel.setpoint), CompPan-rt 1 (faultReset — per-slot S22: no action writes it), ColdRoomPan-rt 0, DashboardPan-ux 0" {
  [ -d "$ROOT/Dashboard" ] && [ -d "$ROOT/Compresores" ] && [ -d "$ROOT/Paccadia" ] || skip "client read tree not on this machine (set C9_CLIENT_ROOT)"
  run "$EW" "$ROOT/Dashboard/DashboardPan/DashboardPan-rt/src"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
  [[ "$output" == *"BRoomPanel"* ]] && [[ "$output" == *"setpoint"* ]]
  # S22 (C10): per-slot exemption. CompPan-rt is 1 — faultReset (BStatusBoolean SUMMARY|OPERATOR) has NO
  # @NiagaraAction that writes it (the class only has HIDDEN tick/powerOnExpired + ackAlarm, none writing
  # faultReset). ColdRoomPan-rt/DashboardPan-ux stay 0.
  run "$EW" "$ROOT/Compresores/CompPan/CompPan-rt/src"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
  [[ "$output" == *"faultReset"* ]]
  for r in Paccadia/ColdRoomPan/ColdRoomPan-rt Dashboard/DashboardPan/DashboardPan-ux; do
    run "$EW" "$ROOT/$r/src"
    [ "$status" -eq 0 ]
    [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
  done
}

# --- EW11: a source dir with NO Java files -> exit 3 + ERROR row, never a silent 0 (K20 / C8 silent-0 lesson; WP9b shape) ---
@test "EW11: no-sources-exit-3 (empty dir and a dir with only a non-Java file -> exit 3 + ERROR row, no WARN)" {
  E="$BATS_TEST_TMPDIR/empty"; rm -rf "$E"; mkdir -p "$E"
  run "$EW" "$E"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]] && [[ "$output" == *"ext-writable-shape"* ]] && [[ "$output" != *"WARN"* ]]
  N="$BATS_TEST_TMPDIR/nojava"; rm -rf "$N"; mkdir -p "$N/com/x"; printf 'not java\n' > "$N/com/x/README.txt"
  run "$EW" "$N"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]] && [[ "$output" != *"WARN"* ]]
}

# ===========================================================================
# S22 (C10 lint-precision RED): the @NiagaraAction exemption must be PER-SLOT — a
# complex OPERATOR slot is exempt only when SOME @NiagaraAction body WRITES THAT
# slot (setX / X.set), not merely because the class exposes ANY action. Real gap
# on df8c7ec: BCompressorControl.faultReset (BStatusBoolean SUMMARY|OPERATOR) is
# exempted only because the class has HIDDEN tick/powerOnExpired/ackAlarm actions,
# none of which writes faultReset. RED-for-the-right-reason: EW-s22-neg + the
# CompPan-rt-1 smoke FAIL on df8c7ec (the class-level exemption suppresses them).
# ---------------------------------------------------------------------------
@test "EW-s22-pos: a complex OPERATOR slot whose value an @NiagaraAction body WRITES is exempt (no WARN)" {
  D="$BATS_TEST_TMPDIR/s22pos"; mkdir -p "$D/com/x"
  cat > "$D/com/x/BWithWriter.java" <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="setpoint", type="BStatusNumeric", flags=Flags.SUMMARY|Flags.OPERATOR)
@NiagaraAction(name="bumpSetpoint")
public class BWithWriter extends BComponent {
  public void doBumpSetpoint() { setSetpoint(new BStatusNumeric(getSetpoint().getValue()+1)); } // writes THAT slot
}
JAVA
  run "$EW" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "EW-s22-neg: a complex OPERATOR slot with an @NiagaraAction that does NOT write it WARNs (per-slot, not class-level)" {
  D="$BATS_TEST_TMPDIR/s22neg"; mkdir -p "$D/com/x"
  cat > "$D/com/x/BUnrelatedAction.java" <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="faultReset", type="BStatusBoolean", flags=Flags.SUMMARY|Flags.OPERATOR)
@NiagaraAction(name="tick", flags=Flags.HIDDEN)
@NiagaraAction(name="ackAlarm")
public class BUnrelatedAction extends BComponent {
  public void doTick() { /* housekeeping — does not write faultReset */ }
  public void doAckAlarm() { /* clears an alarm — does not write faultReset */ }
}
JAVA
  run "$EW" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"faultReset"* ]]
}


@test "EW-s22-neg2: an @NiagaraAction whose do-body writes a DIFFERENT slot does NOT exempt the OPERATOR slot (B831-G1 doAckAlarm shape)" {
  D="$BATS_TEST_TMPDIR/s22neg2"; mkdir -p "$D/com/x"
  cat > "$D/com/x/BAckWriter.java" <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="faultReset", type="BStatusBoolean", flags=Flags.SUMMARY|Flags.OPERATOR)
@NiagaraProperty(name="alarmAck",   type="BStatusBoolean", flags=Flags.SUMMARY|Flags.READONLY)
@NiagaraAction(name="ackAlarm")
public class BAckWriter extends BComponent {
  // action→do<Action>: ackAlarm's write lives in doAckAlarm, but it writes alarmAck, NOT faultReset.
  public void doAckAlarm() { setAlarmAck(new BStatusBoolean(true)); }
}
JAVA
  run "$EW" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"faultReset"* ]]
}
