#!/usr/bin/env bats
# Tests for lint-driver-scaffold.sh — NDriver wizard stubs left in shipped sources.
# The wizard's doPing() calls pingOk() unconditionally, so a device that was never given a real ping
# reports UP forever (silent success). [ev: retro n4-course-handson-ndriver Δ2]
setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-driver-scaffold.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-driver-scaffold"
  TMPDIR_T="$(mktemp -d)"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "LDS-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "LDS1: a finished doPing (socket reachability, course if/else) and a Context-submitted job are clean" {
  run "$L" "$FX/done"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LDS2: the wizard doPing whose only statement is pingOk() FAILs" {
  # Mutation: LDS2 -- drop the pingOk-only body match and the TODO match, and the wizard doPing stops failing
  run "$L" "$FX/wizard"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/wizard/BNfooDevice.java:6  doPing() only calls pingOk()"* ]]
}

@test "LDS3: a doPing that still holds the wizard TODO FAILs even with more statements" {
  # Mutation: LDS3 -- drop the TODO-in-doPing match and a half-edited wizard doPing passes
  mkdir -p "$TMPDIR_T/src"
  printf '%s\n' 'class BX extends BNDevice {' '  public void doPing() {' '    // TODO - add ping implementation' \
    '    log("ping");' '    pingOk();' '  }' '}' > "$TMPDIR_T/src/BX.java"
  run "$L" "$TMPDIR_T/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $TMPDIR_T/src/BX.java:2  doPing() still holds the wizard TODO"* ]]
}

@test "LDS4: the other wizard markers WARN: poll TODO, linkMessage TODO, null message factory, submit(null)" {
  # Mutation: LDS4 -- drop the submit(null) match and the wizard discovery job stops warning
  run "$L" "$FX/wizard"
  [ "$status" -eq 1 ]     # the wizard device's FAIL; the WARN rows below never lower it
  [[ "$output" == *"WARN  lint-driver-scaffold  $FX/wizard/BNfooDevice.java:17  wizard stub: // TODO add poll support"* ]]
  [[ "$output" == *"WARN  lint-driver-scaffold  $FX/wizard/NfooMessageFactory.java:9  wizard stub: // TODO - convert linkMessage"* ]]
  [[ "$output" == *"WARN  lint-driver-scaffold  $FX/wizard/NfooMessageFactory.java:6  toResponse() only returns null"* ]]
  [[ "$output" == *"WARN  lint-driver-scaffold  $FX/wizard/BNfooNetwork.java:6  doSubmitDiscoveryJob() submits with a null Context"* ]]
}

@test "LDS5: a commented-out pingOk-only doPing is not a method body (comments are stripped)" {
  mkdir -p "$TMPDIR_T/src"
  printf '%s\n' 'class BY extends BNDevice {' '  /* public void doPing() { pingOk(); } */' '}' > "$TMPDIR_T/src/BY.java"
  run "$L" "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LDS6: string, char and text-block literals do not hide a stub (\"tcp://host\" and '{' before it, a \"/*\" string, a text block holding braces)" {
  # Mutation: LDS6 -- stop skipping string/char literals in the comment stripper and both stubs stop failing
  run "$L" "$FX/literals"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/literals/BUrlDevice.java:8  doPing() only calls pingOk()"* ]]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/literals/BMarkerDevice.java:8  doPing() only calls pingOk()"* ]]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/literals/BTextBlockDevice.java:10  doPing() only calls pingOk()"* ]]
  [[ "$output" != *"not trusted"* ]]   # a closed text block is parsed, not a reason to distrust the file
}

@test "LDS7: a file whose braces do not balance gets a WARN row instead of silently trusted boundaries" {
  # Mutation: LDS7 -- drop the END balance check and the truncated file reports clean
  run "$L" "$FX/unbalanced"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-driver-scaffold  $FX/unbalanced/BTruncatedDevice.java:9  method boundaries not trusted"* ]]
  mkdir -p "$TMPDIR_T/src"
  printf '%s\n' 'class BZ extends BNDevice {' '  String T = """' '    }' '}' > "$TMPDIR_T/src/BZ.java"
  run "$L" "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-driver-scaffold  $TMPDIR_T/src/BZ.java:4  method boundaries not trusted (ends inside a text block)"* ]]
}

@test "LDS8: WARN-tier stubs alone (network + message factory, no device) exit 0 with their WARN rows" {
  mkdir -p "$TMPDIR_T/src"
  cp "$FX/wizard/BNfooNetwork.java" "$FX/wizard/NfooMessageFactory.java" "$TMPDIR_T/src/"
  run "$L" "$TMPDIR_T/src"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" == *"WARN  lint-driver-scaffold  $TMPDIR_T/src/BNfooNetwork.java:6  doSubmitDiscoveryJob() submits with a null Context"* ]]
  [[ "$output" == *"WARN  lint-driver-scaffold  $TMPDIR_T/src/NfooMessageFactory.java:6  toResponse() only returns null"* ]]
  [[ "$output" == *"WARN  lint-driver-scaffold  $TMPDIR_T/src/NfooMessageFactory.java:9  wizard stub: // TODO - convert linkMessage"* ]]
}

@test "LDS9: annotations, a throws line and a nested device class do not hide a stub" {
  # Mutation: LDS9 -- name the method by the leftmost identifier( without dropping annotations and the annotated stubs stop failing
  run "$L" "$FX/annotated"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/annotated/BSameLineDevice.java:6  doPing() only calls pingOk()"* ]]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/annotated/BArrayAnnotDevice.java:6  doPing() only calls pingOk()"* ]]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/annotated/BThrowsDevice.java:7  doPing() only calls pingOk()"* ]]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $FX/annotated/BOuter.java:8  doPing() only calls pingOk()"* ]]
  [ "$(grep -c '^FAIL ' <<< "$output")" -eq 4 ]
}

@test "LDS10: a backslash in the source path is reported as is" {
  # Mutation: LDS10 -- pass the path with awk -v and its \n becomes a newline in the row
  d="$TMPDIR_T/src/a\\nb"
  mkdir -p "$d"
  cp "$FX/wizard/BNfooDevice.java" "$d/"
  run "$L" "$TMPDIR_T/src"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-driver-scaffold  $d/BNfooDevice.java:6  doPing() only calls pingOk()"* ]]
}

@test "LDS-finderr: an unreadable sub-directory is an env error (exit 3), never a clean pass" {
  # Mutation: LDS-finderr -- ignoring the find exit status skips the unreadable directory and exits 0
  mkdir -p "$TMPDIR_T/src/locked"
  cp "$FX/wizard/BNfooDevice.java" "$TMPDIR_T/src/locked/"
  chmod 000 "$TMPDIR_T/src/locked"
  if [ -r "$TMPDIR_T/src/locked" ]; then chmod 755 "$TMPDIR_T/src/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$TMPDIR_T/src"
  chmod 755 "$TMPDIR_T/src/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}
