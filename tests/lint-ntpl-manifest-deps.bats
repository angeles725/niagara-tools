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

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/../build-n4-module-kit" && pwd)"
  LINT="$KIT/toolbelt/lint-ntpl-manifest-deps.sh"
  M="$BATS_TEST_TMPDIR/ModA"
  mkdir -p "$M/ModA-rt/src/templates"
  printf 'dependencies {\n  nre(":nre")\n  api(":baja")\n}\n' > "$M/ModA-rt/ModA-rt.gradle.kts"
  NH="$BATS_TEST_TMPDIR/niagara"
  mkdir -p "$NH/modules"
  local j
  for j in baja fox-rt jetty-rt hx-wb web-rt control-rt bajaui-wb; do : > "$NH/modules/$j.jar"; done
  unset NIAGARA_HOME   # the machine running the test may name a real installation
}

# _ntpl <out.ntpl> <manifest-root-attrs> <dependency>... — a template archive with both entries
_ntpl() {
  local out="$1" attrs="$2" d stage="$BATS_TEST_TMPDIR/stage.$RANDOM"
  shift 2
  mkdir -p "$stage" "$(dirname "$out")"
  printf '<bajaObjectGraph version="4.0"/>\n' > "$stage/template.bog"
  {
    printf '<template id="t1" version="1.0"%s>\n  <dependencies>\n' "$attrs"
    for d in "$@"; do printf '    <dependency name="%s" vendor="Acme" vendorVersion="1.0"/>\n' "$d"; done
    printf '  </dependencies>\n</template>\n'
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

@test "NTPL10: a template name with a newline is one quoted row; .NTPL in any case is a template" {
  _ntpl "$BATS_TEST_TMPDIR/t.ntpl" "" baja
  zip -q -d "$BATS_TEST_TMPDIR/t.ntpl" template.bog
  cp "$BATS_TEST_TMPDIR/t.ntpl" "$M/ModA-rt/src/templates/a"$'\n'"PASS  x.ntpl"
  cp "$BATS_TEST_TMPDIR/t.ntpl" "$M/ModA-rt/src/templates/B.NTPL"
  run "$LINT" "$M"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "FAIL  ntpl-entries  $M/ModA-rt/src/templates/B.NTPL: lacks template.bog" ]
  [ "${lines[1]}" = "FAIL  ntpl-entries  \"$M/ModA-rt/src/templates/a\\nPASS  x.ntpl\": lacks template.bog" ]
}

@test "NTPL11: a link whose target is outside the module root is not followed: one WARN ntpl-link row, never a sibling module's template" {
  _ntpl "$BATS_TEST_TMPDIR/ModB/ModB-rt/bad.ntpl" "" ghost-rt
  ln -s "$BATS_TEST_TMPDIR" "$M/ModA-rt/up"
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  ntpl-link  $M/ModA-rt/up: a symlink to $(cd "$BATS_TEST_TMPDIR" && pwd -P), outside the module root -- not followed, nothing behind it is checked" ]
}

@test "NTPL12: a link to a part's build output stays pruned (out -> build)" {
  _ntpl "$M/ModA-rt/build/out.ntpl" "" ghost-rt
  ln -s build "$M/ModA-rt/out"
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 0 ]
  [ "$output" = "PASS  lint-ntpl-manifest-deps  $M: no *.ntpl under the root (nothing to check)" ]
}

@test "NTPL13: --niagara-home with an empty value is a usage error; a jar that is a dangling link or a directory ships nothing; a FIFO .ntpl is exit 3" {
  _ntpl "$M/ModA-rt/src/templates/Unit.ntpl" "" fox-rt dir-rt gone-rt
  run "$LINT" --niagara-home "" "$M"
  [ "$status" -eq 3 ]
  [ "${lines[0]}" = "lint-ntpl-manifest-deps: --niagara-home: an empty value names no installation" ]
  mkdir "$NH/modules/dir-rt.jar"
  ln -s "$NH/modules/nothing.jar" "$NH/modules/gone-rt.jar"
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == *"nor shipped by the installation $NH: dir-rt, gone-rt" ]]
  mkfifo "$M/ModA-rt/src/templates/p.ntpl"
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 3 ]
  [[ "$output" == *"find: '$M/ModA-rt/src/templates/p.ntpl': not a regular file" ]]
}

@test "NTPL14: a module the bog uses with no <dependency> and not shipped -> FAIL ntpl-uses; a .napl is checked as a template" {
  local stage="$BATS_TEST_TMPDIR/uses" out="$M/ModA-rt/src/templates/Unit.ntpl"
  mkdir -p "$stage"
  printf '<bajaObjectGraph version="4.0"><p m="b=baja" t="b:Folder"><p n="a" m="c=control k=kitIo v=vendorx" t="c:NumericWritable"/></p></bajaObjectGraph>\n' > "$stage/template.bog"
  printf '<template id="t1" version="1.0">\n  <dependencies>\n    <dependency name="control-rt" vendor="Tridium" vendorVersion="4.10"/>\n  </dependencies>\n</template>\n' > "$stage/template-manifest.xml"
  (cd "$stage" && zip -q -X "$out" template.bog template-manifest.xml)
  : > "$NH/modules/kitIo-rt.jar"
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-uses  $out: template.bog uses vendorx: no <dependency> of template-manifest.xml names them, and the installation $NH does not ship them" ]
  _ntpl "$M/ModA-rt/src/app/App.napl" "" ghost-rt
  run "$LINT" --niagara-home "$NH" "$M"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  ntpl-deps  $M/ModA-rt/src/app/App.napl: manifest dependencies neither declared"*": ghost-rt"* ]]
}
