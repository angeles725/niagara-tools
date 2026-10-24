#!/usr/bin/env bats
# lint-private-api-import.bats — G65: a com.tridium import or fully qualified use in module code is a WARN row,
# except the NDriver framework default allow-list; the rule's golden cases live in
# go/internal/lint/rules/privateapiimport. [ev: retro n4-course-handson-ndriver Δ8]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-private-api-import.sh"
  DRV="$BATS_TEST_TMPDIR/acmeDriver"
  cp -r "$BATS_TEST_DIRNAME/fixtures/driver-lints/acmeDriver" "$DRV"
  MOD="$BATS_TEST_TMPDIR/Mod"
  RT="$MOD/Mod-rt/src/com/example/drv"
  mkdir -p "$RT"
}

MSG='is private API (devguide apiInfo: no compatibility guaranteed)'

@test "PAI-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-private-api-import.sh [--strict] [--allow <glob>]... [--no-default-allow] <module-root>" ]
}

@test "PAI1: the NDriver fixture module is clean with the default allow-list and WARNs every import without it" {
  # Mutation: PAI1 -- dropping the default allow-list turns the first run into 7 WARN rows
  run "$L" --strict "$DRV"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "$L" --no-default-allow "$DRV"
  [ "$status" -eq 0 ]
  [ "$(grep -c "^WARN  lint-private-api-import  " <<< "$output")" -eq 7 ]
  [[ "$output" == *"BAcmeNetwork.java:6  com.tridium.driver.util.DrUtil $MSG"* ]]
}

@test "PAI-flag: --no-default-allow is the engine's own flag, anywhere; a value or --default-allow is a usage error" {
  run "$L" "$DRV" --allow 'com.tridium.none.*' --no-default-allow
  [ "$status" -eq 0 ]
  [ "$(grep -c "^WARN  lint-private-api-import  " <<< "$output")" -eq 7 ]
  run "$L" --no-default-allow=1 "$DRV"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-private-api-import.sh [--strict] [--allow <glob>]... [--no-default-allow] <module-root>" ]
  run "$L" --default-allow off "$DRV"
  [ "$status" -eq 3 ]
  ! grep -q 'default-allow off' "$L"
}

@test "PAI2: an import in a comment or a string is not one; com.tridiumx is another prefix" {
  # Mutation: PAI2 -- matching raw lines (comments and strings kept) WARNs the commented-out import
  printf '%s\n' 'package com.example.drv;' '// import com.tridium.history.exporters.BHistoryToCsv;' \
    'import com.tridiumx.foo.Bar;' 'class A { String s = "new com.tridium.file.exporters.BITableToCsv()"; }' > "$RT/A.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PAI-forms: import, static import and a fully qualified name each WARN; --strict exits 1" {
  printf '%s\n' 'package com.example.drv;' 'import com.tridium.history.exporters.BHistoryToCsv;' \
    'import static com.tridium.foo.Util.bar;' 'class A { Object t = new com.tridium.file.exporters.BITableToCsv(); }' > "$RT/A.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "WARN  lint-private-api-import  $RT/A.java:2  com.tridium.history.exporters.BHistoryToCsv $MSG" ]
  [ "${lines[1]}" = "WARN  lint-private-api-import  $RT/A.java:3  com.tridium.foo.Util.bar $MSG" ]
  [ "${lines[2]}" = "WARN  lint-private-api-import  $RT/A.java:4  com.tridium.file.exporters.BITableToCsv $MSG" ]
  run "$L" --strict "$MOD"
  [ "$status" -eq 1 ]
}

@test "PAI-allow: --allow adds a glob; a srcTest/ source is not read" {
  printf '%s\n' 'package com.example.drv;' 'import com.tridium.history.exporters.BHistoryToCsv;' 'class A {}' > "$RT/A.java"
  mkdir -p "$MOD/Mod-rt/srcTest/com/example/drv"
  printf '%s\n' 'package com.example.drv;' 'import com.tridium.testng.Foo;' 'class T {}' > "$MOD/Mod-rt/srcTest/com/example/drv/T.java"
  run "$L" --strict --allow 'com.tridium.history.**' "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PAI-K20: a root with no part src/ *.java is the ERROR row, exit 3" {
  rm -r "$MOD/Mod-rt"; mkdir -p "$MOD/tools"; printf 'class T {}\n' > "$MOD/tools/T.java"
  run "$L" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-private-api-import  $MOD  no *.java under a -rt, -ux, -wb or -se part's src/ (wrong path: pass the module root)" ]
}
