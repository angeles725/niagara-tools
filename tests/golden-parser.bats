#!/usr/bin/env bats
# C11 T1 — the GOLDEN SET the shared section-D method-boundary parser must satisfy
# across all three lints. lint-timers.sh, lint-silent-protection.sh and
# lint-ext-writable-shape.sh each carry a copy of the parser; T1 extracts one shared
# fragment. This file is the single cross-lint contract that fragment must pass: the
# same method-boundary invariants (depth>=2 guard, Case-B @-line stop, keyword
# exclusion, one-liner detection via PEAK depth, accessor skip) observed through each
# lint's output. [ev: investigador1 B832 593019540; companero C11 T1]
#
# On dab0807 the two NET copies (timers, silent) MISS one-liner methods -> the two
# G-oneliner-* pins are RED; ext-writable already uses PEAK, so G-oneliner-extwritable
# is the GREEN reference. The multi-line / same-method / adapter / accessor cases are
# GREEN regression guards. After T1 (all three on PEAK + accessor skip) all pass.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LT="$KIT/toolbelt/lint-timers.sh"
  LSP="$KIT/toolbelt/lint-silent-protection.sh"
  EW="$KIT/toolbelt/lint-ext-writable-shape.sh"
}

# --- GUARD: a depth-1 multi-line @NiagaraProperty is not a method; cross-method flag/schedule stay clean ---
@test "G-multiline: BMisparse (multi-line @NiagaraProperty defaultValue) is not mis-parsed into a method (lint-timers exit 0)" {
  D="$BATS_TEST_TMPDIR/g-multiline"; mkdir -p "$D"
  cat > "$D/BMisparse.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
@NiagaraType
@NiagaraProperty(
  name = "ackAlarm",
  type = "BAlarmRecord",
  defaultValue = "new BAlarmRecord()"
)
public final class BMisparse extends BComponent
{
  private boolean startingUp = false;
  private Clock.Ticket t;
  public void armA() { startingUp = true; }                                              // FIELD flag, method A
  public void armB() { t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null); }    // schedule, method B
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 0 ]
}

# --- GUARD: a method-LOCAL boolean is re-init each call — not a companion-flag hazard ---
@test "G-samemethod: anyNoHardware (method-local bool set true beside Clock.schedule) does NOT FAIL (lint-timers exit 0)" {
  D="$BATS_TEST_TMPDIR/g-samemethod"; mkdir -p "$D"
  cat > "$D/BLocalFlag.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BLocalFlag extends BComponent {
  private Clock.Ticket durationTicket;
  public void armDefrost() {
    boolean anyNoHardware = false;
    for (int i = 0; i < 3; i++) { anyNoHardware = true; }
    durationTicket = Clock.schedule(this, BRelTime.makeSeconds(5), durationExpired, null);
    if (anyNoHardware) { /* … */ }
  }
  public void stopped() throws Exception { super.stopped(); if (durationTicket != null) { durationTicket.cancel(); durationTicket = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 0 ]
}

# --- GUARD: a BIAlarmSource + newOffnormalAlarm adapter is a Pattern-B surface for a core trip ---
@test "G-adapter: CP-1 (BIAlarmSource + newOffnormalAlarm adapter) surfaces the trip -> lint-silent-protection 0 WARN" {
  D="$BATS_TEST_TMPDIR/g-adapter"; mkdir -p "$D/com/x"
  cat > "$D/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  int step(int target, int onCount, double suction, double suctionLowLimit, boolean suctionValid) {
    if (suctionValid && suction < suctionLowLimit) target = Math.min(target, onCount - 1); // LP floor shed (trip)
    return target;
  }
}
JAVA
  cat > "$D/com/x/BCompressorControl.java" <<'JAVA'
package com.x;
public class BCompressorControl extends BComponent implements BIAlarmSource {
  void raise() { alarmSupport.newOffnormalAlarm(mkData()); }   // Pattern-B surface for the core's trip
}
JAVA
  run "$LSP" "$D"
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
}

# --- RED on dab0807: one-liner companion-flag missed by the NET timers parser ---
@test "G-oneliner-timers: a FIELD flag set true beside Clock.schedule in a ONE-LINER method FAILs companion-flag (RED on dab0807 NET; GREEN under PEAK)" {
  D="$BATS_TEST_TMPDIR/g-ol-timers"; mkdir -p "$D"
  cat > "$D/BOneLinerArm.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BOneLinerArm extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket powerOnTicket;
  public void arm() { startingUp = true; powerOnTicket = Clock.schedule(this, BRelTime.makeSeconds(5), powerOnExpired, null); }
  public void doPowerOnExpired() { startingUp = false; }
  public void stopped() throws Exception { super.stopped(); if (powerOnTicket != null) { powerOnTicket.cancel(); powerOnTicket = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"companion-flag"* ]] && [[ "$output" == *"startingUp"* ]]
}

# --- RED on dab0807: one-liner trip missed by the NET silent-protection parser ---
@test "G-oneliner-silent: a detected trip in a ONE-LINER method with no surface WARNs (RED on dab0807 NET; GREEN under PEAK)" {
  D="$BATS_TEST_TMPDIR/g-ol-silent"; mkdir -p "$D/com/x"
  cat > "$D/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  int step(int target, int onCount, double suction, double suctionLowLimit, boolean suctionValid) { if (suctionValid && suction < suctionLowLimit) target = Math.min(target, onCount - 1); return target; }
}
JAVA
  run "$LSP" "$D"
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
}

# --- REFERENCE: ext-writable already uses PEAK — a one-liner do<Action> IS seen (slot exempt) ---
@test "G-oneliner-extwritable: ext-writable (PEAK reference) sees a ONE-LINER do<Action> body -> slot exempt, 0 WARN (this is the target behavior for the shared fragment)" {
  D="$BATS_TEST_TMPDIR/g-ol-ew"; mkdir -p "$D/com/x"
  cat > "$D/com/x/BWithWriter.java" <<'JAVA'
package com.x;
@NiagaraType
@NiagaraProperty(name="setpoint", type="BStatusNumeric", flags=Flags.SUMMARY|Flags.OPERATOR)
@NiagaraAction(name="bumpSetpoint")
public class BWithWriter extends BComponent {
  public void doBumpSetpoint() { setSetpoint(new BStatusNumeric(getSetpoint().getValue()+1)); }
}
JAVA
  run "$EW" "$D"
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
}

# --- GUARD (B832-G1): a one-liner accessor is not protection logic — must be skipped, not a trip ---
@test "G-accessor: a ONE-LINER accessor (setInhibited) with a guarded boolean write is skipped -> 0 WARN (flips a naive PEAK-only fix without the get/set/is skip)" {
  D="$BATS_TEST_TMPDIR/g-accessor"; mkdir -p "$D/com/x"
  cat > "$D/com/x/Widget.java" <<'JAVA'
package com.x;
public class Widget {
  private boolean inhibited;
  public void setInhibited(boolean b) { if (b) this.inhibited = true; }
}
JAVA
  run "$LSP" "$D"
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
}
