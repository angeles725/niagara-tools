#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-ntpl-manifest-deps.sh (kit lint G92), second half: template names,
# symlinks, installation jars and the bog and Px uses (NTPL10-NTPL14). The first half, its evidence and the
# fixture conventions are in lint-ntpl-manifest-deps.bats; shared setup: tests/lib/ntpl-setup.bash.

load lib/ntpl-setup

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

@test "NTPL15: a template module jar is checked template by template, each row naming <jar>!/<entry>" {
  # Mutation: NTPL15 -- refusing a file root, or skipping the templates inside a jar, passes a template module that needs an unshipped module
  local stage="$BATS_TEST_TMPDIR/jarstage" jar="$BATS_TEST_TMPDIR/ModT.jar"
  mkdir -p "$stage/templates"
  _ntpl "$stage/templates/Good.ntpl" "" baja fox-rt
  _ntpl "$stage/templates/Needs.ntpl" "" baja ghost-rt
  (cd "$stage" && zip -q -X "$jar" templates/Good.ntpl templates/Needs.ntpl)
  run "$LINT" --niagara-home "$NH" "$jar"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  ntpl-deps  $jar!/templates/Needs.ntpl: manifest dependencies not shipped by the installation $NH (a template module jar declares no build dependencies): ghost-rt" ]
  printf 'not a template\n' > "$stage/readme.txt"
  (cd "$stage" && zip -q -X "$BATS_TEST_TMPDIR/Plain.jar" readme.txt)
  run "$LINT" --niagara-home "$NH" "$BATS_TEST_TMPDIR/Plain.jar"
  [ "$status" -eq 0 ]
  [ "$output" = "PASS  lint-ntpl-manifest-deps  $BATS_TEST_TMPDIR/Plain.jar: no *.ntpl or *.napl entry in the jar (nothing to check)" ]
}
