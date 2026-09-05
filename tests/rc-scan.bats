#!/usr/bin/env bats
# RED-FIRST pins for rc-scan.sh (campaign 8 PR6). Scans a module's browser resources
# (**/rc/** — html/js/css ONLY) for defects the Java-side lints never see, from the real
# DashboardPan-ux/src/rc/index.html [CERT-live]:
#   ord   FAIL  hardcoded `station:|` / `slot:/` ORD literal (index.html-class :701 shape)
#   host  FAIL  a hardcoded network host: `http(s)://` with an IPv4 literal, or a non-namespace
#               host. W3C SVG/xlink namespace URIs (http://www.w3.org/...) are NOT hosts.
#   bare-catch  WARN  a write fetch swallowed by `.catch(() => {})` (index.html :1298)
#   null-branch WARN  a `? null :` display branch on a process field (index.html :852-853)
#
# SURFACE: rc-scan.sh <module-root>
#   Row: `<check>  PASS|FAIL|WARN|SKIP  <subject-without-whitespace>  <detail>`
#   Exit: 0 no FAIL (WARN-only still 0) · 1 any FAIL · 3 usage. Consistent with lint-delays/triage.
#
# RED today: build-n4-module-kit/toolbelt/rc-scan.sh does not exist -> every pin fails for the
# right reason (tool absent). Green once PR6 lands the scanner.
#
# NAMED MUTATIONS (post-green):
#   - remove the ORD rule            -> RC1's ord FAIL vanishes; ord fixture is ord-only so exit 1 -> 0.
#   - drop the bare-catch WARN check -> RC3's WARN row vanishes.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  RS="$KIT/toolbelt/rc-scan.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/rc-scan"
}

@test "RC1: a hardcoded ORD literal under rc/ FAILs (ord), exit 1 (ord-only -> the clean mutation target)" {
  run "$RS" "$FX/ord"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"ord"* ]]
  [[ "$output" == *"app.js"* ]]
}

@test "RC2: an IPv4 host literal FAILs (host) but a W3C SVG namespace does NOT (false-positive control)" {
  run "$RS" "$FX/host"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"host"* ]]
  [[ "$output" == *"127.0.0.1"* ]] || [[ "$output" == *"config.js"* ]]
  # the SVG/xlink namespace URIs on the same file must NOT be reported as hosts:
  [[ "$output" != *"www.w3.org"* ]]
}

@test "RC3: a write fetch with a bare .catch(() => {}) WARNs (bare-catch), exit 0 (WARN does not fail)" {
  run "$RS" "$FX/warns"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"bare-catch"* ]]
}

@test "RC4: a ? null : display branch on a process field WARNs (null-branch)" {
  run "$RS" "$FX/warns"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"null-branch"* ]]
}

@test "RC5: a clean rc tree -> exit 0, no FAIL and no WARN" {
  run "$RS" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]] && [[ "$output" != *"WARN"* ]]
}

@test "RC6: an ORD literal in src/ (outside rc/) is NOT scanned (scope: rc/ only)" {
  run "$RS" "$FX/scope"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"Foo.java"* ]]   # src/ is never read
}

@test "RC7: no module-root argument -> exit 3 (usage)" {
  run "$RS"
  [ "$status" -eq 3 ]
}

@test "RC8: real smoke — DashboardPan-ux rc FAILs on the :701 host literal (SKIP if not present)" {
  UX="$HOME/modulos_niagara_n4/Cliente/Leon-Guanjuato/Dashboard/DashboardPan/DashboardPan-ux"
  [ -d "$UX/src/rc" ] || skip "DashboardPan-ux rc not on this machine (local-only real smoke)"
  run "$RS" "$UX"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"host"* ]]
}

# ---- RC9: D9b dot-dir prune -----------------------------------------------
# Design addendum D9b: kit source scanners prune dot-directories. report-module keeps a previous-deploy
# snapshot at <artifact>/.deploy-baseline, which can contain a stale rc/ copy; rc-scan must NOT descend
# into it (nor into .git). The live rc/ here is clean, so a clean exit proves the dot-dir was pruned.
# RED today: rc-scan.sh absent (tool-absent, like the rest of PR6). NAMED MUTATION (post-green): remove
# the dot-dir prune -> the .deploy-baseline host literal is flagged -> RC9 flips.
@test "RC9: a host literal in a .deploy-baseline/ copy is NOT flagged (dot-dir pruned; live rc/ is clean)" {
  run "$RS" "$FX/dotdir"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"config.js"* ]]   # the stale baseline copy is never read
}
