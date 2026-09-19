#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-jasmine-ux.sh
# A -ux module shipping src/rc but no JS specs passes the gate with zero browser tests.
# [ev: retro 2026-09-19-observability]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  JUX="$KIT/toolbelt/lint-jasmine-ux.sh"
  mkdir -p "$TMPDIR_T/Mod-ux"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "JUX-usage: no arg exits 3" { run "$JUX"; [ "$status" -eq 3 ]; }

@test "JUX1: a module with no src/rc is not applicable (exit 0, silent)" {
  run "$JUX" "$TMPDIR_T/Mod-ux"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "JUX2: src/rc present but no srcTest/rc/spec/*.js -> WARN (exit 0)" {
  mkdir -p "$TMPDIR_T/Mod-ux/src/rc"
  printf '<html></html>\n' > "$TMPDIR_T/Mod-ux/src/rc/index.html"
  run "$JUX" "$TMPDIR_T/Mod-ux"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-jasmine-ux"* ]]
  # Named mutation: drop the empty-specs check -> JUX2 WARN vanishes.
}

@test "JUX3: src/rc + a spec .js present -> clean (no WARN)" {
  mkdir -p "$TMPDIR_T/Mod-ux/src/rc" "$TMPDIR_T/Mod-ux/srcTest/rc/spec"
  printf '<html></html>\n' > "$TMPDIR_T/Mod-ux/src/rc/index.html"
  printf 'describe("x",function(){});\n' > "$TMPDIR_T/Mod-ux/srcTest/rc/spec/xSpec.js"
  run "$JUX" "$TMPDIR_T/Mod-ux"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}
