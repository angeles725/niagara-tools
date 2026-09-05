#!/usr/bin/env bats
# RED-FIRST pins for scaffold-module.sh (campaign 7 PR4, issue #45, from B794 companero MinimalPan build).
# Emits the SMALLEST correct N4 module source tree (B790/B793 recipe) so a new module starts gate-green.
#
# SURFACE (provisional — the design owns the fixture location + exact flag set; rebase if it moves, kept in one file):
#   scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version 4.14] [--plugin-version 7.6.17]
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

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SC="$KIT/toolbelt/scaffold-module.sh"
  FIXTURE="$KIT/fixtures/MinimalPan"
  OUT="$BATS_TEST_TMPDIR/out"
  mkdir -p "$OUT"
}

@test "TC1: missing args exits 2 with usage on stderr" {
  run "$SC"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage"* ]]
}

@test "TC2: an invalid module name (1bad — not an uppercase Java identifier) exits 2" {
  run "$SC" 1bad "$OUT"
  [ "$status" -eq 2 ]
}

@test "TC3: the emitted MinimalPan tree byte-equals the bundled fixture (diff -r, minus build/.gradle)" {
  run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle "$OUT/MinimalPan" "$FIXTURE"
  [ "$status" -eq 0 ]
}

@test "TC-K8: the emitted tree is identical under HOME=/nonexistent (no \$HOME coupling)" {
  HOME=/nonexistent run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  run diff -r --exclude=build --exclude=.gradle "$OUT/MinimalPan" "$FIXTURE"
  [ "$status" -eq 0 ]
}

@test "TC4: round-trip preflight->build.sh->verify-module ALL PASS->lint-timers PASS (SKIP without niagara_home/JDK8)" {
  [ -n "${NIAGARA_HOME:-}" ] || skip "no niagara_home/JDK 8 for the real round-trip (local-only bless evidence, never a fake PASS)"
  run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  # D2 layout: emitted root = <out-dir>/<ModuleName>, and inside it the <Module>/<Module>-rt findProjects
  # nesting → the -rt artifact is at $OUT/MinimalPan/MinimalPan/MinimalPan-rt.
  ROOT="$OUT/MinimalPan"; RT="$ROOT/MinimalPan/MinimalPan-rt"
  # scaffold -> the emitted module builds signed (build.sh runs slotomatic first, D3) + passes the gate + the timer lint.
  # build.sh usage: <module-root> <MOD> [niagara_home]; it runs slotomatic + jar + the verify gate itself.
  run bash "$KIT/toolbelt/build.sh" "$ROOT" MinimalPan "$NIAGARA_HOME"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ALL PASS"* ]]
  run bash "$KIT/toolbelt/lint-timers.sh" "$RT/src"
  [ "$status" -eq 0 ]
}
