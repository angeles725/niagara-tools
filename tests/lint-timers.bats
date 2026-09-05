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
