#!/usr/bin/env bats
# Tests for lint-bql-string-concat.sh — BQL built by string concat is an injection risk.
# [ev: retro 2026-09-19-security-model]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-bql-string-concat.sh"
  mkdir -p "$TMPDIR_T/M/src"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "BSC-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "BSC1: a static bql literal is clean" {
  printf 'class A { void f(){ BOrd.make("station:|bql:select * from control:ControlPoint"); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "BSC2: bql: with string concat is flagged (WARN)" {
  printf 'class A { void f(String u){ BOrd.make("alarm:|bql:select * where uuid = %s" + u + "%s"); } }\n' "'" "'" > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"lint-bql-string-concat"* ]]
  # Named mutation: drop the concat-operator test -> BSC2 WARN vanishes.
}
@test "BSC3: BqlQuery.make with concat is flagged (WARN)" {
  printf 'class A { void f(String u){ BqlQuery.make("select * where n=" + u); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]
}
