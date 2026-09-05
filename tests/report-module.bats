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

# ===========================================================================
# CAMPAIGN 8 PR8 — three new member integrations. RM1-3 (B798 jar-mode + --src) stay intact.
# Row grammar unchanged: `<artifact>  PASS|FAIL|WARN|SKIP  <check>  <detail>`.
#   lint-delays     FAIL/WARN passthrough (a delay with no >0 floor) — new per-artifact member.
#   triage-console  runs on --console-dir <dir>; a SKIP row when the flag is absent.
#   schema-risk     one explicit row `<artifact>  PASS|WARN|FAIL  schema-risk  verdict=<V>` using
#                   <artifact>/.deploy-baseline as before-dir and the current artifact as after-dir
#                   (design D9/D9a, design.md:209-211). Exit map: 0->PASS(SAFE), 1->WARN(LOSSY),
#                   2->FAIL(OUTAGE), 3/4->ERROR env row. Absent .deploy-baseline -> SKIP.
#
# RED today: report-module.sh integrates none of these -> each pin fails for the right reason
# (member absent; --console-dir is an unknown flag today -> usage exit 2). NAMED MUTATIONS
# (post-green): drop each aggregation -> its row vanishes and the exit flips.
#
# NOTE: --console-dir is the lead's given surface. schema-risk uses NO flag (design D9/D9a).
# QA FLAG for the impl: .deploy-baseline lives INSIDE the artifact, so schema-risk's recursive
# `find <after-dir> -name '*.java'` (and every member scanning the artifact) would also pick up the
# baseline's .java — the impl MUST exclude the .deploy-baseline dot-dir, else the after snapshot
# carries two 'level' slots. This fixture is built expecting that exclusion.

@test "RM4: a delay with no >0 floor surfaces the lint-delays FAIL row and exits 1 (isolated from lint-timers)" {
  # BDefrost cancels in stopped() -> timer-ticket PASSES, so only the lint-delays 'delay' check FAILs.
  run "$RM" "$FX/delays"
  [ "$status" -eq 1 ]
  [[ "$output" == *"delay"* ]] && [[ "$output" == *"BDefrost"* ]]
}

@test "RM5a: without --console-dir, report-module emits a triage-console SKIP row (exit stays 0 on a clean tree)" {
  run "$RM" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" == *"SKIP"* ]] && [[ "$output" == *"triage-console"* ]]
}

@test "RM5b: with --console-dir, report-module runs triage-console and surfaces the own-frame trace" {
  run "$RM" "$FX/clean" --console-dir "$FX/console-dir"
  [[ "$output" == *"triage-console"* ]]
  [[ "$output" == *"time <= 0"* ]] || [[ "$output" == *"BDefrost"* ]]
}

@test "RM6a: an artifact with no .deploy-baseline -> a schema-risk SKIP row (exit stays 0 on a clean tree)" {
  run "$RM" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" == *"SKIP"* ]] && [[ "$output" == *"schema-risk"* ]]
}

@test "RM6b: a .deploy-baseline retype (OUTAGE) maps to a FAIL schema-risk row (exit 1), NEVER ERROR" {
  # schema-outage/DemoPan-rt carries a .deploy-baseline where 'level' was a double; the current src
  # retypes it to baja:String -> schema-risk exit 2 (OUTAGE). No flag: the baseline is discovered.
  run "$RM" "$FX/schema-outage"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"schema-risk"* ]]
  [[ "$output" == *"verdict=OUTAGE"* ]] || [[ "$output" == *"OUTAGE"* ]]
  [[ "$output" != *"ERROR"* ]]     # exit 2 (OUTAGE) is a finding, not an env fault
}
