#!/usr/bin/env bats
# lint-property-accessors.bats — a B* class that declares a slot by hand (`public static final Property x =
# newProperty(...)`) needs a public get<X>/is<X> and a public set<X>, or the station cannot load the type. The rule's
# golden cases live in go/internal/lint/rules/propertyaccessors. [ev: issue #1159]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-property-accessors.sh"
  SRC="$BATS_TEST_TMPDIR/src"
  mkdir -p "$SRC/com/acme"
}

_row() { # <line> <property> <what is missing>
  printf 'FAIL  lint-property-accessors  %s:%s  property %s of BExt: %s: the station cannot load the type (sys.schema "No setter for property (%s)" / "Cannot load type for BExt", NoClassDefFoundError on every use) (types/structure.md § Property accessors)' \
    "$SRC/com/acme/BExt.java" "$1" "$2" "$3" "$2"
}

@test "PA-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-property-accessors.sh <src-root>" ]
}

@test "PA-fixture: the MinimalPan -rt sources are clean" {
  run "$L" "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PA-clean: a class with a getter and a public setter per slot (is<X> for a boolean) is clean" {
  printf '%s\n' 'package com.acme;' 'public class BExt extends BComponent {' \
    '  public static final Property a = newProperty(0, 0, null);' \
    '  public int getA() { return 0; }' '  public void setA(int v) { }' \
    '  public static final Property on = newProperty(0, false, null);' \
    '  public boolean isOn() { return true; }' '  public void setOn(boolean v) { }' '}' > "$SRC/com/acme/BExt.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PA1: a slot with a getter and no setter FAILs (the issue #1159 type); a class that is not B* is ignored" {
  # Mutation: PA1 -- not requiring a setter passes a slot with a getter and no set<X> (the issue #1159 type)
  printf '%s\n' 'package com.acme;' 'public class BExt extends BPointExtension {' \
    '  public static final Property x = newProperty(0, 0, null);' \
    '  public int getX() { return 0; }' '}' > "$SRC/com/acme/BExt.java"
  printf '%s\n' 'package com.acme;' 'public class Helper {' \
    '  public static final Property y = newProperty(0, 0, null);' '}' > "$SRC/com/acme/Helper.java"
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$(_row 3 x 'no public setX(...)')" ]
}

@test "PA2: a private setter and a missing getter FAIL" {
  # Mutation: PA2 -- accepting a non-public setter (or no getter) passes a slot the station still cannot load
  printf '%s\n' 'package com.acme;' 'public class BExt extends BComponent {' \
    '  public static final Property p = newProperty(0, 0, null);' \
    '  public int getP() { return 0; }' '  private void setP(int v) { }' \
    '  public static final Property q = newProperty(0, 0, null);' \
    '  public void setQ(int v) { }' '}' > "$SRC/com/acme/BExt.java"
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$(_row 3 p 'setP(...) is not public')"$'\n'"$(_row 6 q 'no public getQ() or isQ()')" ]
}

@test "PA-K20: a root with no *.java is the ERROR row, exit 3" {
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-property-accessors  $SRC  no *.java to scan (K20: wrong path or empty scaffold)" ]
}
