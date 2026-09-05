#!/usr/bin/env bats
# RED-FIRST pins for report-module.sh (campaign 7 PR8 stretch, issue #49, B798 baseline · companero).
# One aggregated read-only conformance report composing the campaign-6 toolbelt (verify-module --src,
# slot-coverage parse + dup-keys, lint-timers, --plano when a ux index.html exists) over every profile
# artifact under <module-root>. Invents no new check.
# Contract: openspec/changes/build-n4-module-campaign7/report-module-contract.md.
#
# SURFACE: report-module.sh <module-root> [--target-version 4.14]
#   rows: `<artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>`; severity: verify/lint/plano FAIL→FAIL,
#   dup-keys>0→FAIL, slot-coverage<100%→WARN; a `report-module: N artifacts · … -> CLEAN|ISSUES` summary.
#   exit 0 clean (zero FAIL) / 1 any FAIL / 3 env.
#
# RED today: report-module.sh does not exist -> all three pins fail for the right reason (tool absent).
# The exact ColdRoomPan-rt B798 report is LOCAL bless evidence, not a CI pin (per the lead) — I run it at
# apply time on the real tree.
#
# NAMED MUTATIONS (post-green):
#   - aggregation drops sub-tool FAILs -> RM2 exits 0 (the BLeak lint FAIL no longer surfaces).
#   - --plano always-run (not gated on index.html) -> RM3 sees a plano row on an rt-only tree.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  RM="$KIT/toolbelt/report-module.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/report-module"
}

@test "RM1: a clean rt-only tree -> exit 0 with a CLEAN summary (no FAIL)" {
  run "$RM" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" == *"report-module:"* ]] && [[ "$output" == *"CLEAN"* ]]
}

@test "RM2: a timer-leak class surfaces the lint-timers FAIL row and exits 1" {
  run "$RM" "$FX/leak"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"timer-ticket"* ]] && [[ "$output" == *"BLeak"* ]]
}

@test "RM3: no --plano row on an rt-only tree (the check is gated on a ux src/rc/index.html)" {
  run "$RM" "$FX/clean"
  # Anchor on the tool actually running (exit 0 + summary) so a NEGATIVE-only assert can't pass
  # for the wrong reason (tool absent -> empty output -> trivially "no plano"). RED now, and the
  # --plano-always-run mutation makes a plano row appear -> this flips.
  [ "$status" -eq 0 ]
  [[ "$output" == *"report-module:"* ]]
  [[ "$output" != *"plano"* ]]
}
