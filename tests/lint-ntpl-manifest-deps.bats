#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-ntpl-manifest-deps.sh (kit lint G92): the template archives a
# module ships. A .ntpl is a zip holding template.bog and template-manifest.xml; the lint FAILs an archive
# without one of them, a manifest that is no well-formed template manifest, a <dependency> that neither a
# build file of the module declares nor the installation ships, and a template under newStations/ whose
# manifest is not a station manifest.
# [ev: retro gate-followups-and-template-lints Δ7] [ev: retro gate-followups-and-template-lints Δ8]
# [ev: retro gate-followups-and-template-lints Δ12]
# Every archive is built here with `zip` (tests/helpers/n4-fixtures.bash does the same for jars), and the
# installation is a directory of empty jars named like stock modules; the Go package
# go/internal/lint/rules/ntplmanifest pins the untrusted-input half (unsafe names, read bounds).
# The cases are split by concern: this file (NTPL0-NTPL9: entries, manifest, dependencies, station, the
# installation) and lint-ntpl-manifest-deps-paths.bats (NTPL10-NTPL14: names, links, jars of the installation,
# the bog and Px uses). Shared setup and _ntpl helper: tests/lib/ntpl-setup.bash.

load lib/ntpl-setup

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
  printf '<template>\n  <dependencies>\n</template>\n' > "$stage/template-manifest.xml"
  (cd "$stage" && zip -q -X "$M/ModA-rt/src/templates/Unit.ntpl" template.bog template-manifest.xml)
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  ntpl-manifest  $M/ModA-rt/src/templates/Unit.ntpl: template-manifest.xml is not well-formed XML: 3:"* ]]
}

@test "NTPL3: a dependency nobody declares or ships -> FAIL ntpl-deps naming it; declared ones are clean" {
  _ntpl "$M/ModA-rt/src/templates/Unit.ntpl" "" baja ghost-rt ModA-rt
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-deps  $M/ModA-rt/src/templates/Unit.ntpl: manifest dependencies neither declared (api/nre/implementation) in any *.gradle.kts under $M nor shipped by the installation $NH: ghost-rt" ]
  printf 'dependencies {\n  nre(":nre")\n  api(":baja")\n  api(":ghost-rt")\n}\n' > "$M/ModA-rt/ModA-rt.gradle.kts"
  run "$LINT" --niagara-home "$NH" "$M"
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

@test "NTPL6: a symlinked template and a template under a symlinked directory inside the module are checked, never 'nothing to check'" {
  local store="$M/store"   # link targets inside the module root (K6: a link out of it is not followed)
  _ntpl "$store/broken.zip" "" baja
  zip -q -d "$store/broken.zip" template-manifest.xml
  ln -s "$store/broken.zip" "$M/ModA-rt/l.ntpl"
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-entries  $M/ModA-rt/l.ntpl: lacks template-manifest.xml" ]
  rm "$M/ModA-rt/l.ntpl"
  _ntpl "$store/dir/inner.ntpl" "" baja
  zip -q -d "$store/dir/inner.ntpl" template.bog
  rmdir "$M/ModA-rt/src/templates" "$M/ModA-rt/src"
  ln -s "$store/dir" "$M/ModA-rt/src"
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-entries  $M/ModA-rt/src/inner.ntpl: lacks template.bog" ]
}

@test "NTPL6b: a .ntpl link that leads nowhere is exit 3 naming it, never 'nothing to check'" {
  ln -s "$BATS_TEST_TMPDIR/gone.ntpl" "$M/ModA-rt/l.ntpl"
  run "$LINT" "$M"
  [ "$status" -eq 3 ]
  [[ "$output" == *"cannot list every file under $M: "*"$M/ModA-rt/l.ntpl"* ]]
  [[ "$output" != *"nothing to check"* ]]
}

@test "NTPL7: a station template listing many stock modules, none declared, passes with --niagara-home or NIAGARA_HOME" {
  _ntpl "$M/overlay/defaults/workbench/newStations/Plant.ntpl" ' isStation="true"' fox-rt jetty-rt hx-wb web-rt control-rt bajaui-wb baja
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  NIAGARA_HOME="$NH" run "$LINT" "$M"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NTPL8: without an installation path undeclared dependencies are WARN rows and exit 0, with one row saying why" {
  _ntpl "$M/ModA-rt/src/templates/Unit.ntpl" "" fox-rt ghost-rt
  run "$LINT" "$M"
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "WARN  lint-ntpl-manifest-deps  $M: the installation's module list is not known, so a stock module cannot be told from a missing one: pass --niagara-home <dir> or set NIAGARA_HOME (the ntpl-deps rows are WARN until then)" ]
  [ "${lines[1]}" = "WARN  ntpl-deps  $M/ModA-rt/src/templates/Unit.ntpl: manifest dependencies not declared (api/nre/implementation) in any *.gradle.kts under $M: fox-rt, ghost-rt" ]
  [ "${#lines[@]}" -eq 2 ]
}

@test "NTPL9: an installation path that does not exist or has no modules/ is exit 3, never a run without the list" {
  _ntpl "$M/ModA-rt/src/templates/Unit.ntpl" "" fox-rt
  run "$LINT" --niagara-home "$BATS_TEST_TMPDIR/nope" "$M"
  [ "$status" -eq 3 ]
  [ "$output" = "lint-ntpl-manifest-deps: --niagara-home $BATS_TEST_TMPDIR/nope: no modules/ directory (not a Niagara installation)" ]
  mkdir "$BATS_TEST_TMPDIR/bare"
  NIAGARA_HOME="$BATS_TEST_TMPDIR/bare" run "$LINT" "$M"
  [ "$status" -eq 3 ]
  [ "$output" = "lint-ntpl-manifest-deps: NIAGARA_HOME $BATS_TEST_TMPDIR/bare: no modules/ directory (not a Niagara installation)" ]
}
