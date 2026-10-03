#!/usr/bin/env bats
# lint-license-isoperational-gate.bats — a licensed class whose changed()/action/servlet-write callback acts with
# no isOperational()/isFault() gate (deferred-lints D3). [ev: retro secure-authoring-isoperational-gate Δ1]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LIG="$KIT/toolbelt/lint-license-isoperational-gate.sh"
  SRC="$TMPDIR_T/Mod/src/com/x"
  mkdir -p "$SRC"
}
teardown() { rm -rf "$TMPDIR_T"; }

licensed_head() {
  printf 'package com.x;\npublic class %s extends BAbstractService {\n' "$1"
  printf '  @Override\n  public final Feature getLicenseFeature() {\n'
  printf '    return Sys.getLicenseManager().getFeature("vendor", "feature");\n  }\n'
}

@test "LIG1: an ungated changed() of a licensed class WARNs; a callback that returns on !isOperational() does not" {
  # Mutation: LIG1 -- dropping the gate check WARNs a callback that returns early on !isOperational()
  { licensed_head BSvc
    printf '  @Override\n  public void changed(Property p, Context cx) {\n    super.changed(p, cx);\n    recompute();\n  }\n'
    printf '  public void doTick() {\n    if (!isOperational()) return;\n    recompute();\n  }\n}\n'; } > "$SRC/BSvc.java"
  run "$LIG" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-license-isoperational-gate  "*"BSvc.java:8  licensed class: changed() acts with no isOperational()/isFault() gate"* ]]
  if [[ "$output" == *"doTick"* ]]; then return 1; fi
}

@test "LIG2: getLicenseFeature() returning null (the BAbstractService default) is not licensed: no row" {
  # Mutation: LIG2 -- treating `return null;` as a license feature WARNs an unlicensed service
  { printf 'package com.x;\npublic class BFree extends BAbstractService {\n'
    printf '  public Feature getLicenseFeature() { return null; }\n'
    printf '  public void changed(Property p, Context cx) {\n    recompute();\n  }\n}\n'; } > "$SRC/BFree.java"
  run "$LIG" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LIG3: a callback that only delegates to super (or returns) does not act: no row" {
  # Mutation: LIG3 -- counting a super.changed() call as acting WARNs a callback that only delegates
  { licensed_head BDel
    printf '  public void changed(Property p, Context cx) {\n    super.changed(p, cx);\n  }\n'
    printf '  public void doNothing() {\n    return;\n  }\n}\n'; } > "$SRC/BDel.java"
  run "$LIG" "$TMPDIR_T/Mod"
  [ -z "$output" ]
}

@test "LIG4: a servlet write handler (doPost) WARNs; a read handler (doGet) does not" {
  { licensed_head BWeb
    printf '  public void doGet(WebOp op) {\n    render(op);\n  }\n'
    printf '  public void doPost(WebOp op) {\n    write(op);\n  }\n}\n'; } > "$SRC/BWeb.java"
  run "$LIG" "$TMPDIR_T/Mod"
  [[ "$output" == *"BWeb.java:10  licensed class: doPost() acts"* ]]
  if [[ "$output" == *"doGet"* ]]; then return 1; fi
}

@test "LIG5: no getLicenseFeature() body (none, or an abstract/interface declaration) is out of scope: no row" {
  { printf 'package com.x;\npublic class BPlain extends BComponent {\n'
    printf '  public void changed(Property p, Context cx) {\n    recompute();\n  }\n}\n'; } > "$SRC/BPlain.java"
  { printf 'package com.x;\npublic interface ILic {\n  Feature getLicenseFeature();\n}\n'
    printf 'class Impl {\n  public void doRun() {\n    go();\n  }\n}\n'; } > "$SRC/ILic.java"
  run "$LIG" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LIG6: isFault() and isFatalFault() also count as the gate" {
  { licensed_head BAlt
    printf '  public void doA() {\n    if (isFault()) return;\n    a();\n  }\n'
    printf '  public void doB() {\n    if (isFatalFault()) return;\n    b();\n  }\n}\n'; } > "$SRC/BAlt.java"
  run "$LIG" "$TMPDIR_T/Mod"
  [ -z "$output" ]
}

@test "LIG7: --strict exits 1 on a WARN; usage errors exit 3" {
  { licensed_head BSvc; printf '  public void doPoll() {\n    poll();\n  }\n}\n'; } > "$SRC/BSvc.java"
  run "$LIG" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  run "$LIG" --bogus "$TMPDIR_T/Mod"
  [ "$status" -eq 3 ]
  run "$LIG"
  [ "$status" -eq 3 ]
  run "$LIG" "$TMPDIR_T/nope"
  [ "$status" -eq 3 ]
}

@test "LIG-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: LIG-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
  printf 'x\n' > "$SRC/U.java"
  chmod 000 "$SRC/U.java"
  if [ -r "$SRC/U.java" ]; then chmod 644 "$SRC/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$LIG" "$TMPDIR_T/Mod"
  chmod 644 "$SRC/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "LIG-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: LIG-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
  mkdir -p "$SRC/locked"
  printf 'class B {}\n' > "$SRC/locked/B.java"
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$LIG" "$TMPDIR_T/Mod"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/locked"* ]]
}
