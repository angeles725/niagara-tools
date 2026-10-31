#!/usr/bin/env bats
# port-parity.bats — toolbelt/lint-port-parity.sh: a module that ports a reference frontend declares it in
# <module-root>/.port-parity (ref: header + one waiver per absence); every reference file, DOM id or
# function the port lacks without a decision-backed waiver FAILs. The fixture's module drops a whole
# equipment view, waived line by line with [decision: W5], and renames one file.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  PP="$KIT/toolbelt/lint-port-parity.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/port-parity"
  W="$BATS_TEST_TMPDIR/w"
  cp -r "$FX" "$W"
}

@test "PP1: every absence waived with a decision is clean (exit 0, three INFO rows, then the summary line)" {
  run "$PP" "$W/module"
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 4 ]
  [[ "${lines[0]}" == "INFO  "* ]]
  [[ "${lines[1]}" == "INFO  "* ]]
  [[ "${lines[2]}" == "INFO  "* ]]
  [ "${lines[3]}" = "port-parity: 4 files, 5 ids, 6 functions in the reference; 9 missing, 9 waived, 0 FAIL, 0 WARN" ]
}

@test "PP-drop-waiver: removing one waiver line flips exit 0 to 1 and names the unwaived function" {
  # Mutation: PP-drop-waiver -- a gate that ignores missing items stays exit 0 here
  sed -i '/^function renderEquipment /d' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  js/ui.js:5  missing function renderEquipment (in the reference, not in the port, no waiver)"* ]]
}

@test "PP-no-decision: a waiver without [decision: <ref>] FAILs on its own line" {
  # Mutation: PP-no-decision -- accepting a reason alone lets a reference feature drop with no recorded decision
  sed -i 's/^id eqPanel -- part of the equipment view \[decision: W5\]$/id eqPanel -- part of the equipment view/' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  .port-parity:7  waiver id eqPanel has no [decision: <ref>]"* ]]
}

@test "PP-stale: a waiver matching nothing is a WARN row and keeps exit 0" {
  printf 'function renderLevels -- not dropped [decision: W6]\n' >> "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-port-parity  .port-parity:12  stale waiver: function renderLevels matches no missing item (remove the line)"* ]]
}

@test "PP-cascade: a file waiver waives what only that file defines, as INFO rows" {
  run "$PP" "$W/module"
  [ "$status" -eq 0 ]
  [[ "$output" == *"INFO  lint-port-parity  js/sim.js:1  function simTick waived via file sim.js (defined only in that reference file)"* ]]
  [[ "$output" == *"INFO  lint-port-parity  js/sim.js:6  id simPanel waived via file sim.js (defined only in that reference file)"* ]]
}

@test "PP-cascade-shared: a name the waived file shares with another reference file is not cascaded" {
  # Mutation: PP-cascade-shared -- cascading a name another reference file also defines hides that file's drop
  sed -i '/^function eqHistory /d' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  js/sim.js:5  missing function eqHistory (in the reference, not in the port, no waiver)"* ]]
}

@test "PP-drop-file-waiver: removing the file waiver line flips exit 0 to 1 for the file and what it defined" {
  # Mutation: PP-drop-file-waiver -- a cascade that does not need the file waiver stays exit 0 here
  sed -i '/^file sim.js /d' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  js/sim.js:1  missing file sim.js"* ]]
  [[ "$output" == *"FAIL  lint-port-parity  js/sim.js:1  missing function simTick"* ]]
}

@test "PP-rename: dropping the rename line FAILs the reference file the port renamed" {
  sed -i '/^rename:/d' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  js/update-banner.js:1  missing file update-banner.js"* ]]
}

@test "PP-rename-fn: a function renamed in the port passes as INFO renamed to <port>" {
  run "$PP" "$W/module"
  [ "$status" -eq 0 ]
  [[ "$output" == *"INFO  lint-port-parity  js/ui.js:11  function drawLevelTitle renamed to levelTitle (renamed in the port)"* ]]
}

@test "PP-rename-target: a function rename whose target the port lacks FAILs" {
  # Mutation: PP-rename-target -- accepting a rename whose target is absent hides a dropped function or id
  sed -i 's/^rename: function drawLevelTitle -> levelTitle /rename: function drawLevelTitle -> noSuchTitle /' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  .port-parity:11  rename target function noSuchTitle is not in the port"* ]]
}

@test "PP-rename-reason: a function rename without -- <reason> FAILs" {
  sed -i 's/^rename: function drawLevelTitle -> levelTitle -- renamed in the port$/rename: function drawLevelTitle -> levelTitle/' "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-port-parity  .port-parity:11  rename function drawLevelTitle -> levelTitle has no -- <reason>"* ]]
}

@test "PP-flags: --ref/--port without waivers FAILs every absence, JSON on --format json" {
  run "$PP" --ref "$W/reference" --port "$W/module/src/rc" --format json
  [ "$status" -eq 1 ]
  [[ "$output" == *'"missing": 9'* ]]
  [[ "$output" == *'"fail": 9'* ]]
}

@test "PP-env: a module with no .port-parity is exit 3 naming the file (report-module SKIPs it)" {
  rm "$W/module/.port-parity"
  run "$PP" "$W/module"
  [ "$status" -eq 3 ]
  [ "$output" = "port-parity: no .port-parity in $W/module (the module declares no reference port)" ]
}

@test "PP-env-ref: an empty reference tree is exit 3, never a vacuous pass" {
  rm -r "$W/reference"
  mkdir "$W/reference"
  run "$PP" "$W/module"
  [ "$status" -eq 3 ]
  [[ "$output" == "port-parity: cannot read the reference: "*"no .js, .mjs, .html or .css file"* ]]
}
