#!/usr/bin/env bats
# Tests for lint-arbitrary-ord.sh — BOrd.make(variable) from client input can resolve any ORD.
# [ev: retro 2026-09-19-security-model]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-arbitrary-ord.sh"
  mkdir -p "$TMPDIR_T/M/src"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "AO-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "AO1: BOrd.make of a string literal is clean" {
  printf 'class A { void f(){ BOrd.make("history:"); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "AO2: BOrd.make of a lowercase variable is flagged (WARN)" {
  printf 'class A { void f(String query){ BOrd.make(query); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"lint-arbitrary-ord"* ]]
  # Named mutation: require a leading quote -> AO2 WARN vanishes.
}
@test "AO3: BOrd.make of an UPPER_CASE constant is not flagged" {
  printf 'class A { void f(){ BOrd.make(SERVICE_ORD); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}

# Reviewed call-site marker [ev: retro panccadia-restart-seq-comp-lockout-hours Δ4]: a BOrd.make(var)
# built from a sanitized internal value WARNed on every run with no way to record the review.
# `// lint-arbitrary-ord: reviewed <reason>` on the same or the preceding line suppresses it; the
# reason is mandatory — a bare marker still WARNs.
@test "AO4: a same-line reviewed marker with a reason suppresses the WARN" {
  printf 'class A { void f(){ BOrd.make(tmpOrd); // lint-arbitrary-ord: reviewed backup file name built from a sanitized internal string\n} }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "AO5: a preceding-line reviewed marker with a reason suppresses the WARN" {
  printf 'class A { void f(){\n  // lint-arbitrary-ord: reviewed ord assembled from a code-controlled constant prefix\n  BOrd.make(tmpOrd);\n} }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "AO6: a reviewed marker with no reason still WARNs and names the missing reason" {
  printf 'class A { void f(String q){ BOrd.make(q); // lint-arbitrary-ord: reviewed\n} }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"reviewed marker without a reason"* ]]
}
@test "AO7: a reviewed marker two lines above does not cover the call site" {
  printf 'class A { void f(String q){\n  // lint-arbitrary-ord: reviewed unrelated earlier call\n  int x = 1;\n  BOrd.make(q);\n} }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]
}

# polish-2026-10-02 P2a (#199 WU6a): the marker counts only inside a real // comment. A "//" inside a
# string literal is not a comment start: it neither hides the call nor supplies a marker.
# Named mutation AO8: split code/comment at the first "//" regardless of quotes -> AO8 and AO9 flip.
@test "AO8: a reviewed marker inside a string literal does not suppress the WARN" {
  printf 'class A { void f(String q){ BOrd.make(q); log("see http://x lint-arbitrary-ord: reviewed not a comment"); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"BOrd.make from a variable"* ]]
}
@test "AO9: a // inside a string literal before the call does not hide BOrd.make(var)" {
  printf 'class A { void f(String q){ String u = "a//b"; BOrd.make(q); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"BOrd.make from a variable"* ]]
}
@test "AO10: a real trailing comment marker after a string holding // still suppresses the WARN" {
  printf 'class A { void f(String q){ BOrd.make(q); log("a//b"); // lint-arbitrary-ord: reviewed internal prefix only\n} }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "AO11: an escaped quote inside the string does not end it early (the // after it is still string)" {
  printf 'class A { void f(String q){ String u = "a\\"//b"; BOrd.make(q); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"BOrd.make from a variable"* ]]
}
