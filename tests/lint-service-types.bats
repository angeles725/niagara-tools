#!/usr/bin/env bats
# lint-service-types.bats — a service whose getServiceTypes() returns no type (empty array or null) is never
# registered; one that does not return its own TYPE is a review row. [ev: corpus B1419 §1419.1]
# [ev: retro course-recordings Δ8]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LST="$KIT/toolbelt/lint-service-types.sh"
  SRC="$TMPDIR_T/Mod/src/com/x"
  mkdir -p "$SRC"
}
teardown() { rm -rf "$TMPDIR_T"; }

# svc <class> <getServiceTypes body line> [extra member line] — a minimal service source file
svc() {
  printf 'package com.x;\npublic class %s extends BAbstractService {\n' "$1"
  if [ -n "${3:-}" ]; then printf '  %s\n' "$3"; fi
  printf '  @Override\n  public Type[] getServiceTypes() {\n    %s\n  }\n}\n' "$2"
}

@test "ST1: an empty array (new Type[0] / new Type[] {}) or null FAILs and exits 1 without --strict" {
  # Mutation: ST1 -- dropping the empty-array check passes the IntelliJ 'Implement methods' stub new Type[0]
  svc BZero 'return new Type[0];' > "$SRC/BZero.java"
  svc BBraces 'return new Type[] {};' > "$SRC/BBraces.java"
  svc BNull 'return null;' > "$SRC/BNull.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-service-types  "*"BZero.java:4  getServiceTypes() registers no type"* ]]
  [[ "$output" == *"BBraces.java:4  getServiceTypes() registers no type"* ]]
  [[ "$output" == *"BNull.java:4  getServiceTypes() registers no type"* ]]
}

@test "ST2: an array holding TYPE (one line or several) is clean" {
  # Mutation: ST2 -- requiring the array on the return line flags a multi-line initializer that holds TYPE
  svc BOne 'return new Type[] { TYPE };' > "$SRC/BOne.java"
  { printf 'package com.x;\npublic class BMulti extends BAbstractService {\n'
    printf '  public Type[] getServiceTypes() {\n    return new Type[] {\n      TYPE,\n      BIMulti.TYPE\n    };\n  }\n}\n'; } > "$SRC/BMulti.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ST3: a returned same-file field is resolved: { TYPE } is clean, new Type[0] FAILs" {
  # Mutation: ST3 -- not resolving a returned field passes a serviceTypes field initialized to new Type[0]
  svc BField 'return serviceTypes;' 'private static final Type[] serviceTypes = { TYPE };' > "$SRC/BField.java"
  svc BFieldEmpty 'return serviceTypes;' 'private static final Type[] serviceTypes = new Type[0];' > "$SRC/BFieldEmpty.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BFieldEmpty.java:5  getServiceTypes() registers no type"* ]]
  if [[ "$output" == *"BField.java"* ]]; then return 1; fi
}

@test "ST4: a non-empty array without the service's own TYPE WARNs (exit 0); Own.TYPE counts as own" {
  # Mutation: ST4 -- accepting any X.TYPE as the own type passes a service registered only under another type
  svc BOther 'return new Type[] { BIOther.TYPE };' > "$SRC/BOther.java"
  svc BQual 'return new Type[] { BQual.TYPE, BIQual.TYPE };' > "$SRC/BQual.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-service-types  "*"BOther.java:4  getServiceTypes() does not return the service's own TYPE"* ]]
  if [[ "$output" == *"BQual.java"* ]]; then return 1; fi
}

@test "ST5: a return the lint cannot resolve (super call, inherited field) WARNs, never passes silently" {
  # Mutation: ST5 -- treating an unresolved return as clean passes return super.getServiceTypes()
  svc BSuper 'return super.getServiceTypes();' > "$SRC/BSuper.java"
  svc BInh 'return INHERITED;' > "$SRC/BInh.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-service-types  "*"BSuper.java:4  getServiceTypes() return not resolved in this file"* ]]
  [[ "$output" == *"BInh.java:4  getServiceTypes() return not resolved in this file"* ]]
}

@test "ST6: a declaration without a body, a file without the method and a commented-out stub give no row" {
  # Mutation: ST6 -- matching inside comments flags a commented-out return new Type[0]
  printf 'package com.x;\npublic interface IFoo {\n  Type[] getServiceTypes();\n}\n' > "$SRC/IFoo.java"
  printf 'package com.x;\npublic class BPlain extends BComponent {\n}\n' > "$SRC/BPlain.java"
  { printf 'package com.x;\npublic class BCom extends BAbstractService {\n'
    printf '  public Type[] getServiceTypes() {\n    // return new Type[0];\n    return new Type[] { TYPE };\n  }\n}\n'; } > "$SRC/BCom.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ST8: vendor forms are own: this.getType(), a parenthesized array, an Own.field reference" {
  # Mutation: ST8 -- dropping getType() from the own-type set WARNs the vendor form new Type[] { this.getType() }
  svc BGet 'return new Type[] { this.getType() };' > "$SRC/BGet.java"
  svc BPar 'return (new Type[] { TYPE });' > "$SRC/BPar.java"
  svc BRef 'return BRef.TYPES;' 'public static final Type[] TYPES = { TYPE };' > "$SRC/BRef.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ST9: a field assigned later (static block, Own.field) or cloned is resolved through its assignments" {
  # Mutation: ST9 -- resolving only the declaration line misses a field assigned in a static block to new Type[0]
  { printf 'package com.x;\npublic class BLate extends BAbstractService {\n  private static Type[] serviceTypes;\n'
    printf '  static { BLate.serviceTypes = new Type[0]; }\n'
    printf '  public Type[] getServiceTypes() {\n    return BLate.serviceTypes;\n  }\n}\n'; } > "$SRC/BLate.java"
  svc BClone 'return (Type[]) types.clone();' 'private static final Type[] types = { TYPE };' > "$SRC/BClone.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BLate.java:5  getServiceTypes() registers no type"* ]]
  if [[ "$output" == *"BClone.java"* ]]; then return 1; fi
}

@test "ST10: every return is checked: a later null or empty return FAILs, a later foreign-only array WARNs" {
  # Mutation: ST10 -- checking only the first return passes a getServiceTypes() whose fallback branch returns null
  { printf 'package com.x;\npublic class BTwo extends BAbstractService {\n'
    printf '  public Type[] getServiceTypes() {\n    if (ready) return new Type[] { TYPE };\n    return null;\n  }\n}\n'; } > "$SRC/BTwo.java"
  { printf 'package com.x;\npublic class BMix extends BAbstractService {\n'
    printf '  public Type[] getServiceTypes() {\n    if (ready) return new Type[] { TYPE };\n    return new Type[] { BIMix.TYPE };\n  }\n}\n'; } > "$SRC/BMix.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-service-types  "*"BTwo.java:3  getServiceTypes() registers no type"* ]]
  [[ "$output" == *"WARN  lint-service-types  "*"BMix.java:3  getServiceTypes() does not return the service's own TYPE"* ]]
}

@test "ST11: an assignment after another statement on the same line is read from the assignment, not the line start" {
  # Mutation: ST11 -- reading the initializer from the line start stops at an earlier ';' and misses new Type[0]
  { printf 'package com.x;\npublic class BLine extends BAbstractService {\n  private static Type[] types;\n'
    printf '  static { int k = 0; types = new Type[0]; }\n'
    printf '  public Type[] getServiceTypes() {\n    return types;\n  }\n}\n'; } > "$SRC/BLine.java"
  run "$LST" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BLine.java:5  getServiceTypes() registers no type"* ]]
}

@test "ST7: --strict exits 1 on a WARN; usage errors and a non-directory exit 3" {
  svc BOther 'return new Type[] { BIOther.TYPE };' > "$SRC/BOther.java"
  run "$LST" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  run "$LST"
  [ "$status" -eq 3 ]
  run "$LST" "$TMPDIR_T/nope"
  [ "$status" -eq 3 ]
}

@test "ST-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: ST-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
  printf 'x\n' > "$SRC/U.java"
  chmod 000 "$SRC/U.java"
  if [ -r "$SRC/U.java" ]; then chmod 644 "$SRC/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$LST" "$TMPDIR_T/Mod"
  chmod 644 "$SRC/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "ST-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: ST-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
  mkdir -p "$SRC/locked"
  printf 'class B {}\n' > "$SRC/locked/B.java"
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$LST" "$TMPDIR_T/Mod"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/locked"* ]]
}
