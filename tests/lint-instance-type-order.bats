#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-instance-type-order.sh (issue #847): a @NiagaraType class that
# declares a static INSTANCE after its static Type TYPE is refused by the station ("INSTANCE field is null;
# insure getType() comes last") while tests, build, verify and report pass.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-instance-type-order.sh"
  SRC="$BATS_TEST_TMPDIR/Mod-rt/src/com/x"; mkdir -p "$SRC"
}

# cls <file> <member line>...: a @NiagaraType class whose body is the given lines.
cls() { local f="$1"; shift; { printf '@NiagaraType\npublic class X extends BComponent\n{\n'; printf '%s\n' "$@"; printf '}\n'; } > "$SRC/$f"; }

@test "ITO-usage: no argument exits 3 with the usage line" {
  run "$LINT"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: lint-instance-type-order.sh"* ]]
}

@test "ITO0: INSTANCE after TYPE FAILs on the INSTANCE line, exit 1, with the station error and the fix" {
  cls Bad.java '  public static final Type TYPE = Sys.loadType(X.class);' '  public static final X INSTANCE = new X();'
  run "$LINT" "$BATS_TEST_TMPDIR/Mod-rt/src"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  lint-instance-type-order  $SRC/Bad.java:5  "* ]]
  [[ "$output" == *'"INSTANCE field is null; insure getType() comes last"'* ]]
  [[ "$output" == *"declare INSTANCE before TYPE" ]]
}

@test "ITO1: INSTANCE before TYPE, and a class with no INSTANCE, are clean (exit 0, no output)" {
  # Mutation: ITO1 -- reading INSTANCE position-blind flags a class that declares INSTANCE before TYPE
  cls Good.java '  public static final X INSTANCE = new X();' '  public static final Type TYPE = Sys.loadType(X.class);'
  cls NoInstance.java '  public static final Type TYPE = Sys.loadType(X.class);'
  run "$LINT" "$BATS_TEST_TMPDIR/Mod-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ITO2: INSTANCE text in a comment or a string literal is not a field" {
  # Mutation: ITO2 -- reading INSTANCE from comment or string text flags a quoted declaration
  cls Quoted.java '  public static final Type TYPE = Sys.loadType(X.class);' '  // static X INSTANCE = new X();' \
    '  static final String S = "static X INSTANCE = null;";'
  run "$LINT" "$BATS_TEST_TMPDIR/Mod-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ITO3: a root with no *.java is an ERROR, exit 3" {
  mkdir -p "$BATS_TEST_TMPDIR/empty"
  run "$LINT" "$BATS_TEST_TMPDIR/empty"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-instance-type-order  $BATS_TEST_TMPDIR/empty  no *.java sources found (wrong path or empty source tree)" ]
}
