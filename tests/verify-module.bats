#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/verify-module.sh (THE gate). Fixtures are generated in-test
# by tests/helpers/n4-fixtures.bash — zero committed binaries. Each case names the regression it guards.

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  VM="$KIT/toolbelt/verify-module.sh"
}
teardown() { rm -rf "$TMPDIR_T"; }

# good_dir <dir>: a complete, valid signed module tree (2 classes, 1 declared type, baja 4.14)
good_dir() {
  add_manifest "$1"; add_signature "$1"
  make_class_file "$1/com/x/A.class" 52; make_class_file "$1/com/x/B.class" 52
  make_module_xml "$1" 4.14 com.x.A
}

@test "V1: major 65 on a NON-first class fails bytecode (old build.sh only read the first class)" {
  good_dir "$TMPDIR_T/d"; make_class_file "$TMPDIR_T/d/com/x/B.class" 65
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" "$TMPDIR_T/j.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  bytecode"* ]]
  [[ "$output" == *"com/x/B.class major=65"* ]]
}

@test "V2: missing NIAGARA4.SF fails signed (unsigned jar on a signing-enforcing station)" {
  good_dir "$TMPDIR_T/d"; rm "$TMPDIR_T/d/META-INF/NIAGARA4.SF"
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" "$TMPDIR_T/j.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  signed"* ]]
}

@test "V3: module.xml type without a .class fails types (station boot-loop 'Type not found')" {
  good_dir "$TMPDIR_T/d"; make_module_xml "$TMPDIR_T/d" 4.14 com.x.A com.x.Missing
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" "$TMPDIR_T/j.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  types"* ]] && [[ "$output" == *"com.x.Missing"* ]]
}

@test "V4: --src type count != module-include.xml fails typecount (type annotated, never regenerated)" {
  good_dir "$TMPDIR_T/d"
  mkdir -p "$TMPDIR_T/mod"; make_jar "$TMPDIR_T/Foo-rt.jar" "$TMPDIR_T/d"
  make_module_include "$TMPDIR_T/mod/Foo-rt" com.x.A com.x.B     # 2 in source, 1 packaged
  run "$VM" --src "$TMPDIR_T/mod" "$TMPDIR_T/Foo-rt.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  typecount"* ]] && [[ "$output" == *"declares 1 types"* ]]
}

@test "V5: baja 4.15 fails --target-version 4.14 and passes 4.15 (4.15-built jar rejected by a 4.14 station)" {
  good_dir "$TMPDIR_T/d"; make_module_xml "$TMPDIR_T/d" 4.15 com.x.A
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" --target-version 4.14 "$TMPDIR_T/j.jar"
  [ "$status" -eq 1 ] && [[ "$output" == *"FAIL  baja"* ]]
  run "$VM" --target-version 4.15 "$TMPDIR_T/j.jar"
  [ "$status" -eq 0 ]
}

@test "V6: --stored fails a deflated jar and passes a STORED one (Workbench 'invalid entry compressed size')" {
  good_dir "$TMPDIR_T/d"; head -c 4000 /dev/zero | tr '\0' a > "$TMPDIR_T/d/rc.txt"   # compressible entry
  make_jar "$TMPDIR_T/defl.jar" "$TMPDIR_T/d"; make_stored_jar "$TMPDIR_T/st.jar" "$TMPDIR_T/d"
  run "$VM" --stored "$TMPDIR_T/defl.jar"
  [ "$status" -eq 1 ] && [[ "$output" == *"FAIL  stored"* ]]
  run "$VM" --stored "$TMPDIR_T/st.jar"
  [ "$status" -eq 0 ] && [[ "$output" == *"PASS  stored"* ]]
}

@test "V7: --src raw-number MIN/MAX facet fails facets (BFacets.make(BFacets.MIN, 0d) does not compile)" {
  good_dir "$TMPDIR_T/d"; make_jar "$TMPDIR_T/Foo-rt.jar" "$TMPDIR_T/d"
  make_module_include "$TMPDIR_T/mod/Foo-rt" com.x.A
  mkdir -p "$TMPDIR_T/mod/Foo-rt/src/com/x"
  printf 'class A { Object f = BFacets.make(BFacets.MIN, -40); }\n' > "$TMPDIR_T/mod/Foo-rt/src/com/x/A.java"
  run "$VM" --src "$TMPDIR_T/mod" "$TMPDIR_T/Foo-rt.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  facets"* ]] && [[ "$output" == *"PASS  typecount"* ]]
}

@test "V8: a good jar passes every default check, opt-in checks report SKIP, exit 0" {
  good_dir "$TMPDIR_T/d"; make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" "$TMPDIR_T/j.jar"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  bytecode"* ]] && [[ "$output" == *"PASS  signed"* ]] && [[ "$output" == *"PASS  types"* ]]
  [[ "$output" == *"SKIP  baja"* ]] && [[ "$output" == *"SKIP  stored"* ]] && [[ "$output" == *"SKIP  typecount"* ]]
  [[ "$output" == *"ALL PASS"* ]]
}

@test "V9: no jar exits 2, a non-jar argument exits 2, an unreadable jar exits 3 (usage vs environment)" {
  run "$VM"; [ "$status" -eq 2 ]
  run "$VM" "$TMPDIR_T/notajar.txt"; [ "$status" -eq 2 ]
  run "$VM" "$TMPDIR_T/missing.jar"; [ "$status" -eq 3 ]
}

# --- B4 (Campaign 3 C3-PR1): editor-backup files shipped under rc/ ---
# rc/ backup files (*~, *.bak/.bak2, *.orig) are packaged by processResources — they ship,
# are servable, and bloat the jar (DashboardPan-ux shipped index.html.bak/.bak2). A default WARN
# (does NOT fail — the jar still boots); --strict promotes it to FAIL. Detail NAMES the offenders
# (house style, cf. V1 naming the bad class). RED-first: no rcbackup check exists yet.

@test "V10: an rc/ editor-backup file emits a WARN row naming it, but exit stays 0 (jar otherwise valid)" {
  good_dir "$TMPDIR_T/d"; mkdir -p "$TMPDIR_T/d/rc"
  : > "$TMPDIR_T/d/rc/index.html.bak"; : > "$TMPDIR_T/d/rc/index.html.bak2"
  : > "$TMPDIR_T/d/rc/notes~";         : > "$TMPDIR_T/d/rc/merge.orig"
  : > "$TMPDIR_T/d/rc/icon16.png"      # a real asset — must NOT be flagged
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" "$TMPDIR_T/j.jar"
  [ "$status" -eq 0 ]                                  # WARN does not fail the gate
  [[ "$output" == *"WARN  rcbackup"* ]]
  # globs covered: *.bak, *.bak2 (*.bak*), *~, *.orig — all named; the real asset is not
  [[ "$output" == *"index.html.bak"* ]]
  [[ "$output" == *"notes~"* ]]
  [[ "$output" == *"merge.orig"* ]]
  [[ "$output" != *"icon16.png"* ]]
}

@test "V11: a clean rc/ (only real assets) emits NO rcbackup WARN, exit 0" {
  good_dir "$TMPDIR_T/d"; mkdir -p "$TMPDIR_T/d/rc"
  : > "$TMPDIR_T/d/rc/index.html"; : > "$TMPDIR_T/d/rc/icon16.png"
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" "$TMPDIR_T/j.jar"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN  rcbackup"* ]]
  [[ "$output" == *"ALL PASS"* ]]
}

@test "V12: --strict promotes the rc/ backup WARN to FAIL (exit 1)" {
  good_dir "$TMPDIR_T/d"; mkdir -p "$TMPDIR_T/d/rc"
  : > "$TMPDIR_T/d/rc/index.html.bak"
  make_jar "$TMPDIR_T/j.jar" "$TMPDIR_T/d"
  run "$VM" --strict "$TMPDIR_T/j.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  rcbackup"* ]]
  [[ "$output" == *"index.html.bak"* ]]
}
