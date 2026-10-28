#!/usr/bin/env bats
# module-map.bats — toolbelt/module-map.sh (go/internal/modulemap, whose Go tests pin each subcommand) end to end
# on fresh scaffolds: add-concern keeps a module clean under lint-module-map, strict L3 and lint-size; where and
# tests read the map; scaffold-module.sh add-concern reaches the same tool. [ev: retro modularity-structure Δ1/Δ4]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SC="$KIT/toolbelt/scaffold-module.sh"
  MM="$KIT/toolbelt/module-map.sh"
  OUT="$BATS_TEST_TMPDIR/out"
  mkdir -p "$OUT"
}

# scaffold <type>: a fresh ModuleA of that type; R is its module root.
scaffold() {
  run "$SC" ModuleA "$OUT/$1" --type "$1" --vendor Example
  [ "$status" -eq 0 ]
  mkdir -p "$OUT/$1/ModuleA/.git"  # the L10/L14 upward walk stops here
  R="$OUT/$1/ModuleA/ModuleA"
}

# clean: lint-module-map prints no row, lint-structure no L3 row, lint-size no row on every part.
clean() {
  run "$KIT/toolbelt/lint-module-map.sh" "$R"
  [ "$status" -eq 0 ] && [ -z "$output" ] || { echo "module-map: $output"; false; }
  run "$KIT/toolbelt/lint-structure.sh" "$R"
  [ "$status" -eq 0 ] || { echo "structure: $output"; false; }
  [[ "$output" != *"L3"* ]] || { echo "structure: $output"; false; }
  local p
  for p in "$R"/ModuleA-*/src; do
    run "$KIT/toolbelt/lint-size.sh" "$p"
    [ "$status" -eq 0 ] && [[ "$output" != *"WARN"* && "$output" != *"FAIL"* ]] || { echo "size $p: $output"; false; }
  done
}

@test "MM-ADD1: add-concern on fresh logic and dashboard scaffolds (rt, rt --adapter, ux) keeps them clean" {
  scaffold logic
  run "$SC" add-concern Schedules --root "$R"
  [ "$status" -eq 0 ]
  [[ "$output" == *"created  ModuleA-rt/src/com/example/ModuleA/model/schedules/SchedulesLogic.java"* ]]
  [[ "$output" == *"mapped   MODULE-MAP.md:"*"  schedules"* ]]
  run "$MM" add-concern Battery --root "$R" --adapter
  [ "$status" -eq 0 ]
  [ -f "$R/ModuleA-rt/src/com/example/ModuleA/adapter/battery/BatteryAdapter.java" ]
  [ -f "$R/ModuleA-rt/srcTest/test/com/example/ModuleA/model/battery/BatteryLogicTest.java" ]
  clean
  scaffold dashboard
  run "$MM" add-concern Alarms --root "$R" --part ux
  [ "$status" -eq 0 ]
  [ -f "$R/ModuleA-ux/src/com/example/ModuleA/ux/route/alarms/AlarmsLogic.java" ]
  grep -q '^package com.example.ModuleA.ux.route.alarms;' "$R/ModuleA-ux/src/com/example/ModuleA/ux/route/alarms/AlarmsLogic.java"
  clean
}

@test "MM-ADD2: add-concern refuses an existing concern (exit 1) and leaves the map byte-identical" {
  scaffold logic
  cp "$R/MODULE-MAP.md" "$BATS_TEST_TMPDIR/before"
  run "$MM" add-concern Control --root "$R"
  [ "$status" -eq 1 ]
  [[ "$output" == *"already has a row"* ]]
  cmp "$R/MODULE-MAP.md" "$BATS_TEST_TMPDIR/before"
  run "$MM" add-concern Battery --root "$R"
  [ "$status" -eq 0 ]
  cp "$R/MODULE-MAP.md" "$BATS_TEST_TMPDIR/before"
  run "$MM" add-concern Battery --root "$R"
  [ "$status" -eq 1 ]
  cmp "$R/MODULE-MAP.md" "$BATS_TEST_TMPDIR/before"
}

@test "MM-WHERE1: where names a file's concern, its files and tests; an unlisted file exits 1" {
  scaffold dashboard
  run "$MM" where ModuleA-ux/src/rc/js/services/api.js --root "$R"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "concern  frontend-data" ]
  [[ "$output" == *"tests    ModuleA-ux/srcTest/rc/spec/ModuleASpec.js"* ]]
  run "$MM" where control --root "$R"
  [ "$status" -eq 0 ]
  [[ "$output" == *"files    ModuleA-rt/src/com/example/ModuleA/model/ModuleALogic.java"* ]]
  run "$MM" where ModuleA-rt/nothing.java --root "$R"
  [ "$status" -eq 1 ]
}

@test "MM-TESTS1: tests runs only the concern's pure tests and skips station tests (SKIP without javac or junit)" {
  scaffold logic
  run "$MM" tests station-test --root "$R"
  [ "$status" -eq 0 ]
  [[ "$output" == *"skip  ModuleA-rt/srcTest/test/com/example/ModuleA/BModuleALinkTest.java  imports javax.baja"* ]]
  [[ "$output" != *"run   "* ]]
  command -v javac >/dev/null 2>&1 || skip "no javac on PATH"
  [ -n "$(find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | head -1)" ] || skip "junit-4.13.2.jar not in ~/.gradle"
  run "$MM" add-concern Battery --root "$R"
  [ "$status" -eq 0 ]
  run "$MM" tests battery --root "$R"
  [ "$status" -eq 0 ] || { echo "$output"; false; }
  [[ "$output" == *"run   ModuleA-rt/srcTest/test/com/example/ModuleA/model/battery/BatteryLogicTest.java  com.example.ModuleA.model.battery.BatteryLogicTest"* ]]
  [[ "$output" == *"OK (1 test)"* ]]
  [[ "$output" != *"ModuleALogicTest"* ]]
}
