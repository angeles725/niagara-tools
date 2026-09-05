#!/usr/bin/env bats
# RED-FIRST pins for verify-module.sh --src TWO new WARN sub-checks (campaign 8 PR4). Fixtures are
# generated in-test via helpers/n4-fixtures.bash (zero committed binaries), same as verify-module.bats.
# Real shapes: ColdRoomPan BEvaporatorUnit.java:535/561/583 (newProperty *Mode/OPERATOR numeric),
# BColdRoom.java:47 (BStatusNumeric setpoint), CompPan BCompressorControl.java:108-173/336-347
# (OPERATOR double limits, count-like demand/stagesOn), DashboardReader.java:75 (ord literal).
#
#   facets-req  WARN  a slot missing the facet its role requires:
#     - a *Mode numeric slot with no RANGE and no MIN+MAX
#     - a setpoint|*Temp|*Limit|*Threshold BStatusNumeric with no UNITS
#     - a count-like double (demand|stagesOn|*Count) with no PRECISION 0
#     - an OPERATOR numeric slot with no MIN or MAX
#     (a valid facet with MIN=0 must NOT warn — 0 is a real MIN, not a missing one.)
#   ord-literal  WARN  a Java "station:| / slot:/ literal, EXCEPT inside a @NiagaraProperty
#     defaultValue, under srcTest/**, or in a *OrdConstants* class carrying a comment.
#
# Row (verify-module grammar): `<status>  <check>  <jar>  <detail>`. These are WARN → exit stays 0
# on an otherwise-good jar.
#
# RED today: verify-module.sh has no facets-req / ord-literal check, so on these fixtures it emits
# NO such row → every WARN-expecting pin fails for the right reason (check absent). The good jar still
# PASSes its real checks, so the control pins anchor on a real PASS row (not on tool-absence).
#
# NAMED MUTATIONS (post-green):
#   - strip the facet from the good slot -> its facets-req WARN appears (proves the check keys on the facet).
#   - drop the *OrdConstants* exemption   -> the sanctioned holder (F6) starts WARNing.

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  VM="$KIT/toolbelt/verify-module.sh"
}
teardown() { rm -rf "$TMPDIR_T"; }

good_dir() {
  add_manifest "$1"; add_signature "$1"
  make_class_file "$1/com/x/A.class" 52; make_class_file "$1/com/x/B.class" 52
  make_module_xml "$1" 4.14 com.x.A
}
# mksrc <ClassName> : build a good jar + a --src mod with one Java file read from stdin.
mksrc() {
  good_dir "$TMPDIR_T/d"; make_jar "$TMPDIR_T/Foo-rt.jar" "$TMPDIR_T/d"
  make_module_include "$TMPDIR_T/mod/Foo-rt" com.x.A
  mkdir -p "$TMPDIR_T/mod/Foo-rt/src/com/x"
  cat > "$TMPDIR_T/mod/Foo-rt/src/com/x/$1.java"
}
runvm() { run "$VM" --src "$TMPDIR_T/mod" "$TMPDIR_T/Foo-rt.jar"; }

# ---- facets-req -----------------------------------------------------------
@test "F1: a *Mode OPERATOR numeric slot with no RANGE/MIN+MAX WARNs facets-req (naming the slot)" {
  mksrc BThing <<'JAVA'
package com.x;
public class BThing extends BComponent {
  // real shape BEvaporatorUnit:535 — OPERATOR numeric, facets arg is null
  public static final Property fanMode = newProperty(Flags.SUMMARY | Flags.OPERATOR, 0d, null);
}
JAVA
  runvm
  [[ "$output" == *"WARN  facets-req"* ]] && [[ "$output" == *"fanMode"* ]]
}

@test "F2: a setpoint BStatusNumeric with no UNITS WARNs facets-req" {
  mksrc BThing <<'JAVA'
package com.x;
@NiagaraProperty(name = "setpoint", defaultValue = "new BStatusNumeric(0d)", flags = Flags.SUMMARY)
public class BThing extends BComponent {}
JAVA
  runvm
  [[ "$output" == *"WARN  facets-req"* ]] && [[ "$output" == *"setpoint"* ]]
}

@test "F3: a count-like double slot (demand) with no PRECISION WARNs facets-req" {
  mksrc BThing <<'JAVA'
package com.x;
@NiagaraProperty(name = "demand", type = "double", defaultValue = "0d", flags = Flags.TRANSIENT | Flags.READONLY)
public class BThing extends BComponent {}
JAVA
  runvm
  [[ "$output" == *"WARN  facets-req"* ]] && [[ "$output" == *"demand"* ]]
}

@test "F4: a slot with a valid MIN=0 + MAX facet does NOT WARN facets-req (0 is a real MIN)" {
  mksrc BThing <<'JAVA'
package com.x;
public class BThing extends BComponent {
  // properly faceted: MIN=0 (valid) and MAX=3 — an enum-style ordinal range
  public static final Property fanMode = newProperty(Flags.SUMMARY | Flags.OPERATOR, 0d,
      BFacets.make(BFacets.MIN, BDouble.make(0d), BFacets.MAX, BDouble.make(3d)));
}
JAVA
  runvm
  [[ "$output" == *"PASS  bytecode"* ]]        # anchor: the tool actually ran on a good jar
  [[ "$output" != *"WARN  facets-req"* ]]      # and did not warn on the properly-faceted slot
}

# ---- ord-literal ----------------------------------------------------------
@test "F5: a plain Java station:|/slot:/ literal WARNs ord-literal" {
  mksrc BThing <<'JAVA'
package com.x;
public class BThing {
  static final String SVC = "station:|slot:/Services/DashboardService";
}
JAVA
  runvm
  [[ "$output" == *"WARN  ord-literal"* ]]
}

@test "F6: the same literal inside a *OrdConstants* class with a comment does NOT WARN (exempt; mutation target)" {
  mksrc FooOrdConstants <<'JAVA'
package com.x;
/** Sanctioned single home for the hardcoded service ORD; integrators edit here. */
public final class FooOrdConstants {
  public static final String SVC = "station:|slot:/Services/DashboardService"; // sanctioned
}
JAVA
  runvm
  [[ "$output" == *"PASS  bytecode"* ]]        # anchor: tool ran
  [[ "$output" != *"WARN  ord-literal"* ]]     # exempt holder is silent (drop the exemption -> this WARNs)
}

@test "F7: an ORD inside a @NiagaraProperty defaultValue does NOT WARN ord-literal (exempt)" {
  mksrc BThing <<'JAVA'
package com.x;
@NiagaraProperty(name = "svc", defaultValue = "station:|slot:/Services/DashboardService", flags = Flags.SUMMARY)
public class BThing extends BComponent {}
JAVA
  runvm
  [[ "$output" == *"PASS  bytecode"* ]]        # anchor: tool ran
  [[ "$output" != *"WARN  ord-literal"* ]]
}

# ---- F8: D9b dot-dir prune (bites the EXISTING facets check today) --------
# Design addendum D9b: verify-module --src prunes dot-directories. report-module keeps a previous-deploy
# snapshot at <artifact>/.deploy-baseline; without a prune, verify --src descends into it and flags the
# old deploy's code. This bites the EXISTING raw-facets check TODAY (it greps $pd/src recursively with no
# prune), so F8 is genuinely RED now. NAMED MUTATION (post-green): remove the dot-dir prune -> the raw
# facet under .deploy-baseline is flagged again -> F8 flips.
@test "F8: a raw-number facet under src/.deploy-baseline/ is NOT flagged (dot-dir pruned)" {
  good_dir "$TMPDIR_T/d"; make_jar "$TMPDIR_T/Foo-rt.jar" "$TMPDIR_T/d"
  make_module_include "$TMPDIR_T/mod/Foo-rt" com.x.A
  mkdir -p "$TMPDIR_T/mod/Foo-rt/src/.deploy-baseline/com/x"
  printf 'class Stale { Object f = BFacets.make(BFacets.MIN, -40); }\n' \
    > "$TMPDIR_T/mod/Foo-rt/src/.deploy-baseline/com/x/Stale.java"
  runvm
  [[ "$output" == *"PASS  bytecode"* ]]     # anchor: the tool ran on a good jar
  [[ "$output" != *"FAIL  facets"* ]]       # the raw facet lives under .deploy-baseline -> pruned, not flagged
}
