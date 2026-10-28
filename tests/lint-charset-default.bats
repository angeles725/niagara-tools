#!/usr/bin/env bats
# lint-charset-default.bats — text bytes with the platform default charset WARN (CD1-CD4). The rule's golden cases
# live in go/internal/lint/rules/charsetdefault. [ev: retro course-coverage Δ19]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-charset-default.sh"
  SRC="$BATS_TEST_TMPDIR/src"
  mkdir -p "$SRC/com/acme"
}

@test "CD-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-charset-default.sh [--strict] <src-root>" ]
}

@test "CD-fixture: the MinimalPan -rt sources are clean under --strict" {
  run "$L" --strict "$KIT/fixtures/MinimalPan/MinimalPan/MinimalPan-rt/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "CD1: a no-argument getBytes() WARNs; getBytes(UTF_8) is clean" {
  # Mutation: CD1 -- dropping the no-argument check of getBytes WARNs getBytes(StandardCharsets.UTF_8)
  printf '%s\n' 'package com.acme;' 'class A {' '  byte[] a(String s) { return s.getBytes(java.nio.charset.StandardCharsets.UTF_8); }' \
    '  byte[] b(String s) { return s.getBytes(); }' '}' > "$SRC/com/acme/A.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-charset-default  $SRC/com/acme/A.java:4  CD1 getBytes() with no charset encodes with the platform default: pass StandardCharsets.UTF_8 (METHODOLOGY.md § rt (components))" ]
  run "$L" --strict "$SRC"
  [ "$status" -eq 1 ]
}

@test "CD1-receiver: getBytes() on a String WARNs; a custom class's getBytes() is clean" {
  # Mutation: CD1-receiver -- dropping the String-receiver test WARNs a custom Packet.getBytes()
  printf '%s\n' 'package com.acme;' 'class A {' '  byte[] a(Packet p) { return p.getBytes(); }' \
    '  byte[] b(String s) { return s.getBytes(); }' '}' > "$SRC/com/acme/A.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-charset-default  $SRC/com/acme/A.java:4  CD1 getBytes() with no charset encodes with the platform default: pass StandardCharsets.UTF_8 (METHODOLOGY.md § rt (components))" ]
}

@test "CD2: new String over a byte[] WARNs; over a char[] it is clean" {
  # Mutation: CD2 -- judging new String without the byte-array test WARNs new String(char[])
  printf '%s\n' 'package com.acme;' 'class A {' '  String a(char[] c) { return new String(c); }' \
    '  String b(byte[] b) { return new String(b); }' '}' > "$SRC/com/acme/A.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-charset-default  $SRC/com/acme/A.java:4  CD2 new String(byte[]) with no charset decodes with the platform default: pass StandardCharsets.UTF_8 (METHODOLOGY.md § rt (components))" ]
}

@test "CD3: a one-argument InputStreamReader WARNs; one with a charset is clean" {
  # Mutation: CD3 -- counting every InputStreamReader argument list as charset-less WARNs new InputStreamReader(in, UTF_8)
  printf '%s\n' 'package com.acme;' 'class A {' '  Object a(java.io.InputStream in) { return new java.io.InputStreamReader(in, java.nio.charset.StandardCharsets.UTF_8); }' \
    '  Object b(java.io.InputStream in) { return new java.io.InputStreamReader(in); }' '}' > "$SRC/com/acme/A.java"
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-charset-default  $SRC/com/acme/A.java:4  CD3 new InputStreamReader(in) with no charset uses the platform default: pass StandardCharsets.UTF_8 as the second argument (METHODOLOGY.md § rt (components))" ]
}

@test "CD-K20: a root with no *.java is the ERROR row, exit 3" {
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-charset-default  $SRC  no *.java to scan (K20: wrong path or empty scaffold)" ]
}
