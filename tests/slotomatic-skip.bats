#!/usr/bin/env bats
# Unit tests for build-n4-module-kit/toolbelt/lib/slotomatic-skip.sh — the shared detector for a file Slot-o-Matic
# skipped. Slot-o-Matic logs "Could not process <path>.java: <msg>" (Compiler.java:240-246) and, at the end of the
# run, "The following files failed to compile: " followed by one "  <abs path>: <msg>" line per file
# (Slotomatic.java:148-158), all at WARNING level, and its gradle task still ends BUILD SUCCESSFUL (METHODOLOGY K15).
# [ev: retro r10-slotomatic-skip-fail-open Δ1]
# Named mutation: drop the block-form branch of the awk -> SK-block-only reports nothing and exits 0.

setup() {
  TMPDIR_T="$(mktemp -d)"
  # shellcheck disable=SC1091  # kit lib under test, resolved at runtime
  . "$BATS_TEST_DIRNAME/../build-n4-module-kit/toolbelt/lib/slotomatic-skip.sh"
  LOG="$TMPDIR_T/gradle.log"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SK-none: a log with no skipped file prints nothing and returns 0" {
  printf '> Task :Foo-rt:slotomatic\nBLimiter.java:12: error: cannot find symbol\nBUILD SUCCESSFUL in 3s\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SK-one: one 'Could not process' line -> one FAIL row with its reason, non-zero" {
  printf 'WARNING: Could not process /x/src/com/a/BHmiStatus.java: Found multiple metadata blocks\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java (Found multiple metadata blocks)" ]
}

@test "SK-block-only: only the 'failed to compile' block names the file -> one FAIL row" {
  printf 'The following files failed to compile: \n  /x/src/com/a/BHmiStatus.java: Found multiple metadata blocks\n> Task :Foo-rt:jar\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java (Found multiple metadata blocks)" ]
}

@test "SK-block-bare-path: a block entry with no message still counts" {
  printf 'WARNING: The following files failed to compile:\nWARNING:   /x/src/com/a/BHmiStatus.java\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java ("*")" ]]
}

@test "SK-both-forms: a file named by both forms is counted once" {
  printf 'Could not process /x/B.java: Found multiple metadata blocks\nThe following files failed to compile: \n  /x/B.java: Found multiple metadata blocks\nBUILD SUCCESSFUL\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [ "${lines[0]}" = "FAIL slotomatic-skip /x/B.java (Found multiple metadata blocks)" ]
}

@test "SK-two-files: two skipped files -> two rows in log order" {
  printf 'Could not process /x/A.java: m1\nCould not process C:\\m\\src\\B.java: m2\r\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "FAIL slotomatic-skip /x/A.java (m1)" ]
  [ "${lines[1]}" = 'FAIL slotomatic-skip C:\m\src\B.java (m2)' ]
}

@test "SK-javac-not-entry: a javac 'X.java:12: error' line inside the block is not a skipped file" {
  printf 'The following files failed to compile: \n  /x/A.java: m1\n  /x/C.java:12: error: boom\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "$output" == *"/x/A.java (m1)"* ]]
}

@test "SK-unreadable: a missing log fails closed with a FAIL row" {
  run slotomatic_skip_scan "$TMPDIR_T/missing.log"
  [ "$status" -eq 2 ]
  [[ "$output" == "FAIL slotomatic-skip "*"(gradle log unreadable)" ]]
}

# SK-srctest: Slot-o-Matic also walks srcTest/, where a plain B*Test.java has no metadata block; srcTest classes
# are never packaged into the module jar, so their skip is not a stale-jar FAIL (CompPan exit 34 false positive).
# Named mutation: drop the srcTest guard in add() -> SK-srctest-only returns 1 with a FAIL row.
@test "SK-srctest-only: a skipped srcTest/ file (either path separator) is not a FAIL -> nothing, 0" {
  printf 'WARNING: Could not process /x/srcTest/com/a/BLimiterTest.java: Found multiple metadata blocks\nThe following files failed to compile:\n  C:\\x\\srcTest\\com\\a\\BOtherTest.java\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SK-srctest-mixed: a src/ skip still FAILs beside a srcTest/ one -> only the src/ row, 1" {
  printf 'WARNING: Could not process /x/srcTest/com/a/BLimiterTest.java: Found multiple metadata blocks\nWARNING: Could not process /x/src/com/a/BHmiStatus.java: Found multiple metadata blocks\n' > "$LOG"
  run slotomatic_skip_scan "$LOG"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL slotomatic-skip /x/src/com/a/BHmiStatus.java (Found multiple metadata blocks)" ]
}
