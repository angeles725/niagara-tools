#!/usr/bin/env bats
# lint-duplicate-helper.bats — G41: a second copy of a helper in one module (same body shape of 8+ statements,
# or a private/static helper with the same name and parameter types in another class) is a WARN row; the
# rule's table tests live in go/internal/lint/rules/duplicatehelper. [ev: retro go-lint-duplicate-helper Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  DH="$KIT/toolbelt/lint-duplicate-helper.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  RT="$MOD/Mod-rt/src/com/x"
  UX="$MOD/Mod-ux/src/com/x/ux"
  mkdir -p "$RT" "$UX"
}

# helper <class> <method> — a class holding one private static 10-statement CP437 decoder
helper() {
  printf 'package com.x;\npublic class %s {\n  private static String %s(byte[] in) {\n    char[] out = new char[in.length];\n' "$1" "$2"
  local i; for i in 0 1 2 3 4 5 6 7; do printf '    out[%d] = (char) (in[%d] & 0xff);\n' "$i" "$i"; done
  printf '    return new String(out);\n  }\n}\n'
}

@test "DH-usage: no argument prints the usage and exits 3" {
  run "$DH"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-duplicate-helper.sh [--strict] <module-root>..." ]
}

@test "DH-K20: a root with no part src/ file is the ERROR row, exit 3, never a clean pass" {
  # Mutation: DH-K20 -- dropping the no-part ErrEmptyRoot reports a wrong-path root as a clean exit 0
  mkdir -p "$MOD/tools"; helper BT readCp437 > "$MOD/tools/BT.java"
  run "$DH" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-duplicate-helper  $MOD  no *.java under a -rt, -ux or -wb part's src/ (wrong path: pass the module root)" ]
}

@test "DH1: two identical 10-statement readCp437 bodies in two parts are one WARN row on the later one" {
  # Mutation: DH1 -- dropping the body-shape match passes two identical 10-statement readCp437 bodies clean
  helper BA readCp437 > "$RT/BA.java"; helper BC readCp437 > "$UX/BC.java"
  run "$DH" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-duplicate-helper  $UX/BC.java:3  same shape as $RT/BA.java:3 (readCp437: same body, 10 statements)" ]
}

@test "DH2: two static helpers with one name and parameter types, different bodies, are a WARN row" {
  # Mutation: DH2 -- dropping the signature match passes two static scale(int,double) helpers clean
  printf 'class BA {\n  static int scale(int v, double f) { return (int) (v * f); }\n}\n' > "$RT/BA.java"
  printf 'class BB {\n  static int scale(int x, double k) { return (int) Math.round(x * k); }\n}\n' > "$RT/BB.java"
  run "$DH" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-duplicate-helper  $RT/BB.java:2  same shape as $RT/BA.java:2 (scale(int,double): same name and parameter types)" ]
}

@test "DH3: two @Override implementations of one interface method are clean" {
  # Mutation: DH3 -- dropping the instance-method exemption flags two @Override run() implementations
  printf 'class BA implements Runnable {\n  @Override\n  public void run() { a(); }\n}\n' > "$RT/BA.java"
  printf 'class BB implements Runnable {\n  @Override public void run() { b(); }\n}\n' > "$RT/BB.java"
  run "$DH" "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "DH-strict: the same row exits 1 under --strict" {
  # Mutation: DH-strict -- giving the rule a non-WARN severity exits 0 under --strict with a row
  helper BA readCp437 > "$RT/BA.java"; helper BC readCp437 > "$UX/BC.java"
  run "$DH" --strict "$MOD"
  [ "$status" -eq 1 ]
  [[ "$output" == "WARN  lint-duplicate-helper  $UX/BC.java:3  "* ]]
}
