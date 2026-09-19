#!/usr/bin/env bats
# Tests for lint-arbitrary-ord.sh — BOrd.make(variable) from client input can resolve any ORD.
# [ev: retro 2026-09-19-security-model]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-arbitrary-ord.sh"
  mkdir -p "$TMPDIR_T/M/src"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "AO-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "AO1: BOrd.make of a string literal is clean" {
  printf 'class A { void f(){ BOrd.make("history:"); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "AO2: BOrd.make of a lowercase variable is flagged (WARN)" {
  printf 'class A { void f(String query){ BOrd.make(query); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"lint-arbitrary-ord"* ]]
  # Named mutation: require a leading quote -> AO2 WARN vanishes.
}
@test "AO3: BOrd.make of an UPPER_CASE constant is not flagged" {
  printf 'class A { void f(){ BOrd.make(SERVICE_ORD); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
