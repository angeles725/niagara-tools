#!/usr/bin/env bats
# C12 S2 (B832-G4) RED — a companion-flag hazard inside an instance/static INITIALIZER
# block is a FALSE NEGATIVE. The shared parser (method-boundary.sh) names methods by a
# signature; an initializer `{ ... }` opens at depth >= 2 with no signature, so Case-B
# finds no identifier and the block is never entered. The `!in_m` gate then correctly
# keeps a nested `for`/`if` from being named a method (I3 stays right), but the initializer
# BODY itself — a field flag set beside Clock.schedule, never cleared in stopped()/started()
# — is invisible. That is the same stuck-flag hazard as in a method (the flag is set once at
# construction and never reset), so the lint must catch it.
#
# Fork resolved (per lead): the fix is NOT to name keyword blocks as methods (I3 stays) but
# to recognise a brace-only depth-2 block with no signature as an anonymous body <init>
# (<clinit> after `static`), scan it like a method body, and pair the flag+schedule inside
# it. RED on 66123a2 (v0.22.0): the initializer hazard exits 0; GREEN once <init> bodies are
# scanned (verified via a prototype: initializer FAILs, the for-in-method control stays clean,
# 0 regressions across the five parser suites). [ev: method-boundary.sh:74/:89; B832-G4; probe 850791f12]

load lib/client-root   # C11 T2: blessed client read root (for the real-tree pin); env override wins

setup() { LT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/lint-timers.sh"; }

@test "C12-S2-initializer-hazard: a FIELD flag set true beside Clock.schedule inside an instance initializer FAILs companion-flag (the <init> body must be scanned)" {
  D="$BATS_TEST_TMPDIR/s2"; mkdir -p "$D"
  cat > "$D/BInit.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BInit extends BComponent {
  private boolean startingUp = false;      // class FIELD — set at construction, never cleared
  private Clock.Ticket t;
  {
    startingUp = true;
    t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null);
  }
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"companion-flag"* ]] && [[ "$output" == *"startingUp"* ]]
}

@test "C12-S2-for-in-method-guard: a for(...) { ... } block inside a REAL method is NOT named a method (I3 keyword exclusion stays) -> no false companion-flag" {
  D="$BATS_TEST_TMPDIR/s2g"; mkdir -p "$D"
  cat > "$D/BForCtl.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BForCtl extends BComponent {
  private Clock.Ticket t;
  public void armDefrost() {
    for (int i = 0; i < 3; i++) { boolean anyNoHardware = true; if (anyNoHardware) {} }
    t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null);
  }
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 0 ]
}

@test "C12-S2-lambda-field-guard: a class-level lambda field initializer with the brace on its own line is NOT a real initializer -> its body is not scanned as <init> (no false companion-flag). Flips a naive fix that names every brace-only depth-2 block <init>." {
  D="$BATS_TEST_TMPDIR/s2lam"; mkdir -p "$D"
  cat > "$D/BLamField.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BLamField extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket t;
  private Runnable r = () ->
  {
    startingUp = true;
    t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null);
  };
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 0 ]
}

@test "C12-S2-realtree: lint-timers on the blessed ColdRoomPan-rt tree stays clean (0 <init> false positives on the real corpus — B833: 0 real initializers at ff1b659)" {
  CRP="$C9_CLIENT_ROOT/Paccadia/ColdRoomPan/ColdRoomPan-rt/src"
  [ -d "$CRP" ] || skip "client ColdRoomPan tree not on this machine"
  run "$LT" "$CRP"
  [ "$status" -eq 0 ]
}
