#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/stored-repack.sh. Regression guarded: Workbench
# "invalid entry compressed size" on deflated entries (5-rooms retro lesson 10). Output must be fully
# STORED, MANIFEST.MF first, signature entries next, content byte-identical, never overwrite.
# These two run the wrapper end to end on a jar zip(1) built, read back by unzip and verify-module; every
# other contract (aliases, refusals, usage, the truth fixes) is a golden case of go/internal/storedrepack.

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  R="$KIT/toolbelt/stored-repack.sh"
  d="$TMPDIR_T/src"; add_manifest "$d"; add_signature "$d"
  make_class_file "$d/com/x/A.class" 52; make_module_xml "$d" 4.14 com.x.A
  mkdir -p "$d/rc"; head -c 4000 /dev/zero | tr '\0' a > "$d/rc/index.html"   # compressible -> Defl: in input
  make_jar "$TMPDIR_T/in.jar" "$d"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "S1: output has zero Defl: entries, MANIFEST.MF is entry #1, SF/RSA next, content byte-identical" {
  [ "$(unzip -v "$TMPDIR_T/in.jar" | grep -c 'Defl:')" -gt 0 ]
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/out.jar"
  [ "$status" -eq 0 ]
  [ "$(unzip -v "$TMPDIR_T/out.jar" | grep -c 'Defl:')" -eq 0 ]
  mapfile -t order < <(unzip -Z1 "$TMPDIR_T/out.jar")
  [ "${order[0]}" = "META-INF/MANIFEST.MF" ]
  [ "${order[1]}" = "META-INF/NIAGARA4.SF" ]
  [ "${order[2]}" = "META-INF/NIAGARA4.RSA" ]
  for f in META-INF/MANIFEST.MF META-INF/NIAGARA4.SF META-INF/NIAGARA4.RSA META-INF/module.xml rc/index.html com/x/A.class; do
    cmp <(unzip -p "$TMPDIR_T/in.jar" "$f") <(unzip -p "$TMPDIR_T/out.jar" "$f")
  done
}

# The Go port (go/internal/storedrepack) orders every top-level .SF by name with its .RSA, .DSA and .EC
# blocks in that order; an orphan block, a nested .SF and a lowercase .sf are ordinary entries. The output
# passes its consumer, verify-module --stored.
@test "S5: several aliases (SERVER1 RSA, A RSA+DSA+EC) come in name order after MANIFEST.MF; verify-module --stored passes" {
  # Mutation: S5 -- appending a .SF's blocks in another order than .RSA, .DSA, .EC, or not sorting the .SF names, breaks the head order
  local s="$TMPDIR_T/src"
  rm "$s/META-INF/NIAGARA4.SF" "$s/META-INF/NIAGARA4.RSA" "$TMPDIR_T/in.jar"
  add_signature "$s" SERVER1 RSA; add_signature "$s" A RSA; add_signature "$s" A DSA; add_signature "$s" A EC
  printf 'orphan' > "$s/META-INF/C.RSA"; mkdir -p "$s/META-INF/sub"; printf 'n' > "$s/META-INF/sub/N.SF"; printf 'l' > "$s/META-INF/low.sf"
  make_jar "$TMPDIR_T/in.jar" "$s"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/out.jar"
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
  [[ "$output" == *"signature entries kept: META-INF/A.SF META-INF/A.RSA META-INF/A.DSA META-INF/A.EC META-INF/SERVER1.SF META-INF/SERVER1.RSA)" ]]
  mapfile -t order < <(unzip -Z1 "$TMPDIR_T/out.jar")
  [ "$(printf '%s ' "${order[@]:0:7}")" = \
    "META-INF/MANIFEST.MF META-INF/A.SF META-INF/A.RSA META-INF/A.DSA META-INF/A.EC META-INF/SERVER1.SF META-INF/SERVER1.RSA " ] || { printf '%s\n' "${order[@]}"; return 1; }
  [ "${#order[@]}" -eq 13 ] || { printf '%s\n' "${order[@]}"; return 1; }
  [ "$(unzip -Z1 "$TMPDIR_T/out.jar" | grep -c '/$')" -eq 0 ]   # no directory entries
  for f in META-INF/C.RSA META-INF/sub/N.SF META-INF/low.sf rc/index.html; do cmp <(unzip -p "$TMPDIR_T/in.jar" "$f") <(unzip -p "$TMPDIR_T/out.jar" "$f"); done
  run "$KIT/toolbelt/verify-module.sh" --stored "$TMPDIR_T/out.jar"
  [[ "$output" == *"PASS  stored"* ]] || { echo "$output"; return 1; }
}
