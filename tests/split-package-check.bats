#!/usr/bin/env bats
# split-package-check.bats — project-level check: one Java package declared in two N4 modules (deferred-lints D5,
# issue #142). [ev: retro module-hardening-failure-modes-deltas Δ12] [ev: corpus B1125]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SPC="$KIT/toolbelt/split-package-check.sh"
  P="$TMPDIR_T/proj"
}
teardown() { chmod -R u+rwx "$TMPDIR_T" 2>/dev/null; rm -rf "$TMPDIR_T"; }

# module <rel-dir> [marker] — an N4 artifact dir with a build marker and src/
module() { mkdir -p "$P/$1/src/com/x"; : > "$P/$1/${2:-build.gradle.kts}"; }
# jfile <rel-path> <package-or-empty>
jfile() {
  mkdir -p "$(dirname "$P/$1")"
  if [ -n "$2" ]; then printf 'package %s;\nclass C {}\n' "$2" > "$P/$1"; else printf 'class C {}\n' > "$P/$1"; fi
}

@test "SPC1: the same package in two modules is one WARN naming both modules, exit 0 (advisory)" {
  module GrpA/A/A-rt; module GrpB/B/B-rt
  jfile GrpA/A/A-rt/src/com/x/U.java com.shared.util
  jfile GrpB/B/B-rt/src/com/x/U.java com.shared.util
  jfile GrpB/B/B-rt/src/com/x/V.java com.b.only
  run "$SPC" "$P"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  split-package-check  com.shared.util  declared in 2 modules: GrpA/A/A-rt (GrpA/A/A-rt/src/com/x/U.java), GrpB/B/B-rt (GrpB/B/B-rt/src/com/x/U.java)"* ]]
  if [[ "$output" == *"com.b.only"* ]]; then return 1; fi
}

@test "SPC2: one package over several files of ONE module is not a split" {
  # Mutation: SPC2 -- counting files instead of distinct modules WARNs a package spread over two files of ONE module
  module G/A/A-rt
  jfile G/A/A-rt/src/com/x/U.java com.a
  jfile G/A/A-rt/src/com/x/V.java com.a
  run "$SPC" "$P"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SPC3: --strict exits 1 on a split package (the client-CI gate form)" {
  module G/A/A-rt; module G/A/A-ux module-include.xml
  jfile G/A/A-rt/src/com/x/U.java com.a
  jfile G/A/A-ux/src/com/x/W.java com.a
  run "$SPC" --strict "$P"
  [ "$status" -eq 1 ]
  [[ "$output" == *"com.a  declared in 2 modules: G/A/A-rt"*"G/A/A-ux"* ]]
}

@test "SPC4: srcTest/ is not a module's runtime classpath and is not scanned" {
  # Mutation: SPC4 -- scanning srcTest/ as well WARNs a test fixture that mirrors a runtime package
  module G/A/A-rt; module G/B/B-rt
  jfile G/A/A-rt/src/com/x/U.java com.a
  jfile G/B/B-rt/srcTest/com/x/UTest.java com.a
  jfile G/B/B-rt/src/com/x/B.java com.b
  run "$SPC" "$P"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SPC5: the default package in two modules is a split too" {
  module G/A/A-rt; module G/B/B-rt
  jfile G/A/A-rt/src/U.java ""
  jfile G/B/B-rt/src/V.java ""
  run "$SPC" "$P"
  [[ "$output" == *"split-package-check  (default)  declared in 2 modules"* ]]
}

@test "SPC6: a directory with no module, a missing root or a bad flag is exit 3" {
  mkdir -p "$P/docs"
  run "$SPC" "$P"
  [ "$status" -eq 3 ]
  [[ "$output" == *"no module"* ]]
  run "$SPC" "$TMPDIR_T/nope"
  [ "$status" -eq 3 ]
  run "$SPC" --bogus "$P"
  [ "$status" -eq 3 ]
  run "$SPC"
  [ "$status" -eq 3 ]
}

@test "SPC7: build/ and dot-directories are pruned; a group build file with no src/ is not a module" {
  module G/A/A-rt; : > "$P/G/build.gradle.kts"
  jfile G/A/A-rt/src/com/x/U.java com.a
  mkdir -p "$P/G/A/A-rt/build/gen/src/com/x"; : > "$P/G/A/A-rt/build/gen/build.gradle.kts"
  jfile G/A/A-rt/build/gen/src/com/x/U.java com.a
  mkdir -p "$P/.cache/M/src"; : > "$P/.cache/M/build.gradle.kts"; jfile .cache/M/src/U.java com.a
  run "$SPC" "$P"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SPC-finderr: an unreadable directory under a module's src/ is exit 3 (named), never a clean pass" {
  # Mutation: SPC-finderr -- ignoring the per-module listing status skips the unreadable directory and reports clean
  module G/A/A-rt; module G/B/B-rt
  jfile G/A/A-rt/src/com/x/U.java com.a
  jfile G/B/B-rt/src/com/x/locked/U.java com.a
  chmod 000 "$P/G/B/B-rt/src/com/x/locked"
  if [ -r "$P/G/B/B-rt/src/com/x/locked" ]; then chmod 755 "$P/G/B/B-rt/src/com/x/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SPC" "$P"
  chmod 755 "$P/G/B/B-rt/src/com/x/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"B-rt/src"* ]]
}

@test "SPC-findtree: an unreadable directory in the module search is exit 3, never a module silently missed" {
  # Mutation: SPC-findtree -- ignoring the module-search status skips a module under an unreadable directory
  module G/A/A-rt; module G/Hidden/H-rt
  jfile G/A/A-rt/src/com/x/U.java com.a
  jfile G/Hidden/H-rt/src/com/x/U.java com.a
  chmod 000 "$P/G/Hidden"
  if [ -r "$P/G/Hidden" ]; then chmod 755 "$P/G/Hidden"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SPC" "$P"
  chmod 755 "$P/G/Hidden"
  [ "$status" -eq 3 ]
  [[ "$output" == *"G/Hidden"* ]]
}
