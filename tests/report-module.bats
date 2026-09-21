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

# ===========================================================================
# CAMPAIGN 10 — 9 new lints (2026-09-19). RM1-6 stay intact.
#   Per-artifact src scanners: lint-no-system-out (FAIL), lint-clock-zero-floor (WARN),
#     lint-null-context-write (WARN), lint-bql-string-concat (WARN), lint-arbitrary-ord (WARN).
#   Profile-conditional: lint-se-display (FAIL, -se only), lint-jasmine-ux (WARN, -ux only).
#   Module-root once-per-run: lint-agent-on-shape (FAIL), lint-uberjar-api-conflict (WARN).
# Fixtures: system-out, se-display, agent-on, bql-warn, ux-no-specs (see tests/fixtures/report-module/).
# Named mutations (post-green):
#   - drop System.out check → RM7 no longer exits 1.
#   - drop JFrame check → RM8 no longer exits 1.
#   - drop agent-on validation → RM9 no longer exits 1.
#   - drop BQL concat passthrough → RM10 loses WARN row.
#   - drop jasmine-ux passthrough → RM11 loses WARN row.

@test "RM7: System.out call surfaces lint-no-system-out FAIL row and exits 1" {
  # system-out/DemoPan-rt/src/com/x/BOut.java has System.out.println
  run "$RM" "$FX/system-out"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"lint-no-system-out"* ]] && [[ "$output" == *"BOut.java"* ]]
}

@test "RM8: display class import in a -se artifact surfaces lint-se-display FAIL row and exits 1" {
  # se-display/DemoPan-se/src/com/x/BPanel.java imports javax.swing.JFrame
  run "$RM" "$FX/se-display"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"lint-se-display"* ]] && [[ "$output" == *"BPanel.java"* ]]
}

@test "RM9: malformed agent-on type in module-include.xml surfaces lint-agent-on-shape FAIL row and exits 1" {
  # agent-on/DemoPan-rt/module-include.xml has <on type="FooService"/> — missing module: prefix
  run "$RM" "$FX/agent-on"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"lint-agent-on-shape"* ]] && [[ "$output" == *"module-include.xml"* ]]
}

@test "RM10: BQL string-concat produces lint-bql-string-concat WARN row but exit stays 0 (WARN does not block)" {
  # bql-warn/DemoPan-rt/src/com/x/BQuery.java concatenates into a bql: string
  run "$RM" "$FX/bql-warn"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"lint-bql-string-concat"* ]] && [[ "$output" == *"BQuery.java"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "RM11: -ux artifact with src/rc/ but no JS specs surfaces lint-jasmine-ux WARN row, exit stays 0" {
  # ux-no-specs/DemoPan-ux/src/rc/index.html exists but srcTest/rc/spec/ is absent
  run "$RM" "$FX/ux-no-specs"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"lint-jasmine-ux"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

# ===========================================================================
# NEW LINTS (static-source) — RM12-RM20 (2026-09-20).
#   Per-artifact *-rt: lint-subscribe-without-unsubscribe (WARN, RM12),
#     lint-recovery-path (FAIL, RM13), lint-config-sanity (FAIL, RM14),
#     lint-status-parity (WARN, RM15).
#   Per-artifact *-ux: lint-servlet (FAIL, RM16), rc-scan (FAIL, RM17).
#   Per-artifact *-wb: lint-wb-threading (WARN, RM18).
#   Module-once: lint-structure (FAIL, RM19), lint-write-path (FAIL, RM20).
# Named mutations (post-green):
#   - drop subscribe guard -> RM12 loses WARN row.
#   - drop recovery-path guard -> RM13 exits 0.
#   - drop config-sanity CS1 check -> RM14 exits 0.
#   - drop status-parity check -> RM15 loses WARN row.
#   - drop servlet auth check -> RM16 exits 0.
#   - drop rc-scan ord-literal check -> RM17 exits 0.
#   - drop wb-threading traversal check -> RM18 loses WARN row.
#   - drop structure L7 check -> RM19 exits 0.
#   - drop write-path OPERATOR scan -> RM20 exits 0.

@test "RM12: subscribe-without-unsubscribe surfaces WARN row in *-rt artifact, exit stays 0" {
  # subscribe-warn/DemoPan-rt/src/com/x/BSub.java calls .subscribe( with no unsubscribe
  run "$RM" "$FX/subscribe-warn"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-subscribe-without-unsubscribe"* ]]
  [[ "$output" == *"BSub.java"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "RM13: recovery-path guarded-only safe-off surfaces FAIL row and exits 1" {
  # recovery-fail/DemoPan-rt/src/com/x/BHeat.java writes ResistanceOut only inside execute()
  run "$RM" "$FX/recovery-fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-recovery-path"* ]]
  [[ "$output" == *"BHeat.java"* ]]
}

@test "RM14: config-sanity CS1 (interval<=duration) surfaces FAIL row and exits 1" {
  # config-sanity-fail/DemoPan-rt/src/com/x/BConfig.java has cycleInterval(300s) <= defrostDuration(600s)
  run "$RM" "$FX/config-sanity-fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-config-sanity"* ]]
  [[ "$output" == *"BConfig.java"* ]]
}

@test "RM15: status-parity asymmetric facade surfaces WARN row in *-rt artifact, exit stays 0" {
  # status-parity-warn/DemoPan-rt/src/com/x/BFacade.java has 2 Interval slots, 0 status slots
  run "$RM" "$FX/status-parity-warn"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-status-parity"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "RM16: servlet auth violation surfaces FAIL row in *-ux artifact and exits 1" {
  # servlet-fail/DemoPan-ux/src/com/x/BMyServlet.java extends BWebServlet and writes without auth gate
  run "$RM" "$FX/servlet-fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"BMyServlet.java"* ]]
}

@test "RM17: hardcoded ORD in rc/ surfaces rc-scan FAIL row in *-ux artifact and exits 1" {
  # rc-scan-fail/DemoPan-ux/src/rc/index.html has station:|slot:/ hardcoded ORD
  run "$RM" "$FX/rc-scan-fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"rc-scan"* ]]
}

@test "RM18: doInvoke getNavChildren without invokeLater surfaces WARN row in *-wb artifact, exit stays 0" {
  # wb-threading-warn/DemoPan-wb/src/com/x/BPanel.java calls getNavChildren in doInvoke
  run "$RM" "$FX/wb-threading-warn"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"ui-thread-traversal"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "RM19: 2-part dependency version surfaces lint-structure FAIL row and exits 1" {
  # structure-fail/DemoPan-rt/DemoPan-rt.gradle.kts has api(\":baja:4.14\") — L7 violation
  run "$RM" "$FX/structure-fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-structure"* ]]
}

@test "RM20: uncovered OPERATOR slot surfaces lint-write-path FAIL row and exits 1" {
  # write-path-fail/DemoPan-rt/src/com/x/BControl.java has OPERATOR property \"setpoint\" not in matrix
  run "$RM" "$FX/write-path-fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-write-path"* ]]
}
