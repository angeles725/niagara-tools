#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-clock-zero-floor.sh
# Clock rejects time <= 0; Math.max(0L, delay) is a bad floor — must be Math.max(1L, ...).
# [ev: retro 2026-09-19-gotchas-tier-a]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  CZF="$KIT/toolbelt/lint-clock-zero-floor.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "CZF-usage: no arg exits 3" { run "$CZF"; [ "$status" -eq 3 ]; }

@test "CZF1: a file with no Clock.schedule is not scanned (exit 0, silent)" {
  printf 'class A { long f(long d){ return Math.max(0L, d); } }\n' > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CZF" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "CZF2: Clock.schedule + Math.max(0L, delay) -> WARN (exit 0)" {
  printf 'class A {\n  void f(long d){ Clock.schedule(this, BRelTime.make(Math.max(0L, d)), a, null); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CZF" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-clock-zero-floor"* ]]
  # Named mutation: drop the Math.max(0 case -> CZF2 WARN vanishes.
}

@test "CZF3: Clock.schedule + Math.max(1L, delay) is clean (no WARN)" {
  printf 'class A {\n  void f(long d){ Clock.schedule(this, BRelTime.make(Math.max(1L, d)), a, null); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$CZF" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}
