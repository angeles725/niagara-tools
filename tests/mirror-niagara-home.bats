#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/mirror-niagara-home.sh. Regression guarded: the bitácora
# version ran `rm -rf "$mir"` on any user-supplied path — a typo could wipe a real install or $HOME.

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  M="$KIT/toolbelt/mirror-niagara-home.sh"
  make_niagara_home "$TMPDIR_T/nh" 7.6.17
  : > "$TMPDIR_T/nh/modules/baja.jar"; : > "$TMPDIR_T/nh/modules/Foo-rt.jar"; : > "$TMPDIR_T/nh/modules/control-rt.jar"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "M1: mirror == source is refused with exit 20 and nothing is modified (destroying a real niagara_home)" {
  before="$(find "$TMPDIR_T/nh" | sort)"
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T/nh"
  [ "$status" -eq 20 ] && [[ "$output" == *"refusing"* ]]
  [ "$(find "$TMPDIR_T/nh" | sort)" = "$before" ]
}

@test "M2: mirror inside source, and source inside mirror, are both refused with exit 20" {
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T/nh/modules"
  [ "$status" -eq 20 ] && [ -e "$TMPDIR_T/nh/modules/baja.jar" ]
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T"
  [ "$status" -eq 20 ] && [ -e "$TMPDIR_T/nh/etc/m2/repository" ]
}

@test "M3: an existing non-empty dir without .niagara-mirror is refused and NOT wiped (wiping an arbitrary dir)" {
  mkdir -p "$TMPDIR_T/precious/sub"; : > "$TMPDIR_T/precious/keep.txt"
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T/precious"
  [ "$status" -eq 20 ] && [[ "$output" == *"refusing to wipe"* ]]
  [ -f "$TMPDIR_T/precious/keep.txt" ] && [ -d "$TMPDIR_T/precious/sub" ]
}

@test "M4: happy path links every top-level entry and jar except the excluded one, writes the marker, prints the count" {
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T/mir" Foo-rt.jar
  [ "$status" -eq 0 ] && [[ "$output" == *"2 jars linked"* ]]
  [ -L "$TMPDIR_T/mir/etc" ] && [ -L "$TMPDIR_T/mir/bin" ]
  [ -d "$TMPDIR_T/mir/modules" ] && [ ! -L "$TMPDIR_T/mir/modules" ] && [ -w "$TMPDIR_T/mir/modules" ]
  [ -L "$TMPDIR_T/mir/modules/baja.jar" ] && [ -L "$TMPDIR_T/mir/modules/control-rt.jar" ]
  [ ! -e "$TMPDIR_T/mir/modules/Foo-rt.jar" ]
  grep -q "^source=" "$TMPDIR_T/mir/.niagara-mirror"
  # idempotent: a second run on its own mirror rebuilds it and drops a leftover build artifact
  : > "$TMPDIR_T/mir/modules/built.jar"
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T/mir"
  [ "$status" -eq 0 ] && [ ! -e "$TMPDIR_T/mir/modules/built.jar" ] && [ -L "$TMPDIR_T/mir/modules/Foo-rt.jar" ]
}

@test "M5: an empty source modules/ is tolerated (0 jars linked, exit 0)" {
  rm "$TMPDIR_T/nh/modules/"*.jar
  run "$M" "$TMPDIR_T/nh" "$TMPDIR_T/mir"
  [ "$status" -eq 0 ] && [[ "$output" == *"0 jars linked"* ]]
}
