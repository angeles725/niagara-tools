#!/usr/bin/env bats
# RED-FIRST pins for the timer-ticket conformance lint (campaign 6 PR8, from B787 · investigador1
# conformance audit). A control class that owns a Clock.Ticket (or calls Clock.schedule /
# schedulePeriodically) but has NO stopped() override that cancels the ticket leaks the timer on
# station stop — the real defect on ColdRoomPan BEvaporatorUnit (4 delay tickets cancelled on
# re-arm, never on stop); its siblings BDefrostController (stopped→cancelAll) and BCompressorControl
# (stopped→cancel) are conformant.
#
# SURFACE (provisional — the PR8 writer may fold it into verify-module.sh --src instead of a
# standalone script; if so I rebase these lines, kept in one file):
#   lint-timers.sh <java-root>
#     scans *.java under <java-root>; per class prints a row:
#       FAIL  lint-timers  <Class>  owns a timer with no stopped() cancel
#       PASS  lint-timers  <Class>  timer cancelled in stopped()
#       (a class with no timer emits no FAIL — PASS or is silently skipped)
#     exit 0 when no FAIL, exit 1 when any FAIL.
#
# RED today: build-n4-module-kit/toolbelt/lint-timers.sh does not exist → every case fails for the
# right reason (tool absent). Green once the impl lands the static check.
#
# NAMED MUTATION (run post-green): drop the stopped()-presence check → TL1's owner-without-stopped
# case no longer FAILs (exit 0), proving that check — not an unrelated grep — carries the bite.

load lib/client-root   # C11 T2: one blessed client read root; env override wins [ev: design.md D3b]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-timers.sh"
  SRC="$BATS_TEST_TMPDIR/src"
  mkdir -p "$SRC"

  # Owner WITHOUT stopped(): owns a Clock.Ticket + arms it, never cancels on stop → must FAIL.
  cat > "$SRC/Owner.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class Owner extends BComponent {
  private Clock.Ticket ticket;
  public void arm() { ticket = Clock.schedule(this, BRelTime.makeSeconds(5), armed, null); }
  // no stopped() override — the ticket leaks on station stop
}
JAVA

  # Conformant: same timer but cancels in stopped() → must PASS.
  cat > "$SRC/Conformant.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class Conformant extends BComponent {
  private Clock.Ticket ticket;
  public void arm() { ticket = Clock.schedule(this, BRelTime.makeSeconds(5), armed, null); }
  public void stopped() throws Exception { super.stopped(); if (ticket != null) ticket.cancel(); }
}
JAVA

  # No timers at all → must NOT FAIL.
  cat > "$SRC/NoTimer.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class NoTimer extends BComponent {
  private int count;
  public void bump() { count++; }
}
JAVA
}

@test "TL1: a class that owns a timer with NO stopped() cancel FAILs (exit 1, names the class)" {
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"Owner"* ]]
}

@test "TL2: a class that cancels its ticket in stopped() does NOT FAIL (Conformant is clean)" {
  # Isolate Conformant so TL2 asserts only its verdict.
  rm -f "$SRC/Owner.java" "$SRC/NoTimer.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "TL3: a class with no timer does NOT FAIL (no false positive on timerless code)" {
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "TL4: a Clock.schedule call with discarded return value FAILs (discarded-ticket)" {
  # Discard.java: owns a periodic ticket (properly cancelled in stopped), but also
  # fires a one-shot Clock.schedule without assigning the return value — discarded.
  # This fixture isolates discarded-ticket: timer-ticket would PASS (stopped cancels),
  # so only the discarded-ticket check carries the FAIL, making the mutation proof clean.
  cat > "$SRC/Discard.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class Discard extends BComponent {
  private Clock.Ticket periodic;
  public void start() {
    periodic = Clock.schedulePeriodically(this, BRelTime.makeSeconds(60), refresh, null);
    // fire-and-forget one-shot — discarded ticket:
    Clock.schedule(this, BRelTime.makeSeconds(1), init, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (periodic != null) periodic.cancel();
  }
}
JAVA
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java" "$SRC/NoTimer.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"discarded-ticket"* ]]
}

# ===========================================================================
# CAMPAIGN 8 EXTENSION — lint-timers.sh gains three lifecycle checks, all from
# [CERT-live] PANCCADIA/chihuahua production shapes. RED today: lint-timers.sh
# has only timer-ticket + discarded-ticket, so each FAIL pin sees the current
# tool return exit 0 (it does not implement these checks) → fails for the right
# reason (check absent). Each new test writes its fixture into its OWN dir so
# TL1–TL4's shared $SRC is never touched (they stay green). Fixtures are
# self-contained, sanitized Java (real shapes cited per check).
#
# NEW CHECK LABELS (contract the GREEN impl must emit in the row's <check> column):
#   companion-flag   TC-A   a boolean flag set true beside a Clock.schedule*
#                           assignment, not set false in stopped()
#   jdk-thread       TC-B   a JDK concurrency primitive inside a B* component/service
#   changed-sched    TC-C   a Clock.schedule* reachable from changed()/started()
#                           (≤1 level deep) with no isRunning()/atSteadyState() guard
#
# NAMED MUTATIONS (run post-green, one per check — proves each check carries its
# own bite, not an unrelated grep):
#   TC-A  accept ANY stopped() that cancels the ticket → companion-flag stops
#         firing → TC-A FAIL pin flips green.
#   TC-B  whitelist ScheduledExecutorService → jdk-thread stops firing → TC-B flips.
#   TC-C  drop the one-level-deep call following (only scan changed()/started()
#         bodies directly) → the indirect applyRunCmd() schedule is no longer
#         reachable → changed-sched stops firing → TC-C flips.
# ---------------------------------------------------------------------------

# ---- TC-A: companion-flag -------------------------------------------------
# Real shape: CompPan BCompressorControl.java — :1760 `startingUp = true;` next to
# :1764 `powerOnTicket = Clock.schedule(...)`; stopped() (:1799–1802) cancels the
# ticket only. The flag IS set false at :1864 but in the expiry handler, NOT in
# stopped()/started(); a component stop→start cycle (disable/enable, move, parent
# stop) keeps the same object, so if the stop lands mid-stagger the flag stays true.
# Decision (lead, first principles): a clear ONLY in the expiry path does NOT count;
# it must be cleared in stopped() (or started() — either breaks the carry-over).
# BStaggerHold reproduces the expiry-path-only clear exactly, so this pin also
# forces the impl to scope the clear to the lifecycle callbacks (a bare grep for
# `startingUp = false` anywhere would wrongly PASS it).
@test "TC-A: a flag cleared only in the expiry path (not stopped/started) FAILs naming the field" {
  D="$BATS_TEST_TMPDIR/tca"; mkdir -p "$D"
  cat > "$D/BStaggerHold.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BStaggerHold extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket powerOnTicket;
  public void arm() {
    startingUp = true;
    powerOnTicket = Clock.schedule(this, BRelTime.makeSeconds(5), powerOnExpired, null);
  }
  public void doPowerOnExpired() {
    startingUp = false;   // cleared ONLY in the expiry path (real BCompressorControl:1864) — does NOT count
  }
  public void stopped() throws Exception {
    super.stopped();
    if (powerOnTicket != null) { powerOnTicket.cancel(); powerOnTicket = null; }
    // BUG: startingUp is NOT cleared here; a stop mid-stagger leaves it stuck true for this run.
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"companion-flag"* ]]
  [[ "$output" == *"startingUp"* ]]     # names the offending field
}

@test "TC-A companion: the same flag CLEARED in stopped() does NOT FAIL (proves the check bites on the clear-location)" {
  D="$BATS_TEST_TMPDIR/tca_ok"; mkdir -p "$D"
  cat > "$D/BStaggerClean.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BStaggerClean extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket powerOnTicket;
  public void arm() {
    startingUp = true;
    powerOnTicket = Clock.schedule(this, BRelTime.makeSeconds(5), powerOnExpired, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (powerOnTicket != null) { powerOnTicket.cancel(); powerOnTicket = null; }
    startingUp = false;   // cleared on stop — conformant
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "TC-A companion: the flag cleared in started() (not stopped) does NOT FAIL (either lifecycle callback counts)" {
  # Lead decision: started() breaks the carry-over just as stopped() does, so a clear
  # there must also count. If the impl only accepts stopped(), this pin FAILs — it
  # forces started() into the accepted set. The real fix branch (fix/power-on-hold-vs-defrost
  # 268d70f) clears startingUp in stopped(); this companion pins the started() variant too.
  D="$BATS_TEST_TMPDIR/tca_started"; mkdir -p "$D"
  cat > "$D/BStaggerStarted.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BStaggerStarted extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket powerOnTicket;
  public void arm() {
    startingUp = true;
    powerOnTicket = Clock.schedule(this, BRelTime.makeSeconds(5), powerOnExpired, null);
  }
  public void started() throws Exception {
    super.started();
    startingUp = false;   // cleared on (re)start — breaks the carry-over just as stopped() would
  }
  public void stopped() throws Exception {
    super.stopped();
    if (powerOnTicket != null) { powerOnTicket.cancel(); powerOnTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

# ---- TC-B: jdk-thread -----------------------------------------------------
# Real shape: chihuahua BChiDashboardService.java — :89 `extends BAbstractService`,
# :229 `ScheduledExecutorService _tickScheduler`, :305 Executors.new*, :309 new Thread.
# A B* component/service must schedule via Clock, not a raw JDK thread pool (the pool
# is not tied to station lifecycle and survives stop). Isolated to ScheduledExecutorService
# as the sole trigger so the whitelist mutation flips this cleanly.
@test "TC-B: a ScheduledExecutorService inside a B* service FAILs (jdk-thread)" {
  D="$BATS_TEST_TMPDIR/tcb"; mkdir -p "$D"
  cat > "$D/BThreadSvc.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.Executors;
public class BThreadSvc extends BAbstractService {
  private ScheduledExecutorService _tickScheduler = null;
  public void serviceStarted() throws Exception {
    _tickScheduler = Executors.newSingleThreadScheduledExecutor();
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"jdk-thread"* ]]
  [[ "$output" == *"BThreadSvc"* ]]
}

@test "TC-B companion: a plain (non-B*) helper using ScheduledExecutorService does NOT FAIL (the B* gate)" {
  D="$BATS_TEST_TMPDIR/tcb_ok"; mkdir -p "$D"
  cat > "$D/PlainPool.java" <<'JAVA'
package demo;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.Executors;
public final class PlainPool {
  private final ScheduledExecutorService pool = Executors.newSingleThreadScheduledExecutor();
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

# ---- TC-C: changed-sched --------------------------------------------------
# Real shape: ColdRoomPan BEvaporatorUnit.java — changed() (:849) → applyRunCmd()
# (:876, one level deep) → Clock.schedule (:900/:913). Pre-fix applyRunCmd had NO
# steady-state guard → the engine rejected the schedule during changed() before
# steady state (NotRunningException ×6 in PANCCADIA logs). The fix adds
# :885 `if (!Sys.atSteadyState()) return;`. Both fixtures cancel their tickets in
# stopped() so timer-ticket PASSes — only changed-sched can carry the FAIL.
@test "TC-C: Clock.schedule reachable from changed() via a private method with no steady-state guard FAILs" {
  D="$BATS_TEST_TMPDIR/tcc"; mkdir -p "$D"
  cat > "$D/BUnitPre.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BUnitPre extends BComponent {
  private Clock.Ticket startDelayTicket;
  public void changed(Property p, Context cx) {
    if (!isRunning()) return;
    applyRunCmd();                 // one level deep -> schedules with no steady-state guard
  }
  void applyRunCmd() {
    startDelayTicket = Clock.schedule(this, BRelTime.makeSeconds(5), startDelayExpired, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (startDelayTicket != null) { startDelayTicket.cancel(); startDelayTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"changed-sched"* ]]
}

@test "TC-C companion: the SAME chain guarded by !Sys.atSteadyState() return does NOT FAIL (the fix shape)" {
  D="$BATS_TEST_TMPDIR/tcc_ok"; mkdir -p "$D"
  cat > "$D/BUnitFixed.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BUnitFixed extends BComponent {
  private Clock.Ticket startDelayTicket;
  public void changed(Property p, Context cx) {
    if (!isRunning()) return;
    applyRunCmd();
  }
  void applyRunCmd() {
    if (!Sys.atSteadyState()) return;   // the fix — no schedule before steady state
    startDelayTicket = Clock.schedule(this, BRelTime.makeSeconds(5), startDelayExpired, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (startDelayTicket != null) { startDelayTicket.cancel(); startDelayTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

# ---- TC-D: D9b dot-dir prune ----------------------------------------------
# Design addendum D9b: every kit source scanner prunes dot-directories (.deploy-baseline snapshots,
# .git, etc.). A real defect that lives ONLY under a dot-dir must NOT be flagged — else report-module's
# schema-risk baseline (kept at <artifact>/.deploy-baseline) makes lint-timers double-count the previous
# deploy's classes. RED today: lint-timers.sh does `find <root> -name '*.java'` with NO prune, so it
# DOES flag the leak under .deploy-baseline/ -> the pin fails for the right reason (prune not yet added).
# NAMED MUTATION (post-green): remove the dot-dir prune -> BStale's leak is flagged again -> TC-D flips.
@test "TC-D: a timer leak under .deploy-baseline/ is NOT flagged (dot-dir pruned)" {
  D="$BATS_TEST_TMPDIR/tcd"; mkdir -p "$D/.deploy-baseline/com/x"
  cat > "$D/.deploy-baseline/com/x/BStale.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BStale extends BComponent {
  private Clock.Ticket t;
  public void arm() { t = Clock.schedule(this, BRelTime.makeSeconds(5), armed, null); }
  // no stopped() cancel -> a real leak, but it lives under .deploy-baseline/ so it must be pruned
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"BStale"* ]]
}

# ===========================================================================
# S21 (C10 lint-precision RED): companion-flag must key on a FIELD flag, not a
# method-LOCAL boolean. A local `boolean x = false; … x = true;` is re-initialised
# every call, so it cannot stay stuck across a stop/restart — only a class FIELD
# can. Real FP on main df8c7ec: BDefrostController.java `anyNoHardware` (a method
# local) FAILs beside a Clock.schedule in the same method.
# RED-for-the-right-reason on df8c7ec: S21-neg + S21-smoke FAIL (the tool FPs on a
# local); the existing field-flag TC-A pins stay FAIL (regression guard).
# ---------------------------------------------------------------------------
@test "S21-neg: a method-LOCAL boolean set true beside Clock.schedule* does NOT FAIL (not a stuck-flag hazard)" {
  D="$BATS_TEST_TMPDIR/s21neg"; mkdir -p "$D"
  cat > "$D/BLocalFlag.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BLocalFlag extends BComponent {
  private Clock.Ticket durationTicket;
  public void armDefrost() {
    boolean anyQueued     = false;   // METHOD-LOCAL — re-init each call, cannot stay stuck
    boolean anyNoHardware = false;
    for (int i = 0; i < 3; i++) { anyNoHardware = true; }   // local set true …
    durationTicket = Clock.schedule(this, BRelTime.makeSeconds(5), durationExpired, null); // … beside a schedule
    if (anyQueued) { /* … */ }
  }
  public void stopped() throws Exception {
    super.stopped();
    if (durationTicket != null) { durationTicket.cancel(); durationTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"companion-flag"* ]]
  [[ "$output" != *"anyNoHardware"* ]]
}

@test "S21-pos: a FIELD flag set true beside Clock.schedule*, cleared only off-lifecycle, STILL FAILs (regression guard)" {
  D="$BATS_TEST_TMPDIR/s21pos"; mkdir -p "$D"
  cat > "$D/BFieldFlag.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BFieldFlag extends BComponent {
  private boolean startingUp = false;   // class FIELD — CAN stay stuck across stop/restart
  private Clock.Ticket powerOnTicket;
  public void arm() {
    startingUp = true;
    powerOnTicket = Clock.schedule(this, BRelTime.makeSeconds(5), powerOnExpired, null);
  }
  public void doPowerOnExpired() { startingUp = false; }   // cleared only in expiry — does NOT count
  public void stopped() throws Exception {
    super.stopped();
    if (powerOnTicket != null) { powerOnTicket.cancel(); powerOnTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"companion-flag"* ]] && [[ "$output" == *"startingUp"* ]]
}

@test "S21-smoke: real ColdRoomPan-rt (main-ff1b659) is companion-flag CLEAN -> lint-timers exit 0" {
  ROOT="$C9_CLIENT_ROOT"   # via client-root.bash; env override wins [ev: design.md D3c site 4]
  RT="$ROOT/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"
  [ -d "$RT" ] || skip "client read tree not on this machine (set C9_CLIENT_ROOT)"
  run "$LINT" "$RT"
  [ "$status" -eq 0 ]
  [[ "$output" != *"companion-flag"* ]]
}
# ---------------------------------------------------------------------------
# S21-misparse (Phase-2 brace_depth>=2 method-detect guard, from investigador1's
# BMisparse): a class-level @NiagaraProperty whose defaultValue holds a constructor
# call ("new BAlarmRecord()") sits at depth 1. Without the depth guard the
# annotation's parens are mis-read as a method open, merging armA (FIELD flag) and
# armB (Clock.schedule) into one pseudo-method -> a false companion-flag FAIL. The
# guard (only depth>=2 opens a method) keeps armA/armB separate -> CLEAN. RED on
# df8c7ec (pre-guard mis-parse false-FAILs); the fix is clean, and dropping the
# guard re-introduces the false FAIL.
@test "S21-misparse: a depth-1 @NiagaraProperty(defaultValue=\"new BAlarmRecord()\") does NOT mis-parse into a method; cross-method flag/schedule stay CLEAN" {
  D="$BATS_TEST_TMPDIR/s21mis"; mkdir -p "$D"
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
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"companion-flag"* ]]
}
