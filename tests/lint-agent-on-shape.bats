#!/usr/bin/env bats
# Tests for lint-agent-on-shape.sh — <on type> must be module:Type shaped.
# [ev: retro 2026-09-19-module-wiring]
setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-agent-on-shape.sh"
  mkdir -p "$TMPDIR_T/M/Mod-rt"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "AOS-usage: no arg exits 3" { run "$L"; [ "$status" -eq 3 ]; }

@test "AOS1: a well-formed module:Type on-target is clean" {
  printf '<types><type class="c.X" name="X"><agent><on type="mymod:MyService"/></agent></type></types>\n' \
    > "$TMPDIR_T/M/Mod-rt/module-include.xml"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 0 ]; [[ "$output" != *"FAIL"* ]]
}
@test "AOS2: a missing module prefix is flagged (FAIL)" {
  printf '<types><type class="c.X" name="X"><agent><on type="MyService"/></agent></type></types>\n' \
    > "$TMPDIR_T/M/Mod-rt/module-include.xml"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 1 ]; [[ "$output" == *"FAIL"* ]]; [[ "$output" == *"lint-agent-on-shape"* ]]
  # Named mutation: relax the module:Type regex -> AOS2 FAIL vanishes.
}
@test "AOS3: an empty type value is flagged (FAIL)" {
  printf '<types><type name="X"><agent><on type="mymod:"/></agent></type></types>\n' \
    > "$TMPDIR_T/M/Mod-rt/module-include.xml"
  run "$L" "$TMPDIR_T/M"; [ "$status" -eq 1 ]; [[ "$output" == *"FAIL"* ]]
}
