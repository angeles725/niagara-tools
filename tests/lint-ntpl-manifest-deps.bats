#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-ntpl-manifest-deps.sh (kit lint G92): the template archives a
# module ships. A .ntpl is a zip holding template.bog and template-manifest.xml; the lint FAILs an archive
# without one of them, a malformed manifest, a <dependency> no build file of the module declares, and a
# template under newStations/ whose manifest is not a station manifest.
# [ev: retro gate-followups-and-template-lints Δ7] [ev: retro gate-followups-and-template-lints Δ8]
# Every archive is built here with `zip` (tests/helpers/n4-fixtures.bash does the same for jars); the Go
# package go/internal/lint/rules/ntplmanifest pins the untrusted-input half (unsafe names, read bounds).

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/../build-n4-module-kit" && pwd)"
  LINT="$KIT/toolbelt/lint-ntpl-manifest-deps.sh"
  M="$BATS_TEST_TMPDIR/ModA"
  mkdir -p "$M/ModA-rt/src/templates"
  printf 'dependencies {\n  nre(":nre")\n  api(":baja")\n}\n' > "$M/ModA-rt/ModA-rt.gradle.kts"
}

# _ntpl <out.ntpl> <manifest-root-attrs> <dependency>... — a template archive with both entries
_ntpl() {
  local out="$1" attrs="$2" d stage="$BATS_TEST_TMPDIR/stage.$RANDOM"
  shift 2
  mkdir -p "$stage" "$(dirname "$out")"
  printf '<bajaObjectGraph version="4.0"/>\n' > "$stage/template.bog"
  {
    printf '<templateManifest id="t1" version="1.0"%s>\n  <dependencies>\n' "$attrs"
    for d in "$@"; do printf '    <dependency name="%s" vendor="Acme" vendorVersion="1.0"/>\n' "$d"; done
    printf '  </dependencies>\n</templateManifest>\n'
  } > "$stage/template-manifest.xml"
  (cd "$stage" && zip -q -X "$out" template.bog template-manifest.xml)
}

@test "NTPL0: a module that ships no .ntpl is one PASS row, exit 0; a root that is no module is exit 3" {
  run "$LINT" "$M"
  [ "$status" -eq 0 ]
  [ "$output" = "PASS  lint-ntpl-manifest-deps  $M: no *.ntpl under the root (nothing to check)" ]
  mkdir -p "$BATS_TEST_TMPDIR/empty"
  run "$LINT" "$BATS_TEST_TMPDIR/empty"
  [ "$status" -eq 3 ]
  [[ "$output" == "ERROR  lint-ntpl-manifest-deps  $BATS_TEST_TMPDIR/empty  "* ]]
}

@test "NTPL1: an archive without template.bog or without the manifest -> FAIL ntpl-entries, exit 1" {
  _ntpl "$M/ModA-rt/src/templates/Unit.ntpl" "" baja
  zip -q -d "$M/ModA-rt/src/templates/Unit.ntpl" template.bog
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-entries  $M/ModA-rt/src/templates/Unit.ntpl: lacks template.bog" ]
}

@test "NTPL2: a manifest that is not well-formed XML -> FAIL ntpl-manifest with its position, exit 1" {
  local stage="$BATS_TEST_TMPDIR/bad"
  mkdir -p "$stage"
  printf '<bajaObjectGraph version="4.0"/>\n' > "$stage/template.bog"
  printf '<templateManifest>\n  <dependencies>\n</templateManifest>\n' > "$stage/template-manifest.xml"
  (cd "$stage" && zip -q -X "$M/ModA-rt/src/templates/Unit.ntpl" template.bog template-manifest.xml)
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  ntpl-manifest  $M/ModA-rt/src/templates/Unit.ntpl: template-manifest.xml is not well-formed XML: 3:"* ]]
}

@test "NTPL3: a manifest dependency the build never declares -> FAIL ntpl-deps naming it; declared ones are clean" {
  _ntpl "$M/ModA-rt/src/templates/Unit.ntpl" "" baja ghost-rt ModA-rt
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-deps  $M/ModA-rt/src/templates/Unit.ntpl: manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under $M: ghost-rt" ]
  printf 'dependencies {\n  nre(":nre")\n  api(":baja")\n  api(":ghost-rt")\n}\n' > "$M/ModA-rt/ModA-rt.gradle.kts"
  run "$LINT" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NTPL4: a template under newStations/ whose manifest lacks isStation=\"true\" -> FAIL ntpl-station, exit 1" {
  _ntpl "$M/overlay/defaults/workbench/newStations/Plant.ntpl" "" baja
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  ntpl-station  $M/overlay/defaults/workbench/newStations/Plant.ntpl: a template under newStations/ is a station template"*'lacks isStation="true"' ]]
  rm "$M/overlay/defaults/workbench/newStations/Plant.ntpl"
  _ntpl "$M/overlay/defaults/workbench/newStations/Plant.ntpl" ' isStation="true"' baja
  run "$LINT" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NTPL5: a .ntpl that is no zip archive is refused (FAIL, exit 1), never skipped" {
  printf '<bajaObjectGraph version="4.0"/>\n' > "$M/ModA-rt/src/templates/Unit.ntpl"
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  ntpl-entries  $M/ModA-rt/src/templates/Unit.ntpl: not a zip archive ("*") -- archive refused" ]]
}
