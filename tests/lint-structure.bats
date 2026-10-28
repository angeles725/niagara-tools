#!/usr/bin/env bats
# RED-FIRST pins for lint-structure.sh (campaign 8 PR18, wave3 R18, B817). Source-tree structure lint
# over a <module-root> (iterates every profile via its module-include.xml). L8 (signed-jar) stays in
# verify-module.sh. Row `FAIL|WARN  lint-structure  <path>  L<n>: <reason>`; exit 0/1/3.
#   L1 package naming (com.<vendor>.*, never javax.baja.*)   L2 one @NiagaraType per file
#   L3 pure-model pkg has tests + no baja/tridium imports     L4 lexicon non-empty per artifact w/ >=1 type
#   L5 rt palette non-empty                                   L6 module-include.xml present/consistent
#   L7 3-part dependency version floors                       L9 no empty skeleton artifact (0 class + 0 palette)
#   L10 no absolute host paths in gradle.properties           L11 mixed pure+Baja srcTest declares :test-wb AND junit
#
# RED today: lint-structure.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATION (R18.5): remove a dependency's 3-part floor -> L7 FAIL on that entry.

load helpers/symroot   # assert_symroot_same <dir> <cmd...> (issue #329)

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

@test "LS10: a client gradle.properties with a host niagara_home (C:\\Honeywell\\...) WARNs L10 to migrate it (R10-D8)" {
  run "$LS" "$FX/client-gradleprops"
  [ "$status" -eq 1 ]                        # the fixture's L4/L5/L12 FAILs
  [[ "$output" == *"WARN  lint-structure  $FX/client-gradleprops/gradle.properties:2  L10: niagara_home is a host build-location key: migrate it to the user-level gradle.properties (BUILD-LOOP.md bring-up step 2)"* ]]
}

@test "LS10-userhome: user_home is not a build-location key: a host path under it stays the L10 FAIL (R10-D8)" {
  # Mutation: LS10-userhome -- adding user_home to the build-location keys turns this FAIL into a WARN.
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/.git" "$ROOT/M/Foo-doc"
  printf '<types/>\n' > "$ROOT/M/Foo-doc/module-include.xml"
  printf 'user_home=/home/u\norg.gradle.java.installations.auto-detect=false\n' > "$ROOT/M/gradle.properties"
  run "$LS" "$ROOT/M"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-structure  $ROOT/M/gradle.properties:1  L10: absolute host path in gradle.properties (tracked or not: pass host paths with -P, as build.sh does)" ]
}

@test "LS10-keys: the four build-location keys WARN in a client root (exit 0) and FAIL in a kit fixture (R10-D8, K16)" {
  # Mutation: LS10-keys -- dropping the kit-template check WARNs a fixture's niagara_home instead of FAILing it.
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/.git"
  for d in M kit/build-n4-module-kit/fixtures/MinimalPan; do
    mkdir -p "$ROOT/$d/Foo-doc"
    printf '<types/>\n' > "$ROOT/$d/Foo-doc/module-include.xml"
    printf 'niagara_home=C:\\\\N\nniagara_user_home=C:\\\\U\nnodeHome=/n\norg.gradle.java.installations.paths=/j\norg.gradle.java.installations.auto-detect=false\n' > "$ROOT/$d/gradle.properties"
  done
  run "$LS" "$ROOT/M"
  m_status=$status m_output=$output
  run "$LS" "$ROOT/kit/build-n4-module-kit/fixtures/MinimalPan"
  rm -rf "$ROOT"
  [ "$m_status" -eq 0 ] || { echo "$m_output"; false; }
  i=0
  for k in niagara_home niagara_user_home nodeHome org.gradle.java.installations.paths; do
    i=$((i + 1))
    [[ "$m_output" == *"WARN  lint-structure  $ROOT/M/gradle.properties:$i  L10: $k is a host build-location key: migrate it to the user-level gradle.properties (BUILD-LOOP.md bring-up step 2)"* ]] || { echo "$m_output"; false; }
  done
  [ "$status" -eq 1 ] && [ "$(grep -c '^FAIL  lint-structure  .*gradle.properties:[1-4]  L10: absolute host path' <<<"$output")" -eq 4 ]
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

@test "LS-noprofile: a root without profiles is an ERROR row and exit 3, never a clean pass (#351)" {
  # Mutation: LS-noprofile -- running the module-level checks on a profile-less root passes a wrong path (exit 0).
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/.git" "$ROOT/Group/notes"
  printf 'org.gradle.java.installations.paths=/j\norg.gradle.java.installations.auto-detect=false\n' > "$ROOT/Group/gradle.properties"
  run "$LS" "$ROOT/Group"
  rm -rf "$ROOT"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-structure  $ROOT/Group  no module profiles found (no *-rt/-ux/-wb/-se/-doc or build-file sub-directory: wrong path or not a module root)" ]
}

@test "LS-usage: no <module-root> argument -> exit 3" {
  run "$LS"
  [ "$status" -eq 3 ]
}

# ---------------------------------------------------------------------------
# L12 — absent/commented org.gradle.java.installations.auto-detect=false -> FAIL (R10-D8: paths may be user-level)
# Named mutation: LS7 -- remove the L12 check -> LS12 exits 0
# ---------------------------------------------------------------------------
@test "LS12: gradle.properties missing org.gradle.java.installations.auto-detect=false -> FAIL L12" {
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

@test "LS12-clean: gradle.properties with auto-detect=false and no paths key (paths are user-level, R10-D8) -> no L12" {
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/Foo-rt/src"
  printf 'org.gradle.java.installations.auto-detect=false\n' > "$ROOT/gradle.properties"
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

@test "LS10-real: gradle.properties two levels above module root fires L10 (a build-location key: WARN)" {
  local ROOT
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/.git"                                  # .git sentinel — upward walk stops here
  printf 'niagara_home=C:\\Honeywell\\Niagara-4.14.0\n' > "$ROOT/gradle.properties"
  mkdir -p "$ROOT/proj/Mod/Foo-wb"
  printf 'plugins { id("com.tridium.niagara-module") }\n' > "$ROOT/proj/Mod/Foo-wb/build.gradle.kts"
  run "$LS" "$ROOT/proj/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"WARN  lint-structure  $ROOT/gradle.properties:1  L10: niagara_home "* ]]   # path:line (truth audit T7)
}

@test "LS10-shapes: spaced, colon-separated, indented, POSIX and UNC host paths all fire L10 (truth audit T7)" {
  # Pinned by the wrapper header line "Mutation: LS10-shapes".
  ROOT=$(mktemp -d)
  mkdir -p "$ROOT/.git"
  for i in 1 2 3 4 5; do mkdir -p "$ROOT/M$i/Foo-rt"; done   # a profile: a root without one is an ERROR row
  printf 'niagara_home = C:\\N\n' > "$ROOT/M1/gradle.properties"
  printf 'niagara_home:C:\\N\n' > "$ROOT/M2/gradle.properties"
  printf '  niagara_user_home=C:/U\n' > "$ROOT/M3/gradle.properties"
  printf 'niagara_home=/opt/Niagara-4.14\n' > "$ROOT/M4/gradle.properties"
  printf 'nodeHome=\\\\server\\node\n' > "$ROOT/M5/gradle.properties"
  for i in 1 2 3 4 5; do
    run "$LS" "$ROOT/M$i"
    [ "$status" -eq 1 ] || { echo "M$i: status $status: $output"; rm -rf "$ROOT"; false; }   # L12: no auto-detect
    [[ "$output" == *"M$i/gradle.properties"*"L10: "* ]] || { echo "M$i: $output"; rm -rf "$ROOT"; false; }
  done
  rm -rf "$ROOT"
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
  [[ "$output" == *"FAIL  lint-structure  $ROOT/Group/settings.gradle.kts:1  L14: "* ]]   # path:line (truth audit T7)
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

@test "LS14-block: a hardcoded literal inside a /* */ comment is not code, so it is not flagged" {
  # Mutation: LS14-block -- matching the raw settings lines instead of ktssrc code lines FAILs a commented-out pin
  ROOT=$(mktemp -d)
  _ls14_tree "$ROOT" '/*
  val gradlePluginVersion: String = "7.6.17"
*/'
  run "$LS" "$ROOT/Group/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"L14"* ]]
  # positive control: the same pin out of the comment FAILs, so the settings file was read
  printf '%s\n' '  val gradlePluginVersion: String = "7.6.17"' >> "$ROOT/Group/settings.gradle.kts"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-structure  $ROOT/Group/settings.gradle.kts:4  L14: "* ]]
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

@test "LS15-kotlin: a block inside a nested /* */ comment is not code, so L15 still WARNs" {
  # Mutation: LS15-kotlin -- reading the root script with the Java scanner ends the nested comment at its first */ and passes the block
  ROOT=$(mktemp -d)
  _ls15_tree "$ROOT" "$_LS15_SIGN
/* outer /* inner */
$_LS15_BLOCK
*/"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-structure  $ROOT/Group/build.gradle.kts  L15: applies"* ]]
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

@test "LS15-sync: build.sh takes lib/signing-block.sh; the Go signblock.Parts equals it" {
  # Mutation: LS15-sync -- re-inline a copy of the ERE table in preflight.sh, build.sh or a Go package, or change, drop or reorder
  # an entry (pattern or row name) of the Go signblock.Parts -> this fails.
  lib="$KIT/toolbelt/lib/signing-block.sh"
  grep -q '^SIGN_BLOCK_ERE=(' "$lib"
  grep -q 'SIGN_BLOCK_ERE\[0\]' "$KIT/toolbelt/build.sh"
  grep -q 'lib/signing-block.sh"' "$KIT/toolbelt/build.sh"
  # preflight.sh is the Go wrapper (sample-sign reads signblock.Parts); neither script re-inlines the table.
  for f in preflight.sh build.sh; do
    run grep -q 'allowDefaultProfile\[\[:space:\]\]' "$KIT/toolbelt/$f"
    [ "$status" -ne 0 ]
  done
  # The Go table (go/internal/signblock, read by lint-structure L15 and the preflight port) holds the same
  # entries in the same order; no other Go package may keep a copy.
  # shellcheck disable=SC1090
  . "$lib"
  go="$KIT/../go/internal/signblock/signblock.go"
  run grep -rlF 'allowDefaultProfile[[:space:]]' "$KIT/../go/internal" --include='*.go'
  [ "$output" = "$go" ] || { printf 'Go copies of the table outside signblock:\n%s\n' "$output"; false; }
  # shellcheck disable=SC2016  # the Go source's literal backquotes, not an expansion
  mapfile -t rows < <(sed -n '/^var Parts/,/^}$/p' "$go" |
    sed -n 's/^\t{"\(.*\)", regexp\.MustCompile(`\(.*\)`)},$/\1\t\2/p')
  [ "${#rows[@]}" -eq "${#SIGN_BLOCK_ERE[@]}" ]
  [ "${#rows[@]}" -eq "${#SIGN_BLOCK_NAME[@]}" ]
  [ "${#rows[@]}" -eq 4 ]
  for k in "${!rows[@]}"; do
    [ "${rows[$k]}" = "${SIGN_BLOCK_NAME[$k]}"$'\t'"${SIGN_BLOCK_ERE[$k]}" ] ||
      { printf 'Parts[%s] = %s\nlib      = %s\t%s\n' "$k" "${rows[$k]}" "${SIGN_BLOCK_NAME[$k]}" "${SIGN_BLOCK_ERE[$k]}"; false; }
  done
}

# ---------------------------------------------------------------------------
# L16 — the gradle root's gradle.properties names (niagaraSigningProfile) a LocalSigningProfile whose
# validity - refresh is below 32 days: the signing plugin re-mints its key at every signing and the jars carry a
# signature block no JVM verifies. One predicate with run-niagara-test (go/internal/signingprofile).
# [ev: retro r10-signing-margin-lint Δ1]
# ---------------------------------------------------------------------------
_ls16_tree() {
  # <root>/.git, gradle root <root>/Group (settings + gradle.properties naming signing/p.xml), profile <validity> <refresh>
  _ls14_tree "$1" '  val gradlePluginVersion: String = providers.gradleProperty("niagaraPluginVersion").getOrElse("7.6.17")'
  printf 'niagaraSigningProfile=signing/p.xml\n' > "$1/Group/gradle.properties"
  mkdir -p "$1/Group/signing"
  printf '<?xml version="1.0" encoding="UTF-8"?>\n<properties>\n<entry key="niagara.signing.validity">%s</entry>\n<entry key="niagara.signing.refresh">%s</entry>\n<entry key="niagara.signing.profileType">com.tridium.gradle.plugins.signing.profile.LocalSigningProfile</entry>\n</properties>\n' \
    "$2" "$3" > "$1/Group/signing/p.xml"
}

@test "LS16-margin: a 30/15 LocalSigningProfile FAILs L16 naming both values; 214/183 (margin 31) FAILs; 365/183 passes" {
  # Mutation: LS16-margin -- lowering signingprofile.MinMargin to 31 passes 214/183
  ROOT=$(mktemp -d)
  _ls16_tree "$ROOT" 30 15
  run "$LS" "$ROOT/Group/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-structure  $ROOT/Group/gradle.properties  L16: "*"niagara.signing.validity=30, niagara.signing.refresh=15"*"re-minted at every signing"* ]]
  _ls16_tree "$ROOT" 214 183
  run "$LS" "$ROOT/Group/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"L16: "*"validity=214, niagara.signing.refresh=183"* ]]
  _ls16_tree "$ROOT" 365 183
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ] && [[ "$output" != *"L16"* ]]
}

@test "LS16-malformed: a profile value that is not a whole number is exit 3 with the read error, no verdict" {
  ROOT=$(mktemp -d)
  _ls16_tree "$ROOT" 'a year' 183
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 3 ]
  [[ "$output" == *"lint-structure: cannot scan "*"p.xml: niagara.signing.validity=\"a year\" is not a whole number of days"* ]]
}

@test "LS16-none: a gradle root that declares no niagaraSigningProfile gets no L16 row" {
  ROOT=$(mktemp -d)
  _ls16_tree "$ROOT" 30 15
  printf '# niagaraSigningProfile=signing/p.xml\n' > "$ROOT/Group/gradle.properties"
  run "$LS" "$ROOT/Group/Mod"
  rm -rf "$ROOT"
  [ "$status" -eq 0 ] && [[ "$output" != *"L16"* ]]
}

@test "LS-symroot: a symlinked module root is walked like find -H, same rows as the directory (issue #329)" {
  # Mutation: LS-symroot -- dropping -H from the find in _find_files misses gradle.properties under a symlinked root.
  assert_symroot_same "$FX/client-gradleprops" "$LS" "$FX/client-gradleprops"
  [ "$status" -eq 1 ] && [[ "$output" == *"WARN"*"L10: niagara_home"* ]]
}

# L17 — module.palette type specs: a dotted class name or a B-prefixed name of a declared type FAILs; another
# B-prefixed class-like name and a symbol remapped to another module WARN; an identical repeated m= is clean (the
# decoder keeps one module per symbol). [ev: retro course-coverage Δ20]
_ls17_mod() {
  local m="$1"
  mkdir -p "$m/Acme-rt"
  printf '<types><type name="Room" class="com.acme.BRoom"/></types>\n' > "$m/Acme-rt/module-include.xml"
  printf 'Room.displayName=Room\n' > "$m/Acme-rt/module.lexicon"
}

@test "LS17a: t=\"ac:BRoom\" for the declared type Room FAILs L17; t=\"ac:Room\" is clean" {
  # Mutation: LS17a -- dropping the declared-type lookup turns the B-prefixed own type into a WARN, not the FAIL
  M="$BATS_TEST_TMPDIR/Mod"; _ls17_mod "$M"
  printf '%s\n' '<bajaObjectGraph version="4.0">' '<p m="b=baja" t="b:UnrestrictedFolder">' \
    '  <p n="Room" m="ac=Acme" t="ac:BRoom"/>' '  <p n="Room2" t="ac:Room"/>' '</p>' '</bajaObjectGraph>' > "$M/Acme-rt/module.palette"
  run "$LS" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = 'FAIL  lint-structure  Acme-rt/module.palette:3  L17: module.palette t="ac:BRoom" keeps the class'"'"'s B prefix: the type is ac:Room (Workbench: "Type not found")' ]
}

@test "LS17b: the MinimalDriver fixture repeats m=\"md=MinimalDriver\" and gets no L17 row; a remap WARNs" {
  # Mutation: LS17b -- comparing every repeated m= instead of a different module WARNs the identical repeat
  run "$LS" "$KIT/fixtures/MinimalDriver/MinimalDriver"
  [[ "$output" != *"L17"* ]]
  M="$BATS_TEST_TMPDIR/Mod"; _ls17_mod "$M"
  printf '%s\n' '<bajaObjectGraph version="4.0">' '<p m="b=baja" t="b:UnrestrictedFolder">' \
    '  <p n="Room" m="ac=Acme" t="ac:Room"/>' '  <p n="X" m="ac=Other" t="ac:X"/>' '</p>' '</bajaObjectGraph>' > "$M/Acme-rt/module.palette"
  run "$LS" "$M"
  [ "$status" -eq 0 ]
  [ "$output" = 'WARN  lint-structure  Acme-rt/module.palette:4  L17: module.palette m="ac=Other" redeclares symbol ac (first ac=Acme at line 3): every later t="ac:..." resolves in Other' ]
}
