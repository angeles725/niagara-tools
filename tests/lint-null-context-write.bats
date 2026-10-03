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
