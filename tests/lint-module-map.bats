#!/usr/bin/env bats
# lint-module-map.bats — the toolbelt wrapper of the lint-module-map rule (go/internal/lint/rules/modulemap, whose
# goldens pin every MM row). These pin the wrapper end to end and the mutations its header declares.

setup() {
  LMM="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/lint-module-map.sh"
  M="$BATS_TEST_TMPDIR/M"
  mkdir -p "$M/M-rt/src" "$M/M-rt/srcTest"
  printf 'class A {}\n' > "$M/M-rt/src/A.java"
  printf 'class ATest {}\n' > "$M/M-rt/srcTest/ATest.java"
}

map() {  # map <mode-lines> <files-cell>
  printf '%b\n\n| Concern | Part | Files | Tests | Change here when |\n|---|---|---|---|---|\n| control | rt | %s | M-rt/srcTest/ATest.java | the decision |\n' "$1" "$2" > "$M/MODULE-MAP.md"
}

@test "MMB1: a strict module FAILs an unlisted source file (exit 1); a legacy one WARNs it (exit 0)" {
  printf 'class B {}\n' > "$M/M-rt/src/B.java"
  map 'modularity: strict' 'M-rt/src/A.java'
  run "$LMM" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-module-map  $M/M-rt/src/B.java  MM4: in no concern (list it in the Files of the concern that owns it)" ]
  map 'modularity: legacy\nlegacy-reason: predates the map' 'M-rt/src/A.java'
  run "$LMM" "$M"
  [ "$status" -eq 0 ]
  [[ "$output" == "WARN  lint-module-map  $M/M-rt/src/B.java  MM4: "* ]]
}

@test "MMB2: a listed file that does not exist FAILs MM3 on its map line" {
  map 'modularity: strict' 'M-rt/src/A.java; M-rt/src/Gone.java'
  run "$LMM" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-module-map  $M/MODULE-MAP.md:5  MM3: M-rt/src/Gone.java (concern control) does not exist" ]
}

@test "MMB3: a glob never lists a .java file (MM3 no match + MM4 unlisted)" {
  map 'modularity: strict' 'M-rt/src/**'
  run "$LMM" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == *"MM3: M-rt/src/** (concern control) matches no file"* ]]
  [[ "$output" == *"$M/M-rt/src/A.java  MM4: in no concern"* ]]
}

@test "MMB-clean: a complete strict map is clean (exit 0, no row); no map is one MM1 WARN" {
  map 'modularity: strict' 'M-rt/src/A.java'
  run "$LMM" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  rm "$M/MODULE-MAP.md"
  run "$LMM" "$M"
  [ "$status" -eq 0 ]
  [[ "$output" == "WARN  lint-module-map  $M  MM1: "* ]]
}

@test "MMB4: a strict row whose files sit in two concern folders FAILs MM7 (rows and folders); model/x with adapter/x is one concern" {
  mkdir -p "$M/M-rt/src/a/model/cooling" "$M/M-rt/src/a/model/schedules" "$M/M-rt/src/a/adapter/cooling"
  printf 'class L {}\n' > "$M/M-rt/src/a/model/cooling/L.java"
  printf 'class P {}\n' > "$M/M-rt/src/a/model/schedules/P.java"
  printf 'class BL {}\n' > "$M/M-rt/src/a/adapter/cooling/BL.java"
  map 'modularity: strict' 'M-rt/src/A.java; M-rt/src/a/model/cooling/L.java; M-rt/src/a/adapter/cooling/BL.java; M-rt/src/a/model/schedules/P.java'
  run "$LMM" "$M"
  [ "$status" -eq 1 ]
  # the count is of folder NAMES (cooling, schedules): model/cooling + adapter/cooling are one concern
  [[ "$output" == "FAIL  lint-module-map  $M/MODULE-MAP.md:5  MM7: concern control lists files in concern folders of 2 names (cooling, schedules: "* ]]
  map 'modularity: strict' 'M-rt/src/A.java; M-rt/src/a/model/cooling/L.java; M-rt/src/a/adapter/cooling/BL.java'
  printf '| schedules | rt | M-rt/src/a/model/schedules/P.java | M-rt/srcTest/ATest.java | a schedule |\n' >> "$M/MODULE-MAP.md"
  run "$LMM" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  # the folder-level row: a second row listing a file of the cooling folder
  printf 'class V {}\n' > "$M/M-rt/src/a/model/cooling/V.java"
  printf '| valves | rt | M-rt/src/a/model/cooling/V.java | M-rt/srcTest/ATest.java | a valve |\n' >> "$M/MODULE-MAP.md"
  run "$LMM" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-module-map  $M/M-rt/src/a/model/cooling  MM7: concern folder in 2 rows (control, valves); one row owns a concern folder"* ]]
}
