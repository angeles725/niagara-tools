#!/usr/bin/env bats
# RED-FIRST pins for lint-structure.sh (campaign 8 PR18, wave3 R18, B817). Source-tree structure lint
# over a <module-root> (iterates every profile via its module-include.xml). L8 (signed-jar) stays in
# verify-module.sh. Row `FAIL|WARN  lint-structure  <path>  L<n>: <reason>`; exit 0/1/3.
#   L1 package naming (com.<vendor>.*, never javax.baja.*)   L2 one @NiagaraType per file
#   L3 pure-model pkg has tests + no baja/tridium imports     L4 lexicon non-empty per artifact w/ >=1 type
#   L5 rt palette non-empty                                   L6 module-include.xml present/consistent
#   L7 3-part dependency version floors                       L9 no empty skeleton artifact (0 class + 0 palette)
#   L10 no absolute host paths in tracked gradle.properties   L11 mixed pure+Baja srcTest declares :test-wb AND junit
#
# RED today: lint-structure.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (R18.5): remove a dependency's 3-part floor -> L7 FAIL on that entry.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LS="$KIT/toolbelt/lint-structure.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-structure"
}

@test "LS4: chihuahua-rt with an empty module.lexicon and >=1 declared type FAILs L4, exit 1" {
  run "$LS" "$FX/chihuahua-rt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"L4"* ]]
}

@test "LS9: DashboardPan-wb with 0 classes AND an empty palette FAILs L9 (empty skeleton), exit 1" {
  run "$LS" "$FX/dashboardpan-wb"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"L9"* ]]
}

@test "LS10: a tracked gradle.properties with an absolute host path (C:\\Honeywell\\...) FAILs L10" {
  run "$LS" "$FX/client-gradleprops"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"L10"* ]]
}

@test "LS11: a mixed BTest+JUnit srcTest without both :test-wb and junit gradle decls FAILs L11" {
  run "$LS" "$FX/coldroompan-rt"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"L11"* ]]
}

@test "LS7: a dependency with a 2-part floor (:baja:4.14, not 4.14.0) FAILs L7 (mutation target)" {
  run "$LS" "$FX/deps"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"L7"* ]]
}

@test "LS-prune: an L10 violation under .deploy-baseline/ is NOT flagged (D9b), anchored on L11" {
  run "$LS" "$FX/coldroompan-rt"
  [[ "$output" == *"L11"* ]]                 # anchor: the scan ran and flagged the real L11
  [[ "$output" != *"L10"* ]]                 # the .deploy-baseline gradle.properties is pruned
}

@test "LS-usage: no <module-root> argument -> exit 3" {
  run "$LS"
  [ "$status" -eq 3 ]
}

@test "LS9-real: profile dir with build.gradle.kts but no module-include.xml fires L6+L9" {
  run "$LS" "$FX/skeleton-wb"
  [ "$status" -eq 1 ]
  [[ "$output" == *"L6"* ]]
  [[ "$output" == *"L9"* ]]
}

@test "LS10-real: gradle.properties two levels above module root fires L10" {
  local ROOT
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/.git"                                  # .git sentinel — upward walk stops here
  printf 'niagara_home=C:\\Honeywell\\Niagara-4.14.0\n' > "$ROOT/gradle.properties"
  mkdir -p "$ROOT/proj/Mod/Foo-wb"
  printf 'plugins { id("com.tridium.niagara-module") }\n' > "$ROOT/proj/Mod/Foo-wb/build.gradle.kts"
  run "$LS" "$ROOT/proj/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"L10"* ]]
}

@test "LS-PASS: scaffold-module.sh output passes L1-L11 -> exit 0 (SKIP if scaffold absent)" {
  SC="$KIT/toolbelt/scaffold-module.sh"
  [ -x "$SC" ] || skip "scaffold-module.sh not present"
  OUT="$BATS_TEST_TMPDIR/out"; mkdir -p "$OUT"
  run "$SC" MinimalPan "$OUT"
  [ "$status" -eq 0 ]
  # The scaffold emits MinimalPan/<module-subdir>/MinimalPan-rt; pass the module subdir as root
  run "$LS" "$OUT/MinimalPan/MinimalPan"
  [ "$status" -eq 0 ]                         # a scaffolded module is structurally clean (R18.4/R18.6)
}
