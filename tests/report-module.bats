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

@test "RM21: MD5+credential surfaces lint-no-md5-credential-digest WARN row but exit stays 0 (WARN does not block)" {
  # md5-cred-warn/DemoPan-rt/src/com/x/BAuth.java uses MessageDigest.getInstance("MD5") + BPassword
  run "$RM" "$FX/md5-cred-warn"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"lint-no-md5-credential-digest"* ]] && [[ "$output" == *"BAuth.java"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "RM22: -wb artifact with dialog(...BOrd.NULL) and no chooser surfaces lint-wb-file-chooser WARN row, exit stays 0" {
  # wb-file-chooser-warn/DemoPan-wb/src/com/x/BChooser.java calls dialog(BOrd.NULL) with no BComponentChooser/targetType
  run "$RM" "$FX/wb-file-chooser-warn"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-wb-file-chooser"* ]]
  [[ "$output" == *"BChooser.java"* ]]
  [[ "$output" == *"CLEAN"* ]]
}

@test "RM23: bundled jar containing a major-53 class surfaces lint-bundled-jar-class-version FAIL row and exits 1" {
  if ! command -v zip >/dev/null 2>&1; then skip "zip not available"; fi
  # Build a temporary module with a bundled ext-jar that contains a Java 9 class (major = 53)
  local tmpmod
  tmpmod="$(mktemp -d)"
  mkdir -p "$tmpmod/DemoPan-rt/libs"
  printf '\xca\xfe\xba\xbe\x00\x00\x00\x35\x00\x00' > "$tmpmod/Foo9.class"
  (cd "$tmpmod" && zip -q DemoPan-rt/libs/java9.jar Foo9.class)
  rm "$tmpmod/Foo9.class"
  run "$RM" "$tmpmod"
  rm -rf "$tmpmod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-bundled-jar-class-version"* ]]
  # Named mutation: drop the major > 52 check -> RM23 exits 0 (no FAIL row).
}

# ===========================================================================
# WU4 (fold-2026-10-02-pending-retros) — frontend tooling in the aggregated report.
#   --profile <hmi|lan|both|unknown> and --legacy pass through to rc-scan.sh (RM24-RM26);
#   ESLint (toolbelt/eslint.config.mjs) rows relayed for *-ux src/rc js (RM27-RM29);
#   lint-vendor-floor.sh relayed for *-ux src/rc/vendor (RM30-RM31).
#   A missing tool (eslint / node / acorn) is a visible SKIP row naming it, never a silent pass
#   and never an env fault for the whole report.
# [ev: retro dashboard-frontend-standard Δ10] [ev: retro dashboard-frontend-standard Δ16]
# Named mutations (post-green):
#   - drop the --profile pass-through -> RM24 exits 0 (browser-floor stays WARN).
#   - drop the --legacy pass-through -> RM25 exits 1 (datauri-budget stays FAIL).
#   - relay no eslint rows -> RM27 exits 0.

# _ux_tree <dir> — a copy of the clean ux-no-specs artifact; the caller replaces/adds files under src/rc.
_ux_tree() { mkdir -p "$1"; cp -R "$FX/ux-no-specs/DemoPan-ux" "$1/"; }

@test "RM24: --profile hmi is passed to rc-scan -> a browser-floor row becomes FAIL (exit 1); default stays WARN (exit 0)" {
  local t="$BATS_TEST_TMPDIR/rm24"; _ux_tree "$t"
  printf '<!DOCTYPE html><html><head><style>\n.bg { inset: 0; }\n</style></head><body></body></html>\n' \
    > "$t/DemoPan-ux/src/rc/index.html"
  KIT_ESLINT=/nonexistent run "$RM" "$t"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  rc-scan  index.html:2  browser-floor"* ]]
  KIT_ESLINT=/nonexistent run "$RM" "$t" --profile hmi
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rc-scan  index.html:2  browser-floor"* ]]
}

@test "RM25: --legacy is passed to rc-scan -> an over-budget data URI is WARN (exit 0); default FAIL (exit 1)" {
  local t="$BATS_TEST_TMPDIR/rm25"; _ux_tree "$t"
  local big; big=$(head -c 21000 /dev/zero | tr '\0' 'A')
  printf '<!DOCTYPE html><html><body><img src="data:image/png;base64,%s"></body></html>\n' "$big" \
    > "$t/DemoPan-ux/src/rc/index.html"
  KIT_ESLINT=/nonexistent run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"datauri-budget"* ]]
  KIT_ESLINT=/nonexistent run "$RM" "$t" --legacy
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"datauri-budget"* ]]
}

@test "RM26: an unknown --profile value -> usage exit 2" {
  run "$RM" "$FX/clean" --profile kiosk
  [ "$status" -eq 2 ]
}

@test "RM27: eslint rows are relayed for *-ux rc js (stub eslint): FAIL row -> exit 1" {
  local t="$BATS_TEST_TMPDIR/rm27"; _ux_tree "$t"
  mkdir -p "$t/DemoPan-ux/src/rc/js"
  printf 'var a = 1;\n' > "$t/DemoPan-ux/src/rc/js/app.js"
  local stub="$BATS_TEST_TMPDIR/eslint-stub"
  printf '#!/usr/bin/env bash\nprintf "FAIL  eslint  js/app.js:3  eqeqeq: Expected ===\\n"\nprintf "WARN  eslint  js/app.js:9  max-lines-per-function: too long\\n"\nexit 1\n' > "$stub"
  chmod +x "$stub"
  KIT_ESLINT="$stub" run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"DemoPan-ux  FAIL  eslint  app.js:3  eqeqeq"* ]]
  [[ "$output" == *"DemoPan-ux  WARN  eslint  app.js:9  max-lines-per-function"* ]]
}

@test "RM28: eslint unavailable -> one SKIP eslint row naming it; the report is not an env fault" {
  local t="$BATS_TEST_TMPDIR/rm28"; _ux_tree "$t"
  mkdir -p "$t/DemoPan-ux/src/rc/js"
  printf 'var a = 1;\n' > "$t/DemoPan-ux/src/rc/js/app.js"
  KIT_ESLINT=/nonexistent/eslint run "$RM" "$t"
  [ "$status" -eq 0 ]
  [[ "$output" == *"DemoPan-ux  SKIP  eslint  unavailable"* ]]
}

@test "RM29: a *-ux rc with no own js (vendor only) -> SKIP eslint 'no rc js'" {
  local t="$BATS_TEST_TMPDIR/rm29"; _ux_tree "$t"
  printf '<!DOCTYPE html><html><body></body></html>\n' > "$t/DemoPan-ux/src/rc/index.html"
  KIT_ESLINT=/nonexistent run "$RM" "$t"
  [[ "$output" == *"DemoPan-ux  SKIP  eslint  no rc js"* ]]
}

@test "RM30: vendor-floor tool unavailable (acorn forced missing) -> SKIP vendor-floor row, exit 0" {
  command -v node >/dev/null 2>&1 || skip "node not installed"
  local t="$BATS_TEST_TMPDIR/rm30"; _ux_tree "$t"
  mkdir -p "$t/DemoPan-ux/src/rc/vendor"
  printf 'var lib = 1;\n' > "$t/DemoPan-ux/src/rc/vendor/lib-1.0.0.min.js"
  KIT_ESLINT=/nonexistent KIT_ACORN=/nonexistent/acorn run "$RM" "$t"
  [ "$status" -eq 0 ]
  [[ "$output" == *"DemoPan-ux  SKIP  vendor-floor"* ]] && [[ "$output" == *"acorn"* ]]
}

@test "RM31: a vendored lib above the ES2020 floor -> vendor-floor FAIL row relayed, exit 1" {
  command -v node >/dev/null 2>&1 || skip "node not installed"
  if [ -z "${KIT_ACORN:-}" ] && [ ! -d "$KIT/toolbelt/eslint/node_modules/acorn" ]; then
    node -e 'require.resolve("acorn")' >/dev/null 2>&1 || skip "acorn not installed"
  fi
  local t="$BATS_TEST_TMPDIR/rm31"; _ux_tree "$t"
  mkdir -p "$t/DemoPan-ux/src/rc/vendor"
  printf 'var c = {};\nc.t ??= 1;\n' > "$t/DemoPan-ux/src/rc/vendor/bad-1.0.0.min.js"
  KIT_ESLINT=/nonexistent run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"DemoPan-ux  FAIL  vendor-floor  bad-1.0.0.min.js:2"* ]]
}

@test "RM32: the kit ESLint config loads and pins the Chromium 83 floor + the Δ10 rule set (node import)" {
  command -v node >/dev/null 2>&1 || skip "node not installed"
  run node --input-type=module -e '
    const c = (await import(process.argv[1])).default;
    const m = c.find((o) => o.rules);
    console.log([m.languageOptions.ecmaVersion, m.languageOptions.sourceType,
      m.rules["max-lines-per-function"][1].max, m.rules["no-console"][1].allow.join(),
      m.rules.eqeqeq[0], m.rules["no-unused-vars"][0], c[0].ignores.join()].join("|"));
  ' "$KIT/toolbelt/eslint.config.mjs"
  [ "$status" -eq 0 ]
  [ "$output" = "2020|script|60|error|error|warn|**/vendor/**,**/ext/**,**/*.min.js" ]
}

@test "RM33: real ESLint end-to-end (when installed): == and ES2021 syntax in rc js -> FAIL rows, vendor ignored" {
  local bin="${KIT_ESLINT:-}"
  [ -n "$bin" ] || { [ -x "$KIT/toolbelt/eslint/node_modules/.bin/eslint" ] && bin="$KIT/toolbelt/eslint/node_modules/.bin/eslint"; }
  [ -n "$bin" ] || bin="$(command -v eslint 2>/dev/null || true)"
  [ -n "$bin" ] && [ -x "$bin" ] || skip "eslint not installed (npm install --prefix build-n4-module-kit/toolbelt/eslint)"
  local t="$BATS_TEST_TMPDIR/rm33"; _ux_tree "$t"
  mkdir -p "$t/DemoPan-ux/src/rc/js" "$t/DemoPan-ux/src/rc/vendor"
  printf 'function f(x) {\n  return x == 1;\n}\nf(1);\n' > "$t/DemoPan-ux/src/rc/js/app.js"
  printf 'var c = {};\nc.t ??= 1;\n' > "$t/DemoPan-ux/src/rc/js/floor.js"
  printf 'var v = 1 == 1;\n' > "$t/DemoPan-ux/src/rc/vendor/lib.js"
  KIT_ESLINT="$bin" KIT_ACORN=/nonexistent/acorn run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"DemoPan-ux  FAIL  eslint  app.js:2  eqeqeq"* ]]
  [[ "$output" == *"DemoPan-ux  FAIL  eslint  floor.js:2  parse"* ]]
  [[ "$output" != *"lib.js"* ]]
}

# ===========================================================================
# polish-2026-10-02 P1 — report-module wiring owed since fold WU6b (#199 "WU4 tools").
#   RM34  lint-link-target-flags is a per-artifact member: a READONLY link-in target (comment-block
#         convention) -> FAIL row relayed, exit 1. [ev: retro panccadia-commissioning-lessons Δ1]
#   RM35  the module wiring map is passed as --wiring-map: auto-discovered at <module-root>/../docs/
#         wiring-map.md (the build.sh layout: <repo>/docs + <repo>/<MOD>) -> Table 2 READONLY FAILs.
#   RM36  --wiring-map <file> explicit; a missing file -> exit 3 (env), like the other file inputs.
#   RM37  a clean artifact gets one PASS lint-link-target-flags "clean" row.
#   RM38  slot-coverage's facade FAIL (a *Panel type with no lexicon key) is a FAIL row, exit 1 — it
#         was mapped to WARN by percentage. [ev: retro panccadia-commissioning-lessons Δ9]
#   RM39  lint-silent-protection ADVISORY console-only rows are relayed with their own ADVISORY
#         severity: counted in the summary apart from PASS/WARN/FAIL, verdict stays CLEAN, exit 0.
#         [ev: retro alarm-console-design Δ3]
# NAMED MUTATIONS (observed): RM-ltf (the lint-link-target-flags member output dropped) -> RM34 flips;
#   RM-map (drop the --wiring-map pass-through) -> RM35 flips; RM-facade (drop the slot-coverage FAIL
#   relay) -> RM38 flips; RM-advisory (drop the ADVISORY relay) -> RM39 flips.
# ===========================================================================
LTF_FX="$BATS_TEST_DIRNAME/fixtures/lint-link-target-flags"

# _rt_tree <dir> — a copy of the clean rt-only module tree (one DemoPan-rt artifact)
_rt_tree() { mkdir -p "$1"; cp -R "$FX/clean/DemoPan-rt" "$1/"; }

@test "RM34: a READONLY link-in target (link comment) -> FAIL lint-link-target-flags row, exit 1" {
  local t="$BATS_TEST_TMPDIR/rm34"; _rt_tree "$t"
  cp "$LTF_FX/comment/src/com/x/BRoomPanel.java" "$t/DemoPan-rt/src/com/x/"
  run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"DemoPan-rt  FAIL  lint-link-target-flags  BRoomPanel.java:"*"LTF1"* ]]
}

@test "RM35: the wiring map beside the module (<root>/../docs/wiring-map.md) is passed -> Table 2 READONLY FAILs" {
  local repo="$BATS_TEST_TMPDIR/rm35" t="$BATS_TEST_TMPDIR/rm35/DemoPan"; _rt_tree "$t"
  cp "$LTF_FX/map/src/com/x/BRoomPanel.java" "$t/DemoPan-rt/src/com/x/"
  run "$RM" "$t"
  [ "$status" -eq 0 ]                                   # no map yet: the mirror is not a known target
  [[ "$output" != *"LTF1"* ]]
  mkdir -p "$repo/docs"; cp "$LTF_FX/map/docs/wiring-map.md" "$repo/docs/"
  run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-link-target-flags  BRoomPanel.java:"*"wiring-map Table 2"* ]]
}

@test "RM36: --wiring-map <file> is forwarded; a missing --wiring-map file -> exit 3" {
  local t="$BATS_TEST_TMPDIR/rm36"; _rt_tree "$t"
  cp "$LTF_FX/map/src/com/x/BRoomPanel.java" "$t/DemoPan-rt/src/com/x/"
  run "$RM" "$t" --wiring-map "$LTF_FX/map/docs/wiring-map.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *"wiring-map Table 2"* ]]
  run "$RM" "$t" --wiring-map "$BATS_TEST_TMPDIR/nope.md"
  [ "$status" -eq 3 ]
}

@test "RM37: a clean artifact -> one PASS lint-link-target-flags clean row" {
  run "$RM" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" == *"DemoPan-rt  PASS  lint-link-target-flags  clean"* ]]
}

@test "RM38: a *Panel type with no lexicon key -> FAIL slot-coverage facade row, exit 1" {
  local t="$BATS_TEST_TMPDIR/rm38"; _rt_tree "$t"
  printf '<types>\n  <type class="com.x.BFoo" name="Foo"/>\n  <type class="com.x.BRoomPanel" name="RoomPanel"/>\n</types>\n' \
    > "$t/DemoPan-rt/module-include.xml"
  run "$RM" "$t"
  [ "$status" -eq 1 ]
  [[ "$output" == *"DemoPan-rt  FAIL  slot-coverage  facade type without lexicon: RoomPanel"* ]]
}

@test "RM39: a console-only ADVISORY row is relayed as ADVISORY, counted apart, verdict CLEAN, exit 0" {
  local t="$BATS_TEST_TMPDIR/rm39"; _rt_tree "$t"
  cat > "$t/DemoPan-rt/src/com/x/CompressorControl.java" <<'JAVA'
package com.x;
public class CompressorControl {
  int step(int target, int onCount, double suction, double suctionLowLimit, boolean suctionValid) {
    if (suctionValid && suction < suctionLowLimit) target = Math.min(target, onCount - 1); // LP floor shed (trip)
    return target;
  }
}
JAVA
  cat > "$t/DemoPan-rt/src/com/x/BCompressorControl.java" <<'JAVA'
package com.x;
public class BCompressorControl extends BComponent implements BIAlarmSource {
  void raise() { alarmSupport.newOffnormalAlarm(mkData()); }
}
JAVA
  run "$RM" "$t"
  [ "$status" -eq 0 ]
  [[ "$output" == *"DemoPan-rt  ADVISORY  lint-silent-protection  CompressorControl.java:"*"console-only:"* ]]
  [[ "$output" != *"WARN  lint-silent-protection"* ]]
  [[ "$output" == *"· 1 ADVISORY  ->  CLEAN"* ]]
}
