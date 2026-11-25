#!/usr/bin/env bats
# scaffold-module --with-offload: the logic fixture's opt-in async-offload component (BPanOffload, its two station
# tests, MANAGE_EXECUTION and the rows that register them). The fixture stays the scaffold source; without the flag
# the emitted tree is the one the fixture had before the component, with it every fixture file is emitted.
# [ev: findings dev-course S6, B3] go/internal/scaffoldmodule/offload.go

load lib/scaffold-setup

@test "OFF1: a logic scaffold without --with-offload has no offload file, no MANAGE_EXECUTION and only the two base test entries" {
  # Mutation: OFF1 -- an offloadShape that never drops the fixture-only files emits the component into every module
  run "$SC" Acme "$OUT" --vendor Example
  [ "$status" -eq 0 ]
  RT="$OUT/Acme/Acme/Acme-rt"
  [ -z "$(find "$OUT/Acme" -name '*Offload*' -o -name '*PermissionsTest*')" ]
  [ -z "$(grep -rli 'offload' "$OUT/Acme" || true)" ]
  [ -z "$(grep -rl 'MANAGE_EXECUTION' "$OUT/Acme" || true)" ]
  [ "$(cat "$RT/module-permissions.xml")" = "$(printf '<?xml version="1.0" encoding="UTF-8"?>\n<permissions/>')" ]
  [ "$(grep -c '<type ' "$RT/moduleTest-include.xml")" -eq 2 ]
  grep -q 'AcmeStationTestBase' "$RT/moduleTest-include.xml"
  grep -q 'AcmeLinkTest' "$RT/moduleTest-include.xml"
}

@test "OFF2: --with-offload emits the component renamed, registers it, requests MANAGE_EXECUTION and leaves no marker" {
  # Mutation: OFF2 -- stripping the markers but keeping the overlay's moduleTest-include.xml drops the two test entries
  run "$SC" Acme "$OUT" --vendor Example --with-offload
  [ "$status" -eq 0 ]
  RT="$OUT/Acme/Acme/Acme-rt"
  [ -f "$RT/src/com/example/Acme/BPanOffload.java" ]
  [ -f "$RT/srcTest/test/com/example/Acme/BAcmeOffloadTest.java" ]
  [ -f "$RT/srcTest/test/com/example/Acme/BAcmePermissionsTest.java" ]
  grep -q 'package com.example.Acme;' "$RT/src/com/example/Acme/BPanOffload.java"
  grep -q '"Acme-offload-"' "$RT/src/com/example/Acme/BPanOffload.java"
  grep -q 'name="PanOffload"' "$RT/module-include.xml"
  grep -q 'PanOffload' "$RT/module.palette"
  grep -q '^PanOffload.displayName=' "$RT/module.lexicon"
  grep -q '<name>MANAGE_EXECUTION</name>' "$RT/module-permissions.xml"
  [ "$(grep -c '<type ' "$RT/moduleTest-include.xml")" -eq 4 ]
  grep -q 'BAcmeOffloadTest' "$RT/moduleTest-include.xml"
  grep -q 'BAcmePermissionsTest' "$RT/moduleTest-include.xml"
  grep -q '^| async-offload | rt | Acme-rt/src/com/example/Acme/BPanOffload.java |' "$OUT/Acme/Acme/MODULE-MAP.md"
  grep -q '^| permissions | rt |' "$OUT/Acme/Acme/MODULE-MAP.md"
  [ -z "$(grep -rl 'offload:' "$OUT/Acme" || true)" ]
}

@test "OFF3: the module scaffolded with --with-offload passes the offload lints (action, async, timers, permissions, structure)" {
  run "$SC" Acme "$OUT" --vendor Example --with-offload
  [ "$status" -eq 0 ]
  M="$OUT/Acme/Acme"
  for l in action-handler async-blocking lifecycle-guard intrinsic-lock permissions-xml; do
    run "$KIT/toolbelt/lint-$l.sh" --strict "$M"
    [ "$status" -eq 0 ]
  done
  run "$KIT/toolbelt/lint-timers.sh" "$M/Acme-rt/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL "* ]]
  run "$KIT/toolbelt/lint-structure.sh" "$M"
  [ "$status" -eq 0 ]
}

@test "OFF4: --with-offload on any other type is exit 2 with the refusal and writes nothing" {
  local t
  for t in dashboard driver library service; do
    run "$SC" Acme "$OUT" --vendor Example --type "$t" --with-offload
    [ "$status" -eq 2 ]
    [ "$output" = "scaffold-module: --with-offload needs --type logic; $t has no async-offload component" ]
    [ ! -e "$OUT/Acme" ]
  done
}

@test "OFF5: --with-offload with --with-doc emits both parts and one map with every row, the doc one before plumbing" {
  run "$SC" Acme "$OUT" --vendor Example --with-offload --with-doc
  [ "$status" -eq 0 ]
  M="$OUT/Acme/Acme"
  [ -f "$M/Acme-doc/Acme-doc.gradle.kts" ]
  [ -f "$M/Acme-rt/src/com/example/Acme/BPanOffload.java" ]
  [ "$(awk -F'|' '/^\| /{gsub(/ /,"",$2); gsub(/ /,"",$3); printf "%s/%s ", $2, $3}' "$M/MODULE-MAP.md")" = "Concern/Part control/rt config-save/rt async-offload/rt permissions/rt station-test/rt plumbing/doc plumbing/rt " ]
  [ -z "$(grep -rl 'offload:' "$OUT/Acme" || true)" ]
}

@test "OFF6: the internal --strip-offload-markers mode refuses a malformed block, names the file and line, and changes nothing" {
  mkdir -p "$BATS_TEST_TMPDIR/bad"
  printf '<types>\n  <!-- offload:begin -->\n  <t/>\n</types>\n' > "$BATS_TEST_TMPDIR/bad/module-include.xml"
  run "$SC" --strip-offload-markers "$BATS_TEST_TMPDIR/bad"
  [ "$status" -eq 3 ]
  [ "$output" = "scaffold-module: module-include.xml: line 2: offload:begin is never closed" ]
  grep -q 'offload:begin' "$BATS_TEST_TMPDIR/bad/module-include.xml"
}
