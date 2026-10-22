#!/usr/bin/env bats
# Tests for lint-null-context-write.sh — null Context skips audit + grants BPermissions.all.
# [ev: retro 2026-09-19-security-model]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-null-context-write.sh"
  mkdir -p "$TMPDIR_T/M/src"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "NCW-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "NCW1: a write with a real Context is clean" {
  printf 'class A { void f(Context cx){ parent.set(prop, val, cx); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "NCW2: set(...,null) is flagged (WARN)" {
  printf 'class A { void f(){ parent.set(prop, val, null); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"lint-null-context-write"* ]]
  # Named mutation: drop the ,null match -> NCW2 WARN vanishes.
}
@test "NCW3: invoke(...,null) is flagged (WARN)" {
  printf 'class A { void f(){ comp.invoke(act, arg, null); } }\n' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]
}

@test "NCW-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass (audit A0)" {
  # Mutation: NCW-finderr -- ignoring the find exit status skips the unreadable directory's files and exits 0.
  # [ev: issue #226 R3-find-error-still-fail-open]
  mkdir -p "$TMPDIR_T/M/src/locked"
  printf '%s\n' 'class A {}' > "$TMPDIR_T/M/src/locked/A.java"
  chmod 000 "$TMPDIR_T/M/src/locked"
  if [ -r "$TMPDIR_T/M/src/locked" ]; then chmod 755 "$TMPDIR_T/M/src/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$TMPDIR_T/M"
  chmod 755 "$TMPDIR_T/M/src/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}

# --- BQL with no caller Context (register C19, ord Δ1): a bql: ORD resolved with get(base) / resolve(base),
# get() / resolve() or a null Context runs with no user, so the query sees every component, permissions included
# (BOrd.java:90-107 pass a null Context; corpus B1237 §1237.7 rule 4). Comments are stripped without reading
# string or char literals as comments. [ev: retro n4-course-drivers-bql-ords-files Δ1]
@test "NCW-bql1: BOrd.make(\"bql:...\").get(base) with one argument WARNs" {
  printf '%s\n' 'class A { BITable<?> q() {' \
    '  return (BITable<?>) BOrd.make("station:|slot:/|bql:select * from control:NumericPoint").get(Sys.getStation());' '} }' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]
  [[ "$output" == *"A.java:2"*"bql"*"Context"* ]]
}
@test "NCW-bql2: a multi-line statement ending in .get(base, null) or a bare .get() WARNs" {
  printf '%s\n' 'class A { void q() {' '  Object t = BOrd.make(' '      "bql:select name from baja:Folder")' \
    '      .get(base, null);' '  Object u = BOrd.make("bql:select *").get();' '} }' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"
  [[ "$output" == *"A.java:3"*"bql"* ]]
  [[ "$output" == *"A.java:5"*"bql"* ]]
}
@test "NCW-bql-ok: get(base, cx), resolve(base, cx).get() and a -wb part are clean" {
  printf '%s\n' 'class A { void q(Context cx) {' \
    '  Object t = BOrd.make("bql:select *").get(base, cx);' \
    '  Object u = BOrd.make("bql:select *").resolve(base, cx).get();' '} }' > "$TMPDIR_T/M/src/A.java"
  mkdir -p "$TMPDIR_T/M/src/Foo-wb/src"
  printf '%s\n' 'class W { Object t = BOrd.make("bql:select *").get(base); }' > "$TMPDIR_T/M/src/Foo-wb/src/W.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [ -z "$output" ]
}
@test "NCW-literal: a // inside a string does not hide a null-Context write after it" {
  printf '%s\n' 'class A { void f(){ String u = "http://h"; parent.set(prop, val, null); } }' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [[ "$output" == *"A.java:1"*"null-Context write"* ]]
}
@test "NCW-comment: a write in a comment is not reported" {
  printf '%s\n' 'class A {' '  // parent.set(prop, val, null);' '  /* comp.invoke(act, arg, null); */' '}' > "$TMPDIR_T/M/src/A.java"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [ -z "$output" ]
}
