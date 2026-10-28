#!/usr/bin/env bats
# lint-permissions-xml.bats — G21: a module-permissions.xml root other than <permissions> or a group type other than
# station|workbench|all FAILs; a request without <name>/<purposeKey> and an absolute new File/Paths.get in -rt code
# WARN. The rule's golden cases live in go/internal/lint/rules/permissionsxml.
# [ev: retro n4-course-authoring-files-and-libraries Δ2]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-permissions-xml.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  RT="$MOD/Mod-rt"
  mkdir -p "$RT/src/com/example"
}

USAGE='usage: lint-permissions-xml.sh [--strict] <module-root>...'

@test "PX-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "$USAGE" ]
}

@test "PX1: the kit template and every fixture are clean; the old <modulePermissions/> root FAILs" {
  # Mutation: PX1 -- accepting any root element passes the old <modulePermissions/> fixture root
  cp "$KIT/templates/module-permissions.xml" "$RT/module-permissions.xml"
  run "$L" --strict "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  for f in MinimalPan MinimalDash MinimalDriver; do
    run "$L" --strict "$KIT/fixtures/$f/$f"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
  done
  printf '%s\n' '<modulePermissions/>' > "$RT/module-permissions.xml"
  run "$L" "$MOD"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-permissions-xml  $RT/module-permissions.xml:1  root element <modulePermissions> is not <permissions>: baja reads only <permissions> (NModule), so every request is ignored" ]
}

@test "PX-type: a group type other than station, workbench or all FAILs" {
  printf '%s\n' '<permissions>' '  <niagara-permission-groups type="daemon"/>' '</permissions>' > "$RT/module-permissions.xml"
  run "$L" "$MOD"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-permissions-xml  $RT/module-permissions.xml:2  <niagara-permission-groups type=\"daemon\"> is not station, workbench or all" ]
}

@test "PX-request: a req-permission without purposeKey WARNs; --strict exits 1" {
  printf '%s\n' '<permissions><niagara-permission-groups type="station">' \
    '<req-permission><name>REFLECTION</name></req-permission>' '</niagara-permission-groups></permissions>' > "$RT/module-permissions.xml"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-permissions-xml  $RT/module-permissions.xml:2  <req-permission> REFLECTION has no <purposeKey> (both <name> and <purposeKey> are required)" ]
  run "$L" --strict "$MOD"
  [ "$status" -eq 1 ]
}

@test "PX2: an absolute new File in -rt code WARNs; a home-relative one, a comment and a string do not" {
  # Mutation: PX2 -- matching raw Java lines (comments and strings kept) WARNs a commented-out new File("/etc/passwd")
  printf '%s\n' 'package com.example;' 'class A {' '  java.io.File a = new java.io.File("/tmp/x");' \
    '  java.io.File b = new java.io.File(Sys.getStationHome(), "x");' '  // java.io.File c = new java.io.File("/etc/passwd");' \
    '  String s = "new File(\"/etc/passwd\")";' '}' > "$RT/src/com/example/A.java"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-permissions-xml  $RT/src/com/example/A.java:3  new File(\"/tmp/x\") is an absolute path outside the station and user homes: the security manager refuses it without a FILE_WRITE grant (use Sys.getStationHome())" ]
}

@test "PX-parse: a module-permissions.xml that does not parse is exit 3" {
  printf '%s\n' '<permissions>' '<niagara-permission-groups type="station">' '</permissions>' > "$RT/module-permissions.xml"
  run "$L" "$MOD"
  [ "$status" -eq 3 ]
  [[ "$output" == "lint-permissions-xml: cannot parse $RT/module-permissions.xml: XML syntax error on line 3"* ]]
}

@test "PX-K20: a root with no module-permissions.xml and no -rt src/ *.java is the ERROR row, exit 3" {
  rm -r "$RT"; mkdir -p "$MOD/Mod-ux/src"; printf 'class U {}\n' > "$MOD/Mod-ux/src/U.java"
  run "$L" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-permissions-xml  $MOD  no module-permissions.xml and no *.java under a -rt part's src/ (wrong path: pass the module root)" ]
}
