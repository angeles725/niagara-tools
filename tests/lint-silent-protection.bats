#!/usr/bin/env bats
# RED-FIRST pins for lint-silent-protection.sh (campaign 9 S18, from B824 b5060f60b; sibling of B820).
# A SOURCE lint (like lint-delays.sh), WARN-only (--strict -> FAIL). Flags a PROTECTION TRIP taken
# SILENTLY: a decision that forces an output OFF / sheds a stage / holds a valve on a condition over a
# process variable, that reaches NO operator surface (B821 §821.4 tier-4: no SUMMARY/OPERATOR
# status-or-reason slot written on its path, no BAlarmSourceExt, no readable reason field surfaced).
#
# Real dirty shapes (must FLAG): CP-1 low-suction shed — CompressorControl.step
#   `if (suctionValid && suction < c.suctionLowLimit) target = Math.min(target, onCount-1)` (inline, no
#   named slot); CR-3 `private boolean freezeTripped` forcing the liquid solenoid closed, no status slot.
# Clean anchors (must NOT flag): CP-2 `dischargeHigh` -> surfaced via `dischargeHighAlarm` SUMMARY slot;
#   the defrostSkipped/lastSkipReason surfaced pattern (*Skip*/*Reason* on the allowlist). [ev: corpus B824]
#
# SURFACE: lint-silent-protection.sh [--strict] <java-src-dir>
#   Row:  WARN  lint-silent-protection  <file>:<line>  <method> forces <output>/sheds stage on <cond>
#         — no status/reason/alarm surface in scope; add a *Alarm/*Reason SUMMARY slot or a BAlarmSourceExt
#   Surface-name allowlist (SUMMARY/OPERATOR, NOT the forced output):
#     *Alarm *Fault *Skip* *Reason *Status *Mismatch *Stuck *Available *Fallback
#   Effect-slot exemption: the trip's OWN forced output slot does NOT count as a surface.
#   Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
#
# RED today: lint-silent-protection.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (§824.5, post-green, on mktemp copies): remove the *Skip*/*Reason* writes from the clean
#   fixture -> the previously-clean skip path emits exactly 1 WARN (SP8 pins this directly).

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LSP="$KIT/toolbelt/lint-silent-protection.sh"
  S="$BATS_TEST_TMPDIR/src"
  ROOT="${C9_CLIENT_ROOT:-/home/cristian/modulos_niagara_n4/Cliente/Leon-Guanjuato-worktrees/main-a109249}"   # RP1: blessed read tree, never the local working copy
}
fresh() { rm -rf "$S"; mkdir -p "$S/com/x"; }

# ---- SP1: a silent inline shed (no named field/slot) -> exactly one WARN, grammar names it ----
@test "SP1: flags_inline_output_force_on_limit_with_no_slot (1 WARN, §824.2 grammar)" {
  fresh
  cat > "$S/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  int step(int target, int onCount, double suction, double suctionLowLimit, boolean suctionValid) {
    if (suctionValid && suction < suctionLowLimit) target = Math.min(target, onCount - 1); // hard LP floor -> shed
    return target;
  }
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'WARN')" -eq 1 ]
  [[ "$output" == *"lint-silent-protection"* ]] && [[ "$output" == *"CompressorControl.java"* ]]
  [[ "$output" == *"no status/reason/alarm surface"* ]]
}

# ---- SP2: a named field followed to an *Alarm SUMMARY slot (CP-2) -> clean ----
@test "SP2: clean_named_field_followed_to_alarm_slot (0 WARN)" {
  fresh
  cat > "$S/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  boolean dischargeHigh = false;
  int step(int target, double discharge, double dischargeLimit) {
    if (discharge > dischargeLimit) { dischargeHigh = true; target = 0; } // named trip field
    return target;
  }
}
JAVA
  cat > "$S/com/x/BCompressorControl.java" <<'JAVA'
package com.x;
public class BCompressorControl extends BComponent {
  @NiagaraProperty(name="dischargeHighAlarm", type="BStatusBoolean", defaultValue="new BStatusBoolean(false)", flags=Flags.TRANSIENT|Flags.SUMMARY|Flags.READONLY)
  void sync(CompressorControl ctl) { getDischargeHighAlarm().setValue(ctl.dischargeHigh); }
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# ---- SP3: the trip writes only its OWN forced output SUMMARY slot -> still WARN (effect != surface) ----
@test "SP3: effect_slot_exemption_does_not_count_as_surface (still 1 WARN)" {
  fresh
  cat > "$S/com/x/BEvaporatorUnit.java" <<'JAVA'
package com.x;
public class BEvaporatorUnit extends BComponent {
  @NiagaraProperty(name="valveOut", type="BStatusBoolean", defaultValue="new BStatusBoolean(false)", flags=Flags.SUMMARY|Flags.READONLY)
  void recompute(boolean coilValid, double coil, double freezeSp) {
    boolean freezeTripped = coilValid && coil < freezeSp;      // trip decision
    if (freezeTripped) getValveOut().setValue(false);          // forces its OWN output only
  }
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'WARN')" -eq 1 ]
  [[ "$output" == *"valveOut"* ]] || [[ "$output" == *"freezeTripped"* ]]
}

# ---- SP4: a *Skip*/*Reason* SUMMARY surface on the trip path -> clean (allowlist) ----
@test "SP4: allowlist_reason_slot_makes_it_clean (0 WARN; same private-latch trip shape as SP8 + a *Reason SUMMARY write -> the allowlist is what clears it)" {
  fresh
  cat > "$S/com/x/BDefrostController.java" <<'JAVA'
package com.x;
public class BDefrostController extends BComponent {
  @NiagaraProperty(name="defrostSkipped", type="BStatusBoolean", defaultValue="new BStatusBoolean(false)", flags=Flags.SUMMARY|Flags.READONLY)
  @NiagaraProperty(name="lastSkipReason", type="BString", defaultValue="BString.make(\"\")", flags=Flags.SUMMARY|Flags.READONLY)
  void maybeSkip(boolean overdue, boolean doorOpen) {
    if (doorOpen) { this.defrostSkipped = true; setLastSkipReason("door open"); return; } // private latch + surfaced reason
  }
  private boolean defrostSkipped = false;
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

# ---- SP5: --strict promotes the WARN to exit 1 ----
@test "SP5: strict_flag_promotes_warn_to_fail" {
  fresh
  cat > "$S/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  int step(int target, int onCount, double suction, double lowLimit, boolean valid) {
    if (valid && suction < lowLimit) target = Math.min(target, onCount - 1); // silent shed
    return target;
  }
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  run "$LSP" --strict "$S"
  [ "$status" -eq 1 ]
}

# ---- SP6: no <src-dir> argument -> exit 3 ----
@test "SP6: no src-dir argument -> exit 3" {
  run "$LSP"
  [ "$status" -eq 3 ]
}

# ---- SP7: a silent trip under .deploy-baseline/ is NOT flagged (D9b dot-dir prune) ----
@test "SP7: dot-dir prune (D9b) — a silent shed under .deploy-baseline/ is not counted" {
  fresh; mkdir -p "$S/.deploy-baseline/com/x"
  cat > "$S/com/x/Clean.java" <<'JAVA'
package com.x;
public class Clean { int noop(int t) { return t; } }
JAVA
  cat > "$S/.deploy-baseline/com/x/Stale.java" <<'JAVA'
package com.x;
public class Stale {
  int step(int target, int onCount, double suction, double lowLimit, boolean valid) {
    if (valid && suction < lowLimit) target = Math.min(target, onCount - 1);
    return target;
  }
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]] && [[ "$output" != *"Stale"* ]]
}

# ---- SP8: NAMED MUTATION (§824.5) — remove the *Skip*/*Reason* surface -> the flag APPEARS ----
@test "SP8: mutation_removing_reason_slot_makes_flag_appear (proves the check keys on the surface)" {
  fresh
  # same trip as SP4 but WITHOUT the surfaced slots -> now silent -> must WARN
  cat > "$S/com/x/BDefrostController.java" <<'JAVA'
package com.x;
public class BDefrostController extends BComponent {
  void maybeSkip(boolean overdue, boolean doorOpen) {
    if (doorOpen) { this.defrostSkipped = true; return; } // trip with NO surfaced slot
  }
  private boolean defrostSkipped = false;
}
JAVA
  run "$LSP" "$S"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'WARN')" -eq 1 ]
}

# ---- SP-smoke: the REAL client trees must flag CP-1 + CR-3 and NOT flag CP-2 (SKIP if absent) ----
@test "SP-smoke: EXACT contract at a109249 (C9_CLIENT_ROOT) — CompPan-rt exactly 1 WARN (CompressorControl.java:215, CP-1), ColdRoomPan-rt exactly 1 (BEvaporatorUnit.java:1287, CR-3), DashboardPan-rt 0, DashboardPan-ux 0; CP-2/defrostSkipped/getters never" {
  [ -d "$ROOT/Compresores" ] && [ -d "$ROOT/Paccadia" ] && [ -d "$ROOT/Dashboard" ] || skip "client read tree not on this machine (set C9_CLIENT_ROOT)"
  run "$LSP" "$ROOT/Compresores/CompPan/CompPan-rt/src"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
  [[ "$output" == *"CompressorControl.java:215"* ]]           # CP-1 low-suction shed, a109249 anchor
  [[ "$output" != *"dischargeHighAlarm"* ]]                    # CP-2 is surfaced -> never a subject
  [[ "$output" != *"BCompressorControl"* ]]                    # adapter getters are not trips
  run "$LSP" "$ROOT/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
  [[ "$output" == *"BEvaporatorUnit.java:1287"* ]]            # CR-3 freezeTripped private latch (ABSENT once PR8 wires Pattern A: re-pin to 0 then, R3<->R8)
  [[ "$output" != *"defrostSkipped"* ]]                        # surfaced via slot -> never a subject
  run "$LSP" "$ROOT/Dashboard/DashboardPan/DashboardPan-rt/src"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
  run "$LSP" "$ROOT/Dashboard/DashboardPan/DashboardPan-ux/src"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 0 ]
}

# --- SP9: a source dir with NO Java files -> exit 3 + ERROR row, never a silent 0 (K20 / C8 silent-0 lesson; WP9b shape) ---
@test "SP9: no-sources-exit-3 (empty dir and a dir with only a non-Java file -> exit 3 + ERROR row, no WARN)" {
  E="$BATS_TEST_TMPDIR/empty"; rm -rf "$E"; mkdir -p "$E"
  run "$LSP" "$E"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]] && [[ "$output" == *"lint-silent-protection"* ]] && [[ "$output" != *"WARN"* ]]
  N="$BATS_TEST_TMPDIR/nojava"; rm -rf "$N"; mkdir -p "$N/com/x"; printf 'not java\n' > "$N/com/x/README.txt"
  run "$LSP" "$N"
  [ "$status" -eq 3 ]
  [[ "$output" == *"ERROR"* ]] && [[ "$output" != *"WARN"* ]]
}
