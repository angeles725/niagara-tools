#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-no-system-out.sh
# Module diagnostics must use java.util.logging.Logger, never System.out/err — which
# bypass BLogHistoryService and the Logging service. [ev: retro 2026-09-19-observability]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  NSO="$KIT/toolbelt/lint-no-system-out.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "NSO-usage: no arg exits 3" {
  run "$NSO"
  [ "$status" -eq 3 ]
}

@test "NSO-nondir: a non-directory arg exits 3" {
  run "$NSO" "$TMPDIR_T/does-not-exist"
  [ "$status" -eq 3 ]
}

@test "NSO1: java that uses Logger (no System.out) exits 0 (clean)" {
  printf 'class A {\n  static final Logger LOG = Logger.getLogger("mod");\n  void f() { LOG.fine("hi"); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$NSO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "NSO2: System.out.println is flagged (exit 1)" {
  printf 'class A {\n  void f() { System.out.println("debug"); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$NSO" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-no-system-out"* ]]
  [[ "$output" == *"A.java:2"* ]]
  # Named mutation: drop the System.out.print case -> NSO2 FAIL vanishes (exit 0).
}

@test "NSO3: System.err.print is also flagged (exit 1)" {
  printf 'class A {\n  void f() { System.err.print("oops"); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$NSO" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}

@test "NSO4: a COMMENTED System.out.println is NOT flagged (exit 0)" {
  printf 'class A {\n  void f() { /*ok*/ } // System.out.println("old debug")\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$NSO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  # Named mutation: stop stripping // comments -> NSO4 would FAIL (exit 1).
}

@test "NSO5: dot-directories are pruned" {
  mkdir -p "$TMPDIR_T/Mod/.git"
  printf 'class J { void f(){ System.out.println("x"); } }\n' > "$TMPDIR_T/Mod/.git/J.java"
  run "$NSO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
}

@test "NSO-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass (audit A0)" {
  # Mutation: NSO-finderr -- ignoring the find exit status skips the unreadable directory's files and exits 0.
  # [ev: issue #226 R3-find-error-still-fail-open]
  mkdir -p "$TMPDIR_T/Mod/src/com/x/locked"
  printf '%s\n' 'class A {}' > "$TMPDIR_T/Mod/src/com/x/locked/A.java"
  chmod 000 "$TMPDIR_T/Mod/src/com/x/locked"
  if [ -r "$TMPDIR_T/Mod/src/com/x/locked" ]; then chmod 755 "$TMPDIR_T/Mod/src/com/x/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$NSO" "$TMPDIR_T/Mod"
  chmod 755 "$TMPDIR_T/Mod/src/com/x/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}
