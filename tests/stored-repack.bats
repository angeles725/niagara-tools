#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/stored-repack.sh. Regression guarded: Workbench
# "invalid entry compressed size" on deflated entries (5-rooms retro lesson 10). Output must be fully
# STORED, MANIFEST.MF first, signature entries next, content byte-identical, never overwrite.

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
  [ "${order[0]}" = "META-INF/MANIFEST.MF" ] && [ "${order[1]}" = "META-INF/NIAGARA4.SF" ] && [ "${order[2]}" = "META-INF/NIAGARA4.RSA" ]
  for f in META-INF/MANIFEST.MF META-INF/NIAGARA4.SF META-INF/NIAGARA4.RSA META-INF/module.xml rc/index.html com/x/A.class; do
    cmp <(unzip -p "$TMPDIR_T/in.jar" "$f") <(unzip -p "$TMPDIR_T/out.jar" "$f")
  done
}

@test "S2: an unsigned jar repacks too (manifest first, no signature entries reported)" {
  rm "$TMPDIR_T/src/META-INF/NIAGARA4.SF" "$TMPDIR_T/src/META-INF/NIAGARA4.RSA"; rm "$TMPDIR_T/in.jar"
  make_jar "$TMPDIR_T/in.jar" "$TMPDIR_T/src"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/out.jar"
  [ "$status" -eq 0 ] && [[ "$output" == *"signature entries kept: none"* ]]
  [ "$(unzip -Z1 "$TMPDIR_T/out.jar" | head -1)" = "META-INF/MANIFEST.MF" ]
}

@test "S3: refuses to overwrite an existing out.jar (exit 1, untouched); bad args exit 2; missing input exits 1" {
  printf 'keep' > "$TMPDIR_T/out.jar"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/out.jar"
  [ "$status" -eq 1 ] && [[ "$output" == *"refusing to overwrite"* ]] && [ "$(cat "$TMPDIR_T/out.jar")" = "keep" ]
  run "$R" "$TMPDIR_T/in.jar"; [ "$status" -eq 2 ]
  run "$R" "$TMPDIR_T/nope.jar" "$TMPDIR_T/out2.jar"; [ "$status" -eq 1 ] && [ ! -e "$TMPDIR_T/out2.jar" ]
}

# The repack kept only NIAGARA4.SF/.RSA first: a jar signed under another alias (the N4 course
# CodeSigning profile writes CODESIGN.*, Honeywell SERVER1.*, an EC key <ALIAS>.EC) lost its
# signature-first order. Every META-INF/<ALIAS>.SF with its .RSA/.DSA/.EC block goes right after
# the manifest. [ev: retro n4-course-sample-check-precision Δ4]
@test "S4: alias-named signatures (CODESIGN.SF+.RSA, EC1.SF+.EC) follow MANIFEST.MF, before every other entry" {
  rm "$TMPDIR_T/src/META-INF/NIAGARA4.SF" "$TMPDIR_T/src/META-INF/NIAGARA4.RSA" "$TMPDIR_T/in.jar"
  add_signature "$TMPDIR_T/src" CODESIGN RSA; add_signature "$TMPDIR_T/src" EC1 EC
  make_jar "$TMPDIR_T/in.jar" "$TMPDIR_T/src"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/out.jar"
  [ "$status" -eq 0 ]
  mapfile -t order < <(unzip -Z1 "$TMPDIR_T/out.jar")
  [ "${order[0]}" = "META-INF/MANIFEST.MF" ] || { printf '%s\n' "${order[@]}"; return 1; }
  [ "$(printf '%s\n' "${order[@]:1:4}" | sort | tr '\n' ' ')" = \
    "META-INF/CODESIGN.RSA META-INF/CODESIGN.SF META-INF/EC1.EC META-INF/EC1.SF " ] || { printf '%s\n' "${order[@]}"; return 1; }
  [ "$(printf '%s\n' "${order[@]}" | grep -c '^META-INF/\(CODESIGN\|EC1\)\.')" -eq 4 ]   # each entry exactly once
  [[ "$output" == *"CODESIGN.SF"* ]] && [[ "$output" == *"EC1.EC"* ]]
}

# The Go port (go/internal/storedrepack) orders every top-level .SF by name with its .RSA, .DSA and .EC
# blocks in that order; an orphan block, a nested .SF and a lowercase .sf are ordinary entries. The output
# passes its consumer, verify-module --stored.
@test "S5: several aliases (SERVER1 RSA, A RSA+DSA+EC) come in name order after MANIFEST.MF; verify-module --stored passes" {
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

@test "S6: an existing output directory, or the input itself as output, is refused (exit 1) and nothing is written beside it" {
  mkdir "$TMPDIR_T/outdir"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/outdir"
  [ "$status" -eq 1 ]
  [ "$output" = "stored-repack: refusing to overwrite existing $TMPDIR_T/outdir" ]
  [ -z "$(ls -A "$TMPDIR_T/outdir")" ]
  cp "$TMPDIR_T/in.jar" "$TMPDIR_T/in.copy"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/in.jar"
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to overwrite existing $TMPDIR_T/in.jar" ]]
  cmp "$TMPDIR_T/in.jar" "$TMPDIR_T/in.copy"
  [ "$(find "$TMPDIR_T" -maxdepth 1 -name '.*tmp-*' | wc -l)" -eq 0 ]
}

# The bash checked the output with [ -e ], which follows a symlink: a dangling link passed, zip replaced it
# with a regular file and created its target, exit 0. The Go port refuses it like any existing output.
@test "S7: an output path holding a dangling symlink is refused (exit 1); the link and its missing target stay" {
  ln -s "$TMPDIR_T/nowhere" "$TMPDIR_T/link.jar"
  run "$R" "$TMPDIR_T/in.jar" "$TMPDIR_T/link.jar"
  [ "$status" -eq 1 ]
  [ "$output" = "stored-repack: refusing to overwrite existing $TMPDIR_T/link.jar" ]
  [ -L "$TMPDIR_T/link.jar" ]
  [ ! -e "$TMPDIR_T/nowhere" ]
}

# The bash resolved OUT_ABS with $(cd "$(dirname "$OUT")" && pwd): a missing directory made it "/out.jar", and
# zip tried to create that file in / (exit 15, or a stray /out.jar when / is writable).
@test "S8: an output in a directory that does not exist is exit 1 'cannot write', nothing written" {
  cd "$TMPDIR_T"
  run "$R" in.jar nodir/out.jar
  [ "$status" -eq 1 ]
  [ "$output" = "stored-repack: cannot write nodir/out.jar: No such file or directory" ]
  [ ! -e nodir ]
}
