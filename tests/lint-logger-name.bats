#!/usr/bin/env bats
# lint-logger-name.bats — G40: a Logger.getLogger name other than the exact moduleName (or a <moduleName>.<child>
# logger) and a Logger.getLogger(X.class.getName()) WARN. The rule's golden cases live in
# go/internal/lint/rules/loggername. [ev: retro n4-course-parity-observability-horizons Δ3]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-logger-name.sh"
  MOD="$BATS_TEST_TMPDIR/MinimalPan"
  cp -r "$KIT/fixtures/MinimalPan/MinimalPan" "$MOD"
  SRC="$MOD/MinimalPan-rt/src/com/example/log"
  mkdir -p "$SRC"
}

@test "LN-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-logger-name.sh [--strict] <module-root>..." ]
}

@test "LN-fixture: the MinimalPan fixture is clean under --strict" {
  run "$L" --strict "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LN1: a lower-case logger name WARNs (the name is exact, from moduleName.set)" {
  # Mutation: LN1 -- comparing case-insensitively passes a lower-case "minimalpan" logger in module MinimalPan
  printf '%s\n' 'package com.example.log;' 'class A { java.util.logging.Logger l = java.util.logging.Logger.getLogger("minimalpan"); }' > "$SRC/A.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-logger-name  $SRC/A.java:2  Logger.getLogger(\"minimalpan\") is not the module name MinimalPan or a MinimalPan.<child> logger (types/observability.md § 1)" ]
  run "$L" --strict "$MOD"
  [ "$status" -eq 1 ]
}

@test "LN2: a <module>.<child> logger is clean; a class-named logger WARNs" {
  # Mutation: LN2 -- dropping the <module>.<child> exemption WARNs a "MinimalPan.defrost" child logger
  printf '%s\n' 'package com.example.log;' 'class A {' '  java.util.logging.Logger l = java.util.logging.Logger.getLogger("MinimalPan.defrost");' \
    '  java.util.logging.Logger m = java.util.logging.Logger.getLogger(A.class.getName());' '}' > "$SRC/A.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-logger-name  $SRC/A.java:4  Logger.getLogger(A.class.getName()) names a class, not the module MinimalPan (types/observability.md § 1)" ]
}

@test "LN3: a Java package named test under src/ is read; the part's own test/ is not" {
  # Mutation: LN3 -- pruning every directory named test skips the src/ package com/example/test
  local T="$MOD/MinimalPan-rt/src/com/example/test"
  mkdir -p "$T" "$MOD/MinimalPan-rt/test"
  printf '%s\n' 'package com.example.test;' 'class A { java.util.logging.Logger l = java.util.logging.Logger.getLogger("minimalpan"); }' > "$T/A.java"
  cp "$T/A.java" "$MOD/MinimalPan-rt/test/B.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-logger-name  $T/A.java:2  Logger.getLogger(\"minimalpan\") is not the module name MinimalPan or a MinimalPan.<child> logger (types/observability.md § 1)" ]
}

@test "LN-K20: a root with no part src/ *.java is the ERROR row, exit 3" {
  E="$BATS_TEST_TMPDIR/Empty"; mkdir -p "$E/tools"; printf 'class T {}\n' > "$E/tools/T.java"
  run "$L" "$E"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-logger-name  $E  no *.java under a -rt, -ux, -wb or -se part's src/ (wrong path: pass the module root)" ]
}
