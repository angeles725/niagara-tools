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

# ---------------------------------------------------------------------------
# L12 — absent/commented org.gradle.java.installations block -> FAIL
# Named mutation: LS7 -- remove the L12 check -> LS12 exits 0
# ---------------------------------------------------------------------------
@test "LS12: gradle.properties missing org.gradle.java.installations block -> FAIL L12" {
  ROOT=$(mktemp -d)
  # Minimal module root with a profile and gradle.properties WITHOUT the JDK installation block
  mkdir -p "$ROOT/Foo-rt/src"
  printf 'niagara_home=/opt/niagara\n' > "$ROOT/gradle.properties"
  printf '<module><type name="Foo"/></module>\n' > "$ROOT/Foo-rt/module-include.xml"
  printf 'test\n' > "$ROOT/Foo-rt/module.lexicon"
  printf 'test\n' > "$ROOT/Foo-rt/module.palette"
  run "$LS" "$ROOT"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"L12"* ]]
}

@test "LS12-clean: gradle.properties with both installations keys -> no L12" {
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/Foo-rt/src"
  printf 'org.gradle.java.installations.paths=/usr/lib/jvm/java-8\norg.gradle.java.installations.auto-detect=false\n' \
    > "$ROOT/gradle.properties"
  printf '<module><type name="Foo"/></module>\n' > "$ROOT/Foo-rt/module-include.xml"
  printf 'test\n' > "$ROOT/Foo-rt/module.lexicon"
  printf 'test\n' > "$ROOT/Foo-rt/module.palette"
  run "$LS" "$ROOT"
  rm -rf "$ROOT"
  [[ "$output" != *"L12"* ]]
}

# ---------------------------------------------------------------------------
# L13 — divergent niagara_home across sibling gradle.properties -> FAIL
# Named mutation: LS7 -- remove the L13 check -> LS13 exits 0
# ---------------------------------------------------------------------------
@test "LS13: >=2 sibling gradle.properties with divergent niagara_home -> FAIL L13" {
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/ModA/Foo-rt" "$ROOT/ModB/Bar-rt"
  # Both siblings under ROOT/group-parent/Module
  GROUP="$ROOT/group-parent"
  mkdir -p "$GROUP/ModA/Foo-rt" "$GROUP/ModB/Bar-rt"
  printf 'org.gradle.java.installations.paths=/jdk\norg.gradle.java.installations.auto-detect=false\nniagara_home=/opt/niagara-A\n' \
    > "$GROUP/ModA/gradle.properties"
  printf 'org.gradle.java.installations.paths=/jdk\norg.gradle.java.installations.auto-detect=false\nniagara_home=/opt/niagara-B\n' \
    > "$GROUP/ModB/gradle.properties"
  printf '<module><type name="Foo"/></module>\n' > "$GROUP/ModA/Foo-rt/module-include.xml"
  printf 'test\n' > "$GROUP/ModA/Foo-rt/module.lexicon"
  printf 'test\n' > "$GROUP/ModA/Foo-rt/module.palette"
  # lint-structure takes ModA as the module root; looks one level up (group-parent) for siblings
  run "$LS" "$GROUP/ModA"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"L13"* ]]
}

@test "LS13-clean: sibling gradle.properties all share the same niagara_home -> no L13" {
  ROOT=$(mktemp -d)
  GROUP="$ROOT/group-parent"
  mkdir -p "$GROUP/ModA/Foo-rt" "$GROUP/ModB"
  SAME_HOME='/opt/niagara-same'
  printf 'org.gradle.java.installations.paths=/jdk\norg.gradle.java.installations.auto-detect=false\nniagara_home=%s\n' \
    "$SAME_HOME" > "$GROUP/ModA/gradle.properties"
  printf 'niagara_home=%s\n' "$SAME_HOME" > "$GROUP/ModB/gradle.properties"
  printf '<module><type name="Foo"/></module>\n' > "$GROUP/ModA/Foo-rt/module-include.xml"
  printf 'test\n' > "$GROUP/ModA/Foo-rt/module.lexicon"
  printf 'test\n' > "$GROUP/ModA/Foo-rt/module.palette"
  run "$LS" "$GROUP/ModA"
  rm -rf "$ROOT"
  [[ "$output" != *"L13"* ]]
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

# ---------------------------------------------------------------------------
# L14 — settings.gradle.kts hardcodes gradlePluginVersion (no -PniagaraPluginVersion override)
# [ev: retro panccadia-restart-seq-comp-lockout-hours Δ2]
# Named mutation: LS14 -- drop the L14 check -> a hardcoded literal pin passes (LS14 exits 0)
# ---------------------------------------------------------------------------
_ls14_tree() {
  # <root>/.git sentinel, <root>/Group/settings.gradle.kts, module root <root>/Group/Mod/Foo-rt
  mkdir -p "$1/.git" "$1/Group/Mod/Foo-rt/src"
  printf '%s\n' "$2" > "$1/Group/settings.gradle.kts"
  printf 'plugins { id("com.tridium.niagara-module") }\n' > "$1/Group/Mod/Foo-rt/build.gradle.kts"
  printf '<types><type name="Foo" class="com.x.Foo"/></types>\n' > "$1/Group/Mod/Foo-rt/module-include.xml"
  printf 'Foo.displayName=Foo\n' > "$1/Group/Mod/Foo-rt/module.lexicon"
  printf '<p n="Foo" t="x:Foo"/>\n' > "$1/Group/Mod/Foo-rt/module.palette"
}

@test "LS14: settings.gradle.kts with a hardcoded gradlePluginVersion literal FAILs L14" {
  ROOT=$(mktemp -d)
  _ls14_tree "$ROOT" '  val gradlePluginVersion: String = "7.6.17"'
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"L14"* ]]
  [[ "$output" == *"settings.gradle.kts"* ]]
}

@test "LS14-clean: providers.gradleProperty(\"niagaraPluginVersion\").getOrElse(...) -> no L14" {
  ROOT=$(mktemp -d)
  _ls14_tree "$ROOT" '  val gradlePluginVersion: String = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")'
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]                         # the fixture is otherwise structurally clean
  [[ "$output" != *"L14"* ]]
}

@test "LS14-comment: a commented-out hardcoded literal is not flagged" {
  ROOT=$(mktemp -d)
  _ls14_tree "$ROOT" '  // val gradlePluginVersion: String = "7.6.17"'
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [[ "$output" != *"L14"* ]]
}

# ---------------------------------------------------------------------------
# L15 — the ROOT build.gradle.kts (the dir holding settings.gradle.kts) applies com.tridium.niagara-signing
# without the signing block (issue #266, register C85). Same uncommented-code rule as preflight.sh sample-sign.
# [ev: retro n4-course-signed-jars-gate-fix Δ4]
# Named mutation: LS15 -- drop the L15 check -> the wizard root (LS15) no longer WARNs
# ---------------------------------------------------------------------------
_ls15_tree() {
  # <root>/.git sentinel, <root>/Group/{settings,build}.gradle.kts, module root <root>/Group/Mod/Foo-rt
  _ls14_tree "$1" '  val gradlePluginVersion: String = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")'
  printf '%s\n' "$2" > "$1/Group/build.gradle.kts"
}
_LS15_SIGN='plugins { id("com.tridium.niagara-signing") }'
_LS15_BLOCK='signingServices { signingProfileFactory { allowDefaultProfile.set(false) } }
niagaraSigning {
  aliases.set(listOf("CodeSigning"))
  signingProfileFile.set(layout.projectDirectory.file("profile.xml"))
}'

@test "LS15: a root that applies niagara-signing with no signing block WARNs L15 and names what is missing" {
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]                         # WARN, not FAIL
  [[ "$output" == *"WARN  lint-structure"*"build.gradle.kts"*"L15"* ]]
  [[ "$output" == *"allowDefaultProfile"* ]] && [[ "$output" == *"signingProfileFile"* ]]
}

@test "LS15-clean: the full block (allowDefaultProfile.set(false) + niagaraSigning aliases/signingProfileFile) -> no L15" {
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN
$_LS15_BLOCK"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" != *"L15"* ]]
}

@test "LS15-assign: allowDefaultProfile = false is accepted like .set(false)" {
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN
${_LS15_BLOCK/allowDefaultProfile.set(false)/allowDefaultProfile = false}"
  run "$LS" "$ROOT/Group/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"L15"* ]]
  # positive control: the same root with = true WARNs, so the silence above means "accepted", not "never scanned"
  sed -i 's/allowDefaultProfile = false/allowDefaultProfile = true/' "$ROOT/Group/build.gradle.kts"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-structure  $ROOT/Group/build.gradle.kts  L15: "*"allowDefaultProfile.set(false)"* ]]
}

@test "LS15-partial: the block without allowDefaultProfile WARNs and names only it" {
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN
${_LS15_BLOCK/allowDefaultProfile.set(false)/}"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [[ "$output" == *"L15: applies com.tridium.niagara-signing without (uncommented) allowDefaultProfile.set(false) (or = false) — add"* ]]
}

@test "LS15-comment: a commented-out block or one quoted in a string still WARNs" {
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN
/* $_LS15_BLOCK */
val note = \"allowDefaultProfile.set(false) niagaraSigning { aliases signingProfileFile\""
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [[ "$output" == *"L15"* ]]
}

@test "LS15-nosign: a root that does not apply niagara-signing is not checked" {
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" 'plugins { id("com.tridium.niagara") }'
  run "$LS" "$ROOT/Group/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"L15"* ]]
  # positive control: adding only the signing plugin id to the same root WARNs, so the root was reached
  printf '%s\n' "$_LS15_SIGN" >> "$ROOT/Group/build.gradle.kts"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-structure  $ROOT/Group/build.gradle.kts  L15: "* ]]
}

@test "LS15-barename: run by bare name from inside toolbelt/, the lib still sources and L15 still WARNs" {
  # ${BASH_SOURCE[0]%/*} of a bare name is the name itself, so the lib source failed and L15 skipped silently.
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN"
  cd "$KIT/toolbelt"
  run bash lint-structure.sh "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"L15: applies com.tridium.niagara-signing without"* ]]
  [[ "$output" != *"No such file"* ]] && [[ "$output" != *"unbound variable"* ]]
}

@test "LS15-unreadable: a root build.gradle.kts the scan cannot read WARNs L15 visibly instead of skipping" {
  [ "$(id -u)" -ne 0 ] || skip "root reads a mode-000 file"
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN"
  chmod 000 "$ROOT/Group/build.gradle.kts"
  run "$LS" "$ROOT/Group/Mod"
  chmod 644 "$ROOT/Group/build.gradle.kts"; rm -rf "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-structure  $ROOT/Group/build.gradle.kts  L15: could not scan"* ]]
}

@test "LS15-sync: lint-structure L15, preflight sample-sign and build.sh take the one table in lib/signing-block.sh" {
  # Mutation: re-inline a copy of the ERE table in any consumer -> its literal line appears and this fails.
  grep -q '^SIGN_BLOCK_ERE=(' "$KIT/toolbelt/lib/signing-block.sh"
  # shellcheck disable=SC2016
  # why: the consumer's literal source line is matched, so ${SIGN_BLOCK_ERE[@]} must stay unexpanded
  grep -qxF 'L15_SIGN_ERE=("${SIGN_BLOCK_ERE[@]}")' "$KIT/toolbelt/lint-structure.sh"
  # shellcheck disable=SC2016
  # why: same literal match as the line above
  grep -qxF 'SAMPLE_SIGN_ERE=("${SIGN_BLOCK_ERE[@]}")' "$KIT/toolbelt/preflight.sh"
  grep -q 'SIGN_BLOCK_ERE\[0\]' "$KIT/toolbelt/build.sh"
  for f in lint-structure.sh preflight.sh build.sh; do
    grep -q 'lib/signing-block.sh"' "$KIT/toolbelt/$f"
    run grep -q 'allowDefaultProfile\[\[:space:\]\]' "$KIT/toolbelt/$f"
    [ "$status" -ne 0 ]
  done
}
