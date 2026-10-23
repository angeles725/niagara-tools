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
#   jdk-thread       TC-B   a JDK concurrency primitive inside a B* component/service (re-scoped
#                           2026-10-22: FAIL only a thread-modifying call without MANAGE_EXECUTION)
#   changed-sched    TC-C   a Clock.schedule* reachable from changed()/started()
#                           (≤1 level deep) with no isRunning()/atSteadyState() guard
#
# NAMED MUTATIONS (run post-green, one per check — proves each check carries its
# own bite, not an unrelated grep):
#   TC-A  accept ANY stopped() that cancels the ticket → companion-flag stops
#         firing → TC-A FAIL pin flips green.
#   TC-B  ignore the shutdownNow() call → the modifying-call FAIL stops firing → TC-B flips.
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

# ---- TC-B / JT: jdk-thread (re-scoped, quality round 8 T9b) -----------------
# Creating and starting a JDK thread in a B* class is allowed (Tridium Lab 9 runs a dedicated
# Thread and an ExecutorService in production shape). The real defect is MODIFYING a thread
# (shutdown/shutdownNow/interrupt/setPriority/setDaemon, ExecutorUtil.shutdown) without the
# MANAGE_EXECUTION permission group: chihuahua BChiDashboardService logged "failed to unschedule
# controlTick: access denied (modifyThread)" at every stop (corpus B800 §800.3). FAIL = that call
# with no station/all MANAGE_EXECUTION in the nearest module-permissions.xml; WARN otherwise.
# Layout: $D/m-rt/module-permissions.xml + $D/m-rt/src/demo/<File>.java, lint run on $D/m-rt/src.
_jt_src() {  # _jt_src <dir> <class-file-basename> <java body...>: writes one Java file under <dir>/m-rt/src/demo
  local d="$1" c="$2"; shift 2
  mkdir -p "$d/m-rt/src/demo"
  printf '%s\n' "$@" > "$d/m-rt/src/demo/$c.java"
}
_jt_perms() {  # _jt_perms <dir> <type-attr-of-the-group-holding-MANAGE_EXECUTION | none>
  local d="$1" t="$2"
  mkdir -p "$d/m-rt"
  {
    printf '<permissions>\n'
    printf '  <niagara-permission-groups type="station">\n'
    printf '    <req-permission><name>NETWORK_COMMUNICATION</name></req-permission>\n'
    printf '  </niagara-permission-groups>\n'
    if [ "$t" != none ]; then
      printf '  <niagara-permission-groups type="%s">\n' "$t"
      printf '    <req-permission>\n      <name>MANAGE_EXECUTION</name>\n'
      printf '      <purposeKey>Stop running station threads during station shutdown</purposeKey>\n    </req-permission>\n'
      printf '  </niagara-permission-groups>\n'
    fi
    printf '</permissions>\n'
  } > "$d/m-rt/module-permissions.xml"
}
_jt_exec() {  # a Lab-9-shaped executor component: two-line class header, shutdownNow() in stopped()
  _jt_src "$1" BExec 'package demo;' 'import javax.baja.sys.*;' 'import java.util.concurrent.*;' \
    'public class BExec' '  extends BComponent' '{' \
    '  public void started() { executorService = Executors.newSingleThreadExecutor(); }' \
    '  public void stopped() { if (executorService != null) executorService.shutdownNow(); }' \
    '  private ExecutorService executorService;' '}'
}

@test "TC-B: a thread-modifying call in a B* class with no MANAGE_EXECUTION FAILs (jdk-thread)" {
  D="$BATS_TEST_TMPDIR/tcb"; _jt_exec "$D"; _jt_perms "$D" none
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  jdk-thread"*"BExec"*"MANAGE_EXECUTION"* ]]
}

@test "JT-perm: the same executor with MANAGE_EXECUTION under type=station only WARNs (exit 0)" {
  D="$BATS_TEST_TMPDIR/jtperm"; _jt_exec "$D"; _jt_perms "$D" station
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  jdk-thread"*"BExec"* ]]
  [[ "$output" != *"FAIL"* ]]
}

@test "JT-all: MANAGE_EXECUTION under type=all also counts as granted" {
  D="$BATS_TEST_TMPDIR/jtall"; _jt_exec "$D"; _jt_perms "$D" all
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  jdk-thread"* ]]
}

@test "JT-scope: MANAGE_EXECUTION only under type=workbench does not grant the station and FAILs" {
  D="$BATS_TEST_TMPDIR/jtscope"; _jt_exec "$D"; _jt_perms "$D" workbench
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  jdk-thread"* ]]
}

@test "JT-header: a two-line class header (class / extends on the next line) is still a B* class" {
  D="$BATS_TEST_TMPDIR/jthdr"
  _jt_src "$D" BDedicated 'package demo;' 'import javax.baja.sys.*;' \
    'public class BDedicated' '  extends BComponent' '  implements Runnable' '{' \
    '  public void doExecute() { new Thread(this, "demo-dedicated").start(); }' \
    '  public void run() { }' '}'
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  jdk-thread"*"BDedicated"* ]]
}

@test "JT-create: thread creation alone, no permissions file, WARNs and never FAILs" {
  D="$BATS_TEST_TMPDIR/jtcreate"
  _jt_src "$D" BPoolSvc 'package demo;' 'import javax.baja.sys.*;' 'import java.util.concurrent.*;' \
    'public class BPoolSvc extends BAbstractService {' \
    '  private ScheduledExecutorService pool = Executors.newSingleThreadScheduledExecutor();' '}'
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  jdk-thread"*"BPoolSvc"*"ExecutorUtil"* ]]
}

@test "JT-comment: a shutdownNow() only in a comment or a string is not a modifying call" {
  D="$BATS_TEST_TMPDIR/jtcomment"; _jt_perms "$D" none
  _jt_src "$D" BQuiet 'package demo;' 'import javax.baja.sys.*;' 'import java.util.concurrent.*;' \
    'public class BQuiet extends BComponent {' \
    '  private ExecutorService ex = Executors.newSingleThreadExecutor();' \
    '  // ex.shutdownNow(); needs MANAGE_EXECUTION' \
    '  public String note() { return "call ex.shutdownNow() later"; }' '}'
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  jdk-thread"* ]]
}

@test "JT-self: Thread.currentThread().interrupt() restores the caller's own flag and is not a modifying call" {
  D="$BATS_TEST_TMPDIR/jtself"; _jt_perms "$D" none
  _jt_src "$D" BSleeper 'package demo;' 'import javax.baja.sys.*;' \
    'public class BSleeper extends BComponent {' \
    '  public void doWork() { new Thread(() -> { try { Thread.sleep(10L); }' \
    '    catch (InterruptedException e) { Thread.currentThread().interrupt(); } }, "demo-sleeper").start(); }' '}'
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  jdk-thread"* ]]
}

@test "JT-util: ExecutorUtil.shutdown(svc) is a modifying call (it ends in shutdownNow)" {
  D="$BATS_TEST_TMPDIR/jtutil"; _jt_perms "$D" none
  _jt_src "$D" BUtilSvc 'package demo;' 'import javax.baja.sys.*;' 'import javax.baja.util.ExecutorUtil;' 'import java.util.concurrent.*;' \
    'public class BUtilSvc extends BComponent {' \
    '  public void started() { svc = ExecutorUtil.newSingleThreadBackgroundExecutor("demo-io", 60L, TimeUnit.SECONDS); }' \
    '  public void stopped() { ExecutorUtil.shutdown(svc); }' \
    '  private ExecutorService svc;' '}'
  run "$LINT" "$D/m-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  jdk-thread"*"BUtilSvc"* ]]
}

@test "TC-B companion: a plain (non-B*) helper using ScheduledExecutorService emits no jdk-thread row (the B* gate)" {
  D="$BATS_TEST_TMPDIR/tcb_ok"; mkdir -p "$D"
  cat > "$D/PlainPool.java" <<'JAVA'
package demo;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.Executors;
public final class PlainPool {
  private final ScheduledExecutorService pool = Executors.newSingleThreadScheduledExecutor();
  void stop() { pool.shutdownNow(); }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"jdk-thread"* ]]
}

# ---- JC: job-cancel (WARN) ---------------------------------------------------
# A BSimpleJob's run(Context) loop must check isAlive() and throw JobCancelException:
# doCancel only interrupts the job thread, which reaches a thread blocked in I/O or sleep
# (S/job/BSimpleJob.java:54-67, course Lab 10 BHistoriesToCsvJob.java:105-112).
_jc_job() {  # _jc_job <dir> <run-body-lines...>
  local d="$1"; shift
  mkdir -p "$d"
  printf '%s\n' 'package demo;' 'import javax.baja.job.*;' 'import javax.baja.sys.*;' \
    'public class BExportJob' '  extends BSimpleJob' '{' '  public BExportJob() {}' \
    '  @Override' '  public void run(Context cx) throws Exception' '  {' "$@" '  }' \
    '  private String[] names = new String[0];' '}' > "$d/BExportJob.java"
}

@test "JC-loop: a BSimpleJob run() loop with no isAlive() WARNs job-cancel (exit 0)" {
  D="$BATS_TEST_TMPDIR/jcloop"
  _jc_job "$D" '    for (int i = 0; i < names.length; i++) {' '      progress(i);' '    }'
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  job-cancel"*"BExportJob"*"isAlive"* ]]
}

@test "JC-alive: the same loop checking isAlive() emits no job-cancel row" {
  D="$BATS_TEST_TMPDIR/jcalive"
  _jc_job "$D" '    for (int i = 0; i < names.length; i++) {' \
    '      if (!isAlive()) throw new JobCancelException();' '      progress(i);' '    }'
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"job-cancel"* ]]
}

@test "JC-scope: a loop in a helper method, not in run(), emits no job-cancel row" {
  D="$BATS_TEST_TMPDIR/jcscope"
  _jc_job "$D" '    progress(50);' '  }' '  void helper() {' '    for (int i = 0; i < 3; i++) { progress(i); }'
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"job-cancel"* ]]
}

@test "JC-noloop: a BSimpleJob run() without a loop emits no job-cancel row" {
  D="$BATS_TEST_TMPDIR/jcnoloop"
  _jc_job "$D" '    progress(50);' '    log().message("demo", "done");'
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"job-cancel"* ]]
}

@test "JC-comment: isAlive() only in a comment of the run() loop does not count" {
  D="$BATS_TEST_TMPDIR/jccomment"
  _jc_job "$D" '    // if (!isAlive()) throw new JobCancelException();' '    while (more()) { step(); }'
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  job-cancel"* ]]
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
# ---------------------------------------------------------------------------
# ===========================================================================
# ATSS — atSteadyState-only-timer WARN (retro 2026-09-18-insights-issues-catalog-deltas Δ2)
# A class that arms a timer ONLY in atSteadyState() but never overrides started()
# silently fails to arm the timer on commissioning-time mounts (the component
# receives started() but NOT atSteadyState() when dragged onto an already-running
# station).  Live case: BDefrostController, PANCCADIA León.  [ev: corpus B729 §729.4]
#
# Check label: atSteadyState-only-timer
# Severity: WARN (exit 0 even when emitted — a BTimeTrigger subclass may rely on
#           the parent's started()); FAIL is NOT set.
#
# ATSS1 — class with atSteadyState() timer + no started() override → WARN row emitted, exit 0
# ATSS2 — same shape WITH started() override → no WARN, exit 0 (clean)
# ===========================================================================

@test "ATSS1: a class with Clock.schedulePeriodically in atSteadyState() and NO started() override emits WARN atSteadyState-only-timer (exit 0)" {
  D="$BATS_TEST_TMPDIR/atss1"; mkdir -p "$D"
  cat > "$D/BAtssOnly.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BAtssOnly extends BComponent {
  private Clock.Ticket pollTicket;
  @Override public void atSteadyState() throws Exception {
    super.atSteadyState();
    pollTicket = Clock.schedulePeriodically(this, BRelTime.makeSeconds(60), doPoll, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (pollTicket != null) { pollTicket.cancel(); pollTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"atSteadyState-only-timer"* ]]
}

@test "ATSS2: same class WITH a started() override does NOT emit atSteadyState-only-timer (clean)" {
  D="$BATS_TEST_TMPDIR/atss2"; mkdir -p "$D"
  cat > "$D/BAtssFixed.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BAtssFixed extends BComponent {
  private Clock.Ticket pollTicket;
  @Override public void started() throws Exception {
    super.started();
    if (Sys.atSteadyState()) {
      pollTicket = Clock.schedulePeriodically(this, BRelTime.makeSeconds(60), doPoll, null);
    }
  }
  @Override public void atSteadyState() throws Exception {
    super.atSteadyState();
    pollTicket = Clock.schedulePeriodically(this, BRelTime.makeSeconds(60), doPoll, null);
  }
  public void stopped() throws Exception {
    super.stopped();
    if (pollTicket != null) { pollTicket.cancel(); pollTicket = null; }
  }
}
JAVA
  run "$LINT" "$D"
  [ "$status" -eq 0 ]
  [[ "$output" != *"atSteadyState-only-timer"* ]]
}

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

# ===========================================================================
# audit-2026-10-03 A1 — timer-ticket is checked PER TICKET FIELD on comment- and string-stripped code.
# The old check passed when the token `cancel` appeared anywhere in stopped(): in a comment, in a string, or
# cancelling one ticket of several. A stopped() that leaves one ticket armed leaks that timer on station stop.
# [ev: corpus B787]
# ===========================================================================

_tt() { # _tt <Class> <body-lines...> — a class with fields a, b (Clock.Ticket) armed in arm()
  local c="$1"; shift
  {
    printf 'package demo;\nimport javax.baja.sys.*;\npublic final class %s extends BComponent {\n' "$c"
    printf '  private Clock.Ticket a;\n  private Clock.Ticket b;\n'
    printf '  public void arm() { a = Clock.schedule(this, BRelTime.makeSeconds(5), x, null); b = Clock.schedule(this, BRelTime.makeSeconds(9), y, null); }\n'
    printf '%s\n' "$@"
    printf '}\n'
  } > "$SRC/$c.java"
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java" "$SRC/NoTimer.java"
}

@test "TT-comment: 'cancel' only in a comment inside stopped() FAILs (A1)" {
  # Mutation: TT-comment -- matching the raw line (comments kept) lets a commented 'cancel' pass.
  _tt C1 '  public void stopped() throws Exception { super.stopped(); // TODO a.cancel(); b.cancel();' '  }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  timer-ticket"*"C1.java"*"a"* ]]
}

@test "TT-partial: stopped() cancelling ONE of two tickets FAILs naming the other (A1)" {
  # Mutation: TT-partial -- one cancel call for any field passes the whole class.
  _tt C2 '  public void stopped() throws Exception { super.stopped(); if (a != null) a.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  timer-ticket"*"C2.java"*"ticket b"* ]]
  if [[ "$output" == *"ticket a"* ]]; then return 1; fi
}

@test "TT-string: 'cancel' inside a string literal in stopped() FAILs (A1)" {
  # Mutation: TT-string -- not blanking string literals lets log(\"a.cancel()\") pass.
  _tt C3 '  public void stopped() throws Exception { super.stopped(); log("a.cancel(); b.cancel();"); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
}

@test "TT-helper: stopped() -> cancelAll() -> cancelA()/cancelB() cancelling every ticket PASSes (A1)" {
  # Mutation: TT-helper -- without the callee bodies the stopped() -> cancelAll() -> cancelX() shape false-FAILs.
  _tt C4 '  public void stopped() throws Exception { super.stopped(); cancelAll(); }' \
         '  private void cancelAll() { cancelA(); if (b != null) b.cancel(); }' \
         '  private void cancelA() { if (a != null) { a.cancel(); a = null; } }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  timer-ticket"*"C4.java"* ]]
}

@test "TT-both: stopped() cancelling both tickets PASSes; a helper taking the ticket counts (A1)" {
  _tt C5 '  public void stopped() throws Exception { super.stopped(); a.cancel(); cancelTicket(b); }' \
         '  private static void cancelTicket(Clock.Ticket t) { if (t != null) t.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

@test "TT-array: a Clock.Ticket[] field cancelled element-wise in stopped() PASSes (A1)" {
  cat > "$SRC/C6.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class C6 extends BComponent {
  private final Clock.Ticket[] delays = new Clock.Ticket[4];
  public void arm(int i) { delays[i] = Clock.schedule(this, BRelTime.makeSeconds(5), x, null); }
  public void stopped() throws Exception {
    super.stopped();
    for (int i = 0; i < delays.length; i++) { if (delays[i] != null) delays[i].cancel(); }
  }
}
JAVA
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java" "$SRC/NoTimer.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

@test "TT-finderr: an unreadable sub-directory is an env error (exit 3), never a clean pass (A1)" {
  # Mutation: TT-finderr -- ignoring the find status skips the locked directory's timers and exits 0.
  rm -f "$SRC/Owner.java"
  mkdir -p "$SRC/locked"
  cp "$SRC/Conformant.java" "$SRC/locked/"
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root"; fi
  run "$LINT" "$SRC"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}

@test "TT-overaccept: a non-cancel call taking the field (log(a)) beside another ticket's cancel FAILs naming it (A1b)" {
  # Mutation: TT-overaccept -- accepting ANY call with the field as an argument lets log(a) + b.cancel() pass a.
  _tt C8 '  public void stopped() throws Exception { super.stopped(); log(a); b.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  timer-ticket"*"C8.java"*"ticket a"* ]]
}

@test "TT-boundary: a type merely ending in Ticket (MyTicket note) is not a Clock ticket field (A1b)" {
  # Mutation: TT-boundary -- an unanchored Ticket match reads 'MyTicket note' as a ticket field and false-FAILs.
  _tt C9 '  private MyTicket note;' '  public void stopped() throws Exception { super.stopped(); a.cancel(); b.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  if [[ "$output" == *"note"* ]]; then return 1; fi
}

@test "TT-query: a query named like cancel (isCancelled(a)) does not count as cancelling a (A1c)" {
  # Mutation: TT-query -- a name merely CONTAINING cancel accepts isCancelled(a) as a cancel.
  _tt C10 '  public void stopped() throws Exception { super.stopped(); if (isCancelled(a)) return; b.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  timer-ticket"*"C10.java"*"ticket a"* ]]
}

@test "TT-helper-only: a cancel helper taking the field is the only cancel in scope and still PASSes (A1c)" {
  _tt C11 '  public void stopped() throws Exception { super.stopped(); cancelTicket(a); cancelTicket(b); }' \
          '  private static void cancelTicket(Clock.Ticket t) { if (t != null) t.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

# ===========================================================================
# audit-2026-10-03 A8 — lifecycle WARNs (advisory: exit unchanged).
# orphan-flag: a method other than stopped()/started() cancels ticket T, but leaves the companion flag F that was
#   set true beside `T = Clock.schedule*` — the expiry handler that would clear F never runs, so F stays true.
#   [ev: corpus B801] [ev: corpus B812] (companion-flag doctrine, extended to the cancel path)
# release-gate-parity: started()/atSteadyState() re-applies outputs (apply*()) behind a begin*() gate, and a
#   release point (exit*/end*/finish*/leave*/on*Exit|Expired|End) re-applies the same apply*() without that gate.
#   [ev: retro continuous-fan-post-defrost-delay Δ2] [ev: retro panccadia-restart-seq-comp-lockout-hours Δ8]
#   (types/logic.md § release-point gate rule 2)
# ===========================================================================

_lc() { # _lc <Class> <body-lines...>
  local c="$1"; shift
  { printf 'package demo;\nimport javax.baja.sys.*;\npublic final class %s extends BComponent {\n' "$c"; printf '%s\n' "$@"; printf '}\n'; } > "$SRC/$c.java"
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java" "$SRC/NoTimer.java"
}

_ORPH_HEAD=(
'  private Clock.Ticket startTicket;'
'  private boolean startingUp;'
'  public void arm() { startingUp = true; startTicket = Clock.schedule(this, BRelTime.makeSeconds(5), done, null); }'
'  public void doDone() { startingUp = false; }'
'  public void started() throws Exception { super.started(); startingUp = false; }'
'  public void stopped() throws Exception { super.stopped(); if (startTicket != null) startTicket.cancel(); startingUp = false; }'
)

@test "LC-orphan: a disable path that cancels the ticket but leaves its companion flag true WARNs orphan-flag (A8)" {
  # Mutation: LC-orphan -- dropping the orphan-flag pass lets a cancelled ticket leave startingUp stuck true.
  _lc O1 "${_ORPH_HEAD[@]}" '  public void disable() { if (startTicket != null) startTicket.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  orphan-flag"*"O1.java"*"disable()"*"startTicket"*"startingUp"* ]]
}

@test "LC-orphan-ok: the same disable path clearing the flag (directly or in a callee) does not WARN (A8)" {
  # Mutation: LC-orphan-ok -- ignoring the callee scope false-WARNs disable() -> resetStart().
  _lc O2 "${_ORPH_HEAD[@]}" '  public void disable() { if (startTicket != null) startTicket.cancel(); clearStart(); }' \
         '  private void clearStart() { startingUp = false; }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  if [[ "$output" == *"orphan-flag"* ]]; then return 1; fi
}

@test "LC-orphan-rearm: cancelling before re-arming in the arming method itself does not WARN (A8)" {
  _lc O3 '  private Clock.Ticket startTicket;' '  private boolean startingUp;' \
    '  public void arm() { if (startTicket != null) startTicket.cancel(); startingUp = true; startTicket = Clock.schedule(this, BRelTime.makeSeconds(5), done, null); }' \
    '  public void doDone() { startingUp = false; }' \
    '  public void stopped() throws Exception { super.stopped(); if (startTicket != null) startTicket.cancel(); startingUp = false; }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]   # WARN-only: a negative LC case must also exit 0 (#246 A8 R3-neg-tests-no-status)
  if [[ "$output" == *"orphan-flag"* ]]; then return 1; fi
}

_GATE_HEAD=(
'  private boolean restartSequencing;'
'  public void started() throws Exception { super.started(); beginRestartSequencing(); applyRunCmd(); applyFanRunMode(); }'
'  private void beginRestartSequencing() { restartSequencing = true; }'
'  private void applyRunCmd() { }'
'  private void applyFanRunMode() { }'
)

@test "LC-gate: a release point re-applying gated outputs without the gate WARNs release-gate-parity (A8)" {
  # Mutation: LC-gate -- dropping the parity pass lets exitDefrost() re-assert outputs ungated (the 2.2.x defect shape).
  _lc G1 "${_GATE_HEAD[@]}" '  public void exitDefrost() { applyRunCmd(); applyFanRunMode(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  release-gate-parity"*"G1.java"*"exitDefrost()"*"beginRestartSequencing"* ]]
}

@test "LC-gate-ok: the release point calling the gate first does not WARN (A8)" {
  # Mutation: LC-gate-ok -- not looking for the gate call in the release point false-WARNs the fixed shape.
  _lc G2 "${_GATE_HEAD[@]}" '  public void exitDefrost() { beginRestartSequencing(); applyRunCmd(); applyFanRunMode(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]   # WARN-only: a negative LC case must also exit 0 (#246 A8 R3-neg-tests-no-status)
  if [[ "$output" == *"release-gate-parity"* ]]; then return 1; fi
}

@test "LC-gate-none: without a begin*() gate in started() no release point is judged (A8)" {
  _lc G3 '  public void started() throws Exception { super.started(); applyRunCmd(); }' '  private void applyRunCmd() { }' \
         '  public void exitDefrost() { applyRunCmd(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]   # WARN-only: a negative LC case must also exit 0 (#246 A8 R3-neg-tests-no-status)
  if [[ "$output" == *"release-gate-parity"* ]]; then return 1; fi
}

@test "LC-orphan-heal: a class-wide self-heal (if (startTicket == null && startingUp) startingUp = false) releases the orphan (A8)" {
  _lc O4 "${_ORPH_HEAD[@]}" '  public void disable() { if (startTicket != null) startTicket.cancel(); startTicket = null; }' \
         '  public void execute() { if (startTicket == null && startingUp) startingUp = false; }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]   # WARN-only: a negative LC case must also exit 0 (#246 A8 R3-neg-tests-no-status)
  if [[ "$output" == *"orphan-flag"* ]]; then return 1; fi
}

@test "LC-orphan-direct: only the method that cancels the ticket directly reports, not each of its callers (A8)" {
  _lc O5 "${_ORPH_HEAD[@]}" '  private void cancelStart() { if (startTicket != null) startTicket.cancel(); }' \
         '  public void disable() { cancelStart(); }' '  public void reset() { cancelStart(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]   # WARN-only: a negative LC case must also exit 0 (#246 A8 R3-neg-tests-no-status)
  [ "$(grep -c 'orphan-flag' <<< "$output")" -eq 1 ]
  [[ "$output" == *"cancelStart()"* ]]
}

@test "LC-orphan-callers: a cancel helper whose every caller clears the flag does not WARN (A8)" {
  # Mutation: LC-orphan-callers -- ignoring the callers reports a helper whose callers own the flag.
  _lc O6 "${_ORPH_HEAD[@]}" '  private void cancelStart() { if (startTicket != null) startTicket.cancel(); }' \
         '  public void disable() { cancelStart(); startingUp = false; }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]   # WARN-only: a negative LC case must also exit 0 (#246 A8 R3-neg-tests-no-status)
  if [[ "$output" == *"orphan-flag"* ]]; then return 1; fi
}

@test "TT-safecancel: safeCancel(a) / doCancel(b) helpers in stopped() PASS; isCancelled/wasCancelled do not (A8b)" {
  # Mutation: TT-safecancel -- requiring the name to START with cancel false-FAILs safeCancel(a)/doCancel(b).
  _tt C12 '  public void stopped() throws Exception { super.stopped(); safeCancel(a); doCancel(b); }' \
          '  private static void safeCancel(Clock.Ticket t) { if (t != null) t.cancel(); }' \
          '  private static void doCancel(Clock.Ticket t) { if (t != null) t.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  _tt C13 '  public void stopped() throws Exception { super.stopped(); if (wasCancelled(a)) return; b.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ticket a"* ]]
}

@test "TT-helper-body: a helper counts only when its same-file BODY cancels — logCancelRequest(a)/cancelled(a) do not (A8c)" {
  # Mutation: TT-helper-body -- accepting a helper by its name again lets logCancelRequest(a) pass a.
  _tt C14 '  public void stopped() throws Exception { super.stopped(); logCancelRequest(a); cancelled(b); }' \
          '  private void logCancelRequest(Clock.Ticket t) { log("cancel requested"); }' \
          '  private boolean cancelled(Clock.Ticket t) { return t == null; }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ticket a"* ]]
  [[ "$output" == *"ticket b"* ]]
}

@test "TT-helper-any-name: a same-file helper whose body cancels its argument counts whatever its name (A8c)" {
  _tt C15 '  public void stopped() throws Exception { super.stopped(); stopTimer(a); stopTimer(b); }' \
          '  private static void stopTimer(Clock.Ticket t) { if (t != null) t.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

@test "TT-helper-nested: a cancelling helper called inside a condition (if (stopTimer(a)) …) still counts (A8c)" {
  # Mutation: TT-helper-nested -- a separator class that admits "(" binds the call to the enclosing 'if' and misses it.
  _tt C16 '  public void stopped() throws Exception { super.stopped(); if (stopTimer(a)) { } if (stopTimer(b)) { } }' \
          '  private static boolean stopTimer(Clock.Ticket t) { if (t == null) return false; t.cancel(); return true; }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

@test "TT-helper-param: a helper counts only when its body cancels its PARAMETER, not another ticket (A8d)" {
  # Mutation: TT-helper-param -- any .cancel( in the helper body counts, so stopTimer(a) { other.cancel(); } passes a.
  _tt C17 '  private Clock.Ticket other;' \
          '  public void stopped() throws Exception { super.stopped(); stopTimer(a); b.cancel(); }' \
          '  private void stopTimer(Clock.Ticket t) { if (other != null) other.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ticket a"* ]]
}

@test "TT-helper-this: this.cancelTicket(a) counts like cancelTicket(a) (A8d)" {
  # Mutation: TT-helper-this -- rejecting every qualified call false-FAILs this.cancelTicket(a).
  _tt C18 '  public void stopped() throws Exception { super.stopped(); this.cancelTicket(a); this.cancelTicket(b); }' \
          '  private void cancelTicket(Clock.Ticket t) { if (t != null) t.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

@test "TT-helper-sigline: a helper whose opening brace is on the next line still has its parameters read (A8d)" {
  # Mutation: TT-helper-sigline -- reading parameters only from the brace line false-FAILs the Allman-style helper.
  _tt C19 '  public void stopped() throws Exception { super.stopped(); safeCancel(a); safeCancel(b); }' \
          '  private static void safeCancel(Clock.Ticket ticket)' '  {' '    if (ticket != null) ticket.cancel();' '  }'
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
}

@test "TT-noname: a schedule with no named Ticket field FAILs with a readable row, no '(ticket)' placeholder (#246)" {
  # Mutation: TT-noname -- the placeholder rendered as "does not cancel ticket (ticket)".
  {
    printf 'package demo;\nimport javax.baja.sys.*;\npublic final class C20 extends BComponent {\n'
    printf '  public void arm() { Clock.schedule(this, BRelTime.makeSeconds(5), x, null); }\n'
    printf '  public void stopped() throws Exception { super.stopped(); }\n}\n'
  } > "$SRC/C20.java"
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java" "$SRC/NoTimer.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  timer-ticket"*"C20.java:4  stopped() cancels no ticket"* ]]
  if [[ "$output" == *"(ticket)"* ]]; then return 1; fi
}

@test "TT-helper-other: other.cancelTicket(a) is another object's method, not this file's helper -> FAIL (#246 A8d)" {
  # Mutation: TT-helper-other -- accepting any qualified call as this file's helper passes a ticket nobody cancels.
  _tt C21 '  public void stopped() throws Exception { super.stopped(); other.cancelTicket(a); this.cancelTicket(b); }' \
          '  private void cancelTicket(Clock.Ticket t) { if (t != null) t.cancel(); }'
  run "$LINT" "$SRC"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  timer-ticket"*"C21.java"*"ticket a"* ]]
  if [[ "$output" == *"ticket b"* ]]; then return 1; fi
}

# --- Truth fixes 2026-10-23 (R8-D14): the port reads code, and every row carries its line ---
# Seeds: truth audit 2026-10-22 T1 (a schedule only in a comment or a string false-FAILed; rows had no line).

@test "TR1-code: a Clock.schedule only in a comment or a string owns no timer (no row, exit 0)" {
  # Mutation: TR1-code -- scanning the raw lines (comments and strings kept) FAILs a schedule that is only mentioned
  {
    printf 'package demo;\nimport javax.baja.sys.*;\npublic final class BNote extends BComponent {\n'
    printf '  public void arm() {\n    // t = Clock.schedule(this, BRelTime.makeSeconds(5), armed, null);\n'
    printf '    log("Clock.schedule(this, BRelTime.makeSeconds(5), armed, null); Clock.Ticket t;");\n  }\n}\n'
  } > "$SRC/BNote.java"
  rm -f "$SRC/Owner.java" "$SRC/Conformant.java" "$SRC/NoTimer.java"
  run "$LINT" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "TR1-shape: every row is <sev>  <check>  <file>:<line>  <msg>, the line pointing at the finding" {
  # Mutation: TR1-shape -- leaving Line 0 prints the file-level "<file>: <msg>" row with no line
  mkdir -p "$BATS_TEST_TMPDIR/m-rt/src"
  printf '<permissions/>\n' > "$BATS_TEST_TMPDIR/m-rt/module-permissions.xml"
  {
    printf 'package demo;\nimport javax.baja.sys.*;\npublic final class BLeak extends BComponent {\n'
    printf '  private Clock.Ticket ticket;\n  private boolean busy;\n  public void kick() {\n'
    printf '    Clock.schedule(this, BRelTime.makeSeconds(1), kicked, null);\n    busy = true;\n  }\n'
    printf '  private ExecutorService ex = Executors.newSingleThreadExecutor();\n'
    printf '  public void stopped() { ex.shutdownNow(); }\n}\n'
  } > "$BATS_TEST_TMPDIR/m-rt/src/BLeak.java"
  run "$LINT" "$BATS_TEST_TMPDIR/m-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  discarded-ticket  $BATS_TEST_TMPDIR/m-rt/src/BLeak.java:7  "* ]]
  [[ "$output" == *"FAIL  timer-ticket  $BATS_TEST_TMPDIR/m-rt/src/BLeak.java:4  "* ]]
  [[ "$output" == *"FAIL  companion-flag  $BATS_TEST_TMPDIR/m-rt/src/BLeak.java:8  "* ]]
  # the permissions file is named by its module directory, never a second absolute path
  [[ "$output" == *"FAIL  jdk-thread  $BATS_TEST_TMPDIR/m-rt/src/BLeak.java:11  BLeak modifies a thread (shutdownNow()) but m-rt/module-permissions.xml requests no MANAGE_EXECUTION"* ]]
  # no row is file-level
  if printf '%s\n' "$output" | grep -qE '\.java: '; then return 1; fi
}
