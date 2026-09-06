#!/usr/bin/env bats
# C12 S1 (B832-G3) RED — the shared parser's Case-B @-stop is a FALSE NEGATIVE on a
# parameter-annotation continuation line. In toolbelt/lib/method-boundary.sh:84 the
# backward scan from a brace-only `{` stops at the first line starting with `@`
# (meant to skip a class/field annotation). But a method whose signature wraps with a
# PARAMETER annotation on the continuation —
#     void m(
#         @Ann x)
#     {
# — has that `@Ann x)` line reached first, so the scan stops and the method is never
# named; its body (field-flag + Clock.schedule, a trip, a do-body) is invisible to every
# consuming lint. Reproduced: lint-timers gives 0 companion-flag WITH the @-stop, 1
# WITHOUT. Absent from the ff1b659 client corpus, so C11 kept it defensive; C12 fixes it.
#
# RED on 66123a2 (v0.22.0): the param-annotation fixture is a real companion-flag hazard
# (field flag set beside Clock.schedule, never cleared in stopped()/started()) but the lint
# exits 0 — the FN. GREEN once the @-stop distinguishes a class/field annotation (stop) from
# a parameter-annotation continuation (do not stop), or is removed (0 regressions across the
# five parser suites when dropped). [ev: method-boundary.sh:84; B832-G3; probe 850791f12]

setup() { LT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/lint-timers.sh"; }

@test "C12-S1-atstop-paramann: a companion-flag hazard in a method whose signature has a PARAMETER annotation on a continuation line still FAILs (the @-stop must not swallow the method)" {
  D="$BATS_TEST_TMPDIR/s1"; mkdir -p "$D"
  cat > "$D/BParam.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BParam extends BComponent {
  private boolean startingUp = false;      // class FIELD — can stay stuck across stop/restart
  private Clock.Ticket t;
  void arm(
      @Nonnull Object x)
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

@test "C12-S1-classann-guard: a method preceded by a class/field-level @NiagaraProperty annotation is NOT misparsed — the @-stop's legitimate case stays clean (the fix must not regress BMisparse)" {
  D="$BATS_TEST_TMPDIR/s1g"; mkdir -p "$D"
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
  public void armA() { startingUp = true; }                                              // flag, method A
  public void armB() { t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null); }    // schedule, method B
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 0 ]
}

@test "C12-S1-singleline-ann-guard: a single-line leading annotation before a method (@Override public void arm() {) does NOT swallow the method — the @-stop's legitimate no-op stays a no-op" {
  D="$BATS_TEST_TMPDIR/s1sl"; mkdir -p "$D"
  cat > "$D/BSingle.java" <<'JAVA'
package demo;
import javax.baja.sys.*;
public final class BSingle extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket t;
  @Override public void arm() { startingUp = true; t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null); }
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"companion-flag"* ]]
}

@test "C12-S1-generics-paramann (B833-G2): a param-annotation continuation whose type carries generics (Map<String,Integer>) is still a FALSE NEGATIVE today and must FAIL — the paren-balance @-stop rule must count '(' only, not the '<>' of generics" {
  D="$BATS_TEST_TMPDIR/s1gen"; mkdir -p "$D"
  cat > "$D/BGen.java" <<'JAVA'
package demo;
import javax.baja.sys.*; import java.util.Map;
public final class BGen extends BComponent {
  private boolean startingUp = false;
  private Clock.Ticket t;
  void arm(
      @Nonnull Map<String,Integer> x)
  {
    startingUp = true;
    t = Clock.schedule(this, BRelTime.makeSeconds(5), exp, null);
  }
  public void stopped() throws Exception { super.stopped(); if (t != null) { t.cancel(); t = null; } }
}
JAVA
  run "$LT" "$D"
  [ "$status" -eq 1 ]
  [[ "$output" == *"companion-flag"* && "$output" == *"startingUp"* ]]
}
