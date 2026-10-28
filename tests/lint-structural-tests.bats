#!/usr/bin/env bats
# lint-structural-tests.bats — G34: a srcTest read of a .java source as text WARNs (FAIL under --strict) when the
# part's srcTest has no BTestNg/BTestNgStation class. The rule's golden cases live in
# go/internal/lint/rules/structuraltests. [ev: retro testing-parity-btestng Δ5]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-structural-tests.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  T="$MOD/Mod-rt/srcTest/test/com/example"
  mkdir -p "$T"
  printf '%s\n' 'package test.com.example;' 'public class ShapeTest {' \
    '  String s = new String(java.nio.file.Files.readAllBytes(java.nio.file.Paths.get("src/com/example/BPan.java")));' \
    '}' > "$T/ShapeTest.java"
}

ROW_TAIL='as text and Mod-rt/srcTest has no BTestNg or BTestNgStation class: test the behaviour in a station test (types/moduleTest.md)'

@test "ST-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-structural-tests.sh [--strict] <module-root>..." ]
}

@test "ST-warn: a structural test alone WARNs; --strict prints FAIL and exits 1" {
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-structural-tests  $T/ShapeTest.java:3  reads \"src/com/example/BPan.java\" $ROW_TAIL" ]
  run "$L" --strict "$MOD"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-structural-tests  $T/ShapeTest.java:3  reads \"src/com/example/BPan.java\" $ROW_TAIL" ]
}

@test "ST1: a BTestNg class (qualified superclass) in the part clears it" {
  # Mutation: ST1 -- accepting only BTestNgStation WARNs a part whose test class extends com.tridium.testng.BTestNg
  printf '%s\n' 'package test.com.example;' 'public class BPanTest extends com.tridium.testng.BTestNg {}' > "$T/BPanTest.java"
  run "$L" --strict "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ST2: a .json fixture read is clean even when the file names a .java file elsewhere" {
  # Mutation: ST2 -- matching any .java literal in the file WARNs a test that reads a .json fixture and only names a .java file
  printf '%s\n' 'package test.com.example;' 'public class ShapeTest {' \
    '  String j = new String(java.nio.file.Files.readAllBytes(java.nio.file.Paths.get("srcTest/fixture.json")));' \
    '  String n = "BPan.java";' '}' > "$T/ShapeTest.java"
  run "$L" --strict "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ST-K20: a root with no srcTest/ *.java is the ERROR row, exit 3" {
  rm -r "$MOD/Mod-rt/srcTest"; mkdir -p "$MOD/Mod-rt/src"; printf 'class A {}\n' > "$MOD/Mod-rt/src/A.java"
  run "$L" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-structural-tests  $MOD  no *.java under a srcTest/" ]
}
