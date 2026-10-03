#!/usr/bin/env bats
# Tests for lint-uberjar-api-conflict.sh — a lib both uberjar()'d and api()'d = classloader conflict.
# [ev: retro 2026-09-19-module-wiring]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-uberjar-api-conflict.sh"
  mkdir -p "$TMPDIR_T/M"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "UAC-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "UAC1: uberjar + a DIFFERENT api dep is clean" {
  printf 'dependencies {\n  api(":baja")\n  uberjar("com.opencsv:opencsv:5.7.1")\n}\n' > "$TMPDIR_T/M/x-rt.gradle.kts"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"WARN"* ]]
}
@test "UAC2: same coordinate uberjar'd AND api'd is flagged (WARN)" {
  printf 'dependencies {\n  api("com.fasterxml.jackson.core:jackson-core:2.13.1")\n  uberjar("com.fasterxml.jackson.core:jackson-core:2.13.1")\n}\n' > "$TMPDIR_T/M/x-rt.gradle.kts"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" == *"WARN"* ]]; [[ "$output" == *"jackson-core"* ]]
  # Named mutation: skip the comm -12 intersection -> UAC2 WARN vanishes.
}

@test "UAC-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass (audit A0)" {
  # Mutation: UAC-finderr -- ignoring the find exit status skips the unreadable directory's files and exits 0.
  # [ev: issue #226 R3-find-error-still-fail-open]
  mkdir -p "$TMPDIR_T/M/locked"
  printf '%s\n' 'x' > "$TMPDIR_T/M/locked/build.gradle.kts"
  chmod 000 "$TMPDIR_T/M/locked"
  if [ -r "$TMPDIR_T/M/locked" ]; then chmod 755 "$TMPDIR_T/M/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$TMPDIR_T/M"
  chmod 755 "$TMPDIR_T/M/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}
