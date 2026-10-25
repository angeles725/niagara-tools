#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-agent-on-include-sync.sh (kit lint G99, issue #473): a @NiagaraType
# @AgentOn type whose module-include.xml <type> line lacks one of its <on type> entries; the processor never adds
# them to an existing agent-less line (types/module-wiring.md § 2).
# [ev: retro n4-course-handson-slotomatic-authoring Δ5]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-agent-on-include-sync.sh"
  M="$BATS_TEST_TMPDIR/trafficLightDriver"
  cp -r "$BATS_TEST_DIRNAME/fixtures/agent-on-include-sync/trafficLightDriver" "$M"
  INC="$M/trafficLightDriver-wb/module-include.xml"
  JAVA="$M/trafficLightDriver-wb/src/com/tridiumuniversity/trafficLightDriver/ui/BTrafficLightDriverPointManager.java"
}

# java <file> <annotation-line>: a part source declaring class B in package p.
java() { mkdir -p "$(dirname "$1")"; printf 'package p;\n%s\npublic class B {}\n' "$2" > "$1"; }

@test "AOIS-usage: no argument exits 3 with the usage line" {
  run "$L"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: lint-agent-on-include-sync.sh"* ]]
}

@test "AOIS-lab: the course Bonus Lab trafficLightDriver-wb tree is clean" {
  run "$L" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AOIS1: the Bonus Lab tree with its <agent> block removed FAILs naming BTrafficLightDriverPointManager" {
  # Mutation: AOIS1 -- skipping a <type> line with no <agent> turns this case green (exit 0, no row)
  perl -0pi -e 's#\s*<agent>.*?</agent>\s*#\n  #s' "$INC"
  run "$L" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-agent-on-include-sync  $JAVA:16  com.tridiumuniversity.trafficLightDriver.ui.BTrafficLightDriverPointManager: module-include.xml <type> line lacks <on type> for: trafficLightDriver:TrafficLightDriverPointDeviceExt, trafficLightDriver:TrafficLightDriverPointFolder" ]
}

@test "AOIS-partial: one declared on type missing is named alone" {
  perl -0pi -e 's#\s*<on type="trafficLightDriver:TrafficLightDriverPointFolder"/>##' "$INC"
  run "$L" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BTrafficLightDriverPointManager: module-include.xml <type> line lacks <on type> for: trafficLightDriver:TrafficLightDriverPointFolder" ]]
}

@test "AOIS-forms: a single string and two @AgentOn are read; a second @AgentOn's type missing FAILs" {
  local p="$BATS_TEST_TMPDIR/P/P-wb"
  java "$p/src/p/B.java" '@NiagaraType(agent = @AgentOn(types = "m:A"))'
  printf '<types><type class="p.B" name="B"><agent><on type="m:A"/></agent></type></types>\n' > "$p/module-include.xml"
  run "$L" "$BATS_TEST_TMPDIR/P"
  [ "$status" -eq 0 ]
  java "$p/src/p/B.java" '@NiagaraType(agent = {@AgentOn(types = "m:A"), @AgentOn(types = {"m:C"})})'
  run "$L" "$BATS_TEST_TMPDIR/P"
  [ "$status" -eq 1 ]
  [[ "$output" == *"p.B: module-include.xml <type> line lacks <on type> for: m:C" ]]
}

@test "AOIS2: an @AgentOn in a comment or a string is not one" {
  # Mutation: AOIS2 -- tokenizing raw lines (comments kept) FAILs the commented-out @AgentOn
  local p="$BATS_TEST_TMPDIR/P/P-wb"
  mkdir -p "$p"
  printf '<types><type class="p.B" name="B"/></types>\n' > "$p/module-include.xml"
  java "$p/src/p/B.java" '/* @NiagaraType(agent = @AgentOn(types = {"m:A"})) */ // @AgentOn(types = "m:B")'
  run "$L" "$BATS_TEST_TMPDIR/P"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  java "$p/src/p/B.java" 'class S { String s = "@NiagaraType(agent = @AgentOn(types = \"m:B\"))"; }'
  run "$L" "$BATS_TEST_TMPDIR/P"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "AOIS-absent: a class with no <type> line FAILs naming the file it is missing from" {
  perl -0pi -e 's#<type class=.*?</type>##s' "$INC"
  run "$L" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BTrafficLightDriverPointManager: no <type class=\"com.tridiumuniversity.trafficLightDriver.ui.BTrafficLightDriverPointManager\"> line in $INC (needs <on type> for: "* ]]
}

@test "AOIS-env: a root with no module-include.xml, a malformed one or an unreadable directory exits 3" {
  rm "$INC"
  run "$L" "$M"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-agent-on-include-sync  $M  no module-include.xml under the root (wrong path: pass a module or part root)" ]
  printf '<types>\n' > "$INC"
  run "$L" "$M"
  [ "$status" -eq 3 ]
  [[ "$output" == "lint-agent-on-include-sync: $INC:2:1: "* ]]
  printf '<types/>\n' > "$INC"
  mkdir "$M/locked"; chmod 000 "$M/locked"
  if [ -r "$M/locked" ]; then chmod 755 "$M/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$M"
  chmod 755 "$M/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}

@test "AOIS3: a production module-include.xml registering a srcTest/ type or a moduleTest-include.xml type FAILs (test-type-in-prod)" {
  # Mutation: AOIS3 -- dropping the test-type-in-prod sub-check (testLeakRows) turns this case green (exit 0, no row)
  local p="$BATS_TEST_TMPDIR/ModuleA/ModuleA-rt"
  java "$p/src/p/B.java" '@NiagaraType'
  mkdir -p "$p/srcTest/p/test"
  printf 'package p.test;\n@NiagaraType\npublic class BModuleATest extends BTestNg {}\n' > "$p/srcTest/p/test/BModuleATest.java"
  printf '<types>\n<type name="B" class="p.B"/>\n</types>\n' > "$p/module-include.xml"
  printf '<types>\n<type name="ModuleATest" class="p.test.BModuleATest"/>\n</types>\n' > "$p/moduleTest-include.xml"
  run "$L" "$BATS_TEST_TMPDIR/ModuleA"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  # compileModuleTestJava (gradle plugin 7.3.40) appended the test type to the production file
  printf '<types>\n<type name="B" class="p.B"/>\n<type name="ModuleATest" class="p.test.BModuleATest"/>\n<type name="X" class="p.test.BX"/>\n</types>\n' > "$p/module-include.xml"
  printf '<types>\n<type name="X" class="p.test.BX"/>\n</types>\n' > "$p/moduleTest-include.xml"
  run "$L" "$BATS_TEST_TMPDIR/ModuleA"
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "FAIL  test-type-in-prod  $p/module-include.xml:3  p.test.BModuleATest: test type in the production module-include.xml (source $p/srcTest/p/test/BModuleATest.java) -- move the <type> line to moduleTest-include.xml; compileModuleTestJava appends it here (types/moduleTest.md)" ]
  [[ "${lines[1]}" == "FAIL  test-type-in-prod  $p/module-include.xml:4  p.test.BX: test type in the production module-include.xml (also listed in $p/moduleTest-include.xml) -- "* ]]
  [ "${#lines[@]}" -eq 2 ]
}

@test "AOIS3-env: a malformed moduleTest-include.xml exits 3 naming its position" {
  local p="$BATS_TEST_TMPDIR/ModuleA/ModuleA-rt"
  mkdir -p "$p"
  printf '<types/>\n' > "$p/module-include.xml"
  printf '<types>\n' > "$p/moduleTest-include.xml"
  run "$L" "$BATS_TEST_TMPDIR/ModuleA"
  [ "$status" -eq 3 ]
  [[ "$output" == "lint-agent-on-include-sync: $p/moduleTest-include.xml:2:1: "* ]]
}
