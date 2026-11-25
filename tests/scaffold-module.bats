#!/usr/bin/env bats
# RED-FIRST pins for scaffold-module.sh (campaign 7 PR4, issue #45, from B794 companero MinimalPan build).
# Emits the SMALLEST correct N4 module source tree (B790/B793 recipe) so a new module starts gate-green.
#
# SURFACE (provisional — the design owns the fixture location + exact flag set; rebase if it moves, kept in one file):
#   scaffold-module.sh <ModuleName> <out-dir> --vendor <v> [--target-version 4.14] [--plugin-version 7.6.17]
#   exit: 0 ok · 2 usage OR invalid name (name must be a Java identifier starting uppercase) · 3 env
#         (out-dir not creatable, or already contains the module)
#
# RED today: build-n4-module-kit/toolbelt/scaffold-module.sh does NOT exist and the golden fixture
# build-n4-module-kit/fixtures/MinimalPan/ is not bundled yet, so TC1/TC2/TC3/TC-K8 fail for the right
# reason (tool absent). TC4 SKIPs without a niagara_home/JDK 8 — it is local-only round-trip bless
# evidence, never a fake PASS. Green once PR4 lands the emitter + the fixture.
#
# NAMED MUTATIONS (run post-green):
#   - drop the <MOD>-rt.gradle.kts emission        -> TC3 diff fails (tree differs from the fixture).
#   - emit module.xml instead of module-include.xml -> TC3 fails (B793 correction #3).
#   - drop the stopped() cancel in the emitted component -> TC4 lint-timers FAILs locally.

# The cases are split by concern: this file (arguments, the emitted trees, the round trip, signing, literal
# values, a failed emit), scaffold-module-driver.bats and scaffold-module-driver-station.bats (--type driver),
# scaffold-module-station-test.bats (the station-test overlay) and scaffold-module-map.bats (the module map).
# Shared setup: tests/lib/scaffold-setup.bash.

load lib/scaffold-setup

@test "TC1: missing args exits 2 with usage on stderr" {
  run "$SC"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage"* ]]
}

@test "TC2: an invalid module name (1bad — not an uppercase Java identifier) exits 2" {
  run "$SC" 1bad "$OUT"
  [ "$status" -eq 2 ]
}

@test "TC-V1: no --vendor exits 2 naming the option, and a version of the wrong shape exits 2, before anything is written" {
  # Mutation: TC-V1 -- a default vendor (the old defaultVendor) emits com/<default>/ and exits 0.
  run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 2 ]
  [[ "${lines[0]}" == "scaffold-module: --vendor is required: the module's vendor, also its Java package segment once lowercased (e.g. --vendor Acme)" ]]
  [[ "${lines[1]}" == "usage: scaffold-module.sh <ModuleName> <out-dir> --vendor <v> "* ]]
  run "$SC" MinimalPan "$OUT" --vendor Acme --target-version 4.99 --plugin-version banana
  [ "$status" -eq 2 ]
  [ "$output" = "scaffold-module: --plugin-version must be a plugin version <major>.<minor>.<patch>, digits only (e.g. 7.6.22): banana" ]
  run "$SC" MinimalPan "$OUT" --vendor Acme --target-version 4
  [ "$status" -eq 2 ]
  [ "$output" = "scaffold-module: --target-version must be a Niagara release <major>.<minor>, digits only (e.g. 4.15): 4" ]
  [ -z "$(ls -A "$OUT")" ]
}

# The logic fixture carries the opt-in async-offload component (--with-offload): its marker lines exist only in the
# fixture, so the round trip compares the --with-offload tree with the fixture minus them (tests/scaffold-module-offload.bats).
fixture_without_markers() {
  local copy="$BATS_TEST_TMPDIR/fixture"
  mkdir -p "$copy"
  cp -r "$FIXTURE" "$copy/"
  grep -rlE 'offload:(begin|end|line)' "$copy/MinimalPan" | xargs sed -i -e '/offload:begin/d' -e '/offload:end/d' -e 's/ <!-- offload:line -->$//'
  FIXTURE_PLAIN="$copy/MinimalPan"
}

@test "TC3: the emitted MinimalPan tree (--with-offload) byte-equals the bundled fixture minus its offload markers (diff -r, minus build/.gradle)" {
  fixture_without_markers
  run "$SC" MinimalPan "$OUT" --vendor Angeles --with-offload
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE "$OUT/MinimalPan" "$FIXTURE_PLAIN"
  [ "$status" -eq 0 ]
}

@test "TC-K8: the emitted tree is identical under HOME=/nonexistent (no \$HOME coupling)" {
  fixture_without_markers
  HOME=/nonexistent run "$SC" MinimalPan "$OUT" --vendor Angeles --with-offload
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE "$OUT/MinimalPan" "$FIXTURE_PLAIN"
  [ "$status" -eq 0 ]
}

@test "TC-DASH1: --type dashboard emits MinimalDash tree byte-equals the bundled fixture (diff -r, minus build/.gradle, the login overlay and the lines it adds)" {
  # The login overlay (templates/dashboard-login, tests/scaffold-module-login.bats) adds files to -rt and inserts
  # lines into five plumbing files: everything else is byte-equal, and an inserted file loses no fixture line.
  run "$SC" MinimalDash "$OUT" --type dashboard --vendor Angeles
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE --exclude=MinimalDash-rt --exclude=MODULE-MAP.md "$OUT/MinimalDash" "$FIXTURE_DASH"
  [ "$status" -eq 0 ]
  R=MinimalDash-rt
  run diff -r --exclude=build --exclude=.gradle $ST_EXCLUDE --exclude=login --exclude=rc --exclude=BMinimalDashService.java \
    --exclude=module-include.xml --exclude=module.palette --exclude=module.lexicon --exclude=MinimalDash-rt.gradle.kts \
    "$OUT/MinimalDash/MinimalDash/$R" "$FIXTURE_DASH/MinimalDash/$R"
  [ "$status" -eq 0 ]
  for f in MODULE-MAP.md $R/module-include.xml $R/module.palette $R/module.lexicon $R/MinimalDash-rt.gradle.kts; do
    run diff "$FIXTURE_DASH/MinimalDash/$f" "$OUT/MinimalDash/MinimalDash/$f"
    [ "$status" -eq 1 ] || { echo "$f: not changed by the login overlay"; false; }
    [[ "$output" != *$'\n<'* && "$output" != '<'* ]] || { echo "$f: a fixture line was lost"; echo "$output"; false; }
  done
}

@test "TC-DASH3: the emitted -ux servlet overrides only real BWebServlet entry points (never doService(WebOp))" {
  # BWebServlet's overridable dispatch set is service/doGet/doPost/doPut/doDelete(WebOp); an @Override of
  # doService(WebOp) failed :ModuleA-ux:compileJava on both 4.14 and 4.15 (types/dashboard.md override contract).
  # Named mutation: rename doGet back to doService in the fixture -> this test fails.
  run "$SC" ModuleA "$OUT" --type dashboard --vendor Example
  [ "$status" -eq 0 ]
  V="$OUT/ModuleA/ModuleA/ModuleA-ux/src/com/example/ModuleA/ux/route/BModuleAServlet.java"
  run grep -nE 'void doService\(WebOp' "$V"
  [ "$status" -eq 1 ]
  grep -qE 'public void doGet\(WebOp op\)' "$V"
}

@test "TC-DASH2: invalid --type exits 2" {
  run "$SC" MyModule "$OUT" --type unknown
  [ "$status" -eq 2 ]
}

@test "TC4: round-trip preflight->build.sh->verify-module ALL PASS->lint-timers PASS (SKIP without niagara_home/JDK8)" {
  [ -n "${NIAGARA_HOME:-}" ] || skip "no niagara_home/JDK 8 for the real round-trip (local-only bless evidence, never a fake PASS)"
  run "$SC" MinimalPan "$OUT" --vendor Angeles
  [ "$status" -eq 0 ]
  # D2 layout: emitted root = <out-dir>/<ModuleName>, and inside it the <Module>/<Module>-rt findProjects
  # nesting → the -rt artifact is at $OUT/MinimalPan/MinimalPan/MinimalPan-rt.
  ROOT="$OUT/MinimalPan"; RT="$ROOT/MinimalPan/MinimalPan-rt"
  # scaffold -> the emitted module builds signed (build.sh runs slotomatic first, D3) + passes the gate + the timer lint.
  # build.sh usage: <module-root> <MOD> [niagara_home]; it runs slotomatic + jar + the verify gate itself.
  # build.sh refuses a real niagara_home (no .niagara-mirror): build into a throwaway mirror, never the install.
  run bash "$KIT/toolbelt/mirror-niagara-home.sh" "$NIAGARA_HOME" "$BATS_TEST_TMPDIR/nh-mirror" MinimalPan-rt.jar
  [ "$status" -eq 0 ]
  run bash "$KIT/toolbelt/build.sh" "$ROOT" MinimalPan "$BATS_TEST_TMPDIR/nh-mirror"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ALL PASS"* ]]
  run bash "$KIT/toolbelt/lint-timers.sh" "$RT/src"
  [ "$status" -eq 0 ]
}

@test "TC-SIGN: every scaffold type emits the root signing block, so lint-structure L15 stays silent (issue #266)" {
  # Named mutation: drop the niagaraSigning { aliases; signingProfileFile } part of the block from a fixture
  # root (keep allowDefaultProfile) -> the lint exits 0 with an L15 WARN on this root, and this test fails.
  for t in logic dashboard driver library; do
    run "$SC" "Sign${t^}" "$OUT/$t" --type "$t" --vendor Example
    [ "$status" -eq 0 ]
    [[ "$output" == *"niagaraSigningAlias"* ]]
    root="$OUT/$t/Sign${t^}"
    mkdir -p "$root/.git"                      # the L10/L14 upward walk stops here, not in the test tmpdir
    grep -q 'allowDefaultProfile.set(false)' "$root/build.gradle.kts"
    run "$KIT/toolbelt/lint-structure.sh" "$root/Sign${t^}"
    [ "$status" -eq 0 ]                        # not 3 (usage/env: wrong path) and not 1 (FAIL)
    [[ "$output" != *"L15"* ]]
    # positive control: the same root without its aliases line WARNs, so the lint reached this exact root
    sed -i '/aliases\.set(/d' "$root/build.gradle.kts"
    run "$KIT/toolbelt/lint-structure.sh" "$root/Sign${t^}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"WARN  lint-structure  $root/build.gradle.kts  L15: "*"aliases"* ]]
  done
}

@test "TC-S2a: a value holding a fixture token is written literally (a later substitution never rewrites it)" {
  # Mutation: S2 -- applying the substitution rules one after another rewrites the 4.14 inside the plugin version.
  run "$SC" MinimalPan "$OUT" --vendor Angeles --plugin-version 7.4.14 --target-version 9.9
  [ "$status" -eq 0 ]
  grep -qF 'getOrElse("7.4.14")' "$OUT/MinimalPan/settings.gradle.kts"
  grep -qF 'OptimizerSupervisor 9.9 -> 7.4.14 (default)' "$OUT/MinimalPan/settings.gradle.kts"
}

@test "TC-S2b: a failed emit leaves nothing in the out-dir and the next run is not refused" {
  # Mutation: S2 -- without the discard of a failed emit the partial MinimalPan tree stays in the out-dir.
  local kit="$BATS_TEST_TMPDIR/kit" t
  mkdir -p "$kit/toolbelt" "$kit/templates"
  ln -s "$SC" "$kit/toolbelt/scaffold-module.sh"
  ln -s "$KIT/fixtures" "$kit/fixtures"
  for t in "$KIT"/templates/*; do
    [ "${t##*/}" = VERSION-LEDGER.md ] || ln -s "$t" "$kit/templates/${t##*/}"
  done
  run "$kit/toolbelt/scaffold-module.sh" MinimalPan "$OUT" --vendor Angeles
  [ "$status" -eq 3 ]
  [ "$output" = "scaffold-module: version-ledger template not found: $kit/toolbelt/../templates/VERSION-LEDGER.md
scaffold-module: removed the partial $OUT/MinimalPan" ]
  [ -z "$(ls -A "$OUT")" ]
  run "$SC" MinimalPan "$OUT" --vendor Angeles
  [ "$status" -eq 0 ]
}
