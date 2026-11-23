#!/usr/bin/env bats
# lint-rdbms-dialect-descriptor.bats — a history export with its dialect fixed in code WARNs, and so does a
# Postgres JDBC URL in a module with no BRdbms dialect (G88). The rule's cases live in
# go/internal/lint/rules/rdbmsdialect. [ev: retro doc-coverage-audit Δ16]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-rdbms-dialect-descriptor.sh"
  SRC="$BATS_TEST_TMPDIR/Mod/Mod-rt/src/com/acme"
  mkdir -p "$SRC"
}

@test "RD-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-rdbms-dialect-descriptor.sh [--strict] <module-root>..." ]
}

@test "RD-fixture: the MinimalPan module is clean under --strict" {
  run "$L" --strict "$KIT/fixtures/MinimalPan/MinimalPan"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "RD1: a dialect built in code WARNs, one taken from the database is clean" {
  printf '%s\n' 'package com.acme;' 'class BA {' '  Object e = new BSqlServerHistoryExport();' '}' > "$SRC/BA.java"
  printf '%s\n' 'package com.acme;' 'class BB {' '  Object e = db.getHistories().getExportDescriptorType().getInstance();' '}' > "$SRC/BB.java"
  run "$L" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-rdbms-dialect-descriptor  $BATS_TEST_TMPDIR/Mod/Mod-rt/src/com/acme/BA.java:3  hard-coded dialect descriptor: take it from getHistories().getExportDescriptorType().getInstance() (types/cloud-connector.md § 11)" ]
  run "$L" --strict "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 1 ]
}

@test "RD2: a Postgres URL WARNs, unless the module ships a class extending BRdbms" {
  printf '%s\n' 'package com.acme;' 'class BA {' '  String u = "jdbc:postgresql://db/x";' '}' > "$SRC/BA.java"
  run "$L" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BA.java:3  postgres jdbc url: "* ]]
  printf '%s\n' 'package com.acme;' 'class BPg extends BRdbms {' '}' > "$SRC/BPg.java"
  run "$L" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "RD-srctest: srcTest is not read, so a fixed dialect in a test stays clean under --strict" {
  mkdir -p "$BATS_TEST_TMPDIR/Mod/Mod-rt/srcTest/com/acme"
  printf '%s\n' 'package com.acme;' 'class BA { int x; }' > "$SRC/BA.java"
  printf '%s\n' 'package com.acme;' 'class T { Object e = new BMySQLHistoryExport(); }' > "$BATS_TEST_TMPDIR/Mod/Mod-rt/srcTest/com/acme/T.java"
  run "$L" --strict "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "RD-K20: a root with no -rt Java is an ERROR row and exit 3, never a clean pass" {
  mkdir -p "$BATS_TEST_TMPDIR/Other/Other-wb/src"
  printf '%s\n' 'class W {}' > "$BATS_TEST_TMPDIR/Other/Other-wb/src/W.java"
  run "$L" "$BATS_TEST_TMPDIR/Other"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-rdbms-dialect-descriptor  $BATS_TEST_TMPDIR/Other  no *.java under a -rt part's src/ (wrong path: pass the module root)" ]
}
