#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-confirm-required.sh (types/actions.md § 3): a @NiagaraAction whose
# name holds a destructive word (reset, clear, delete, remove, purge, wipe, factory, restart, reboot, overwrite)
# must declare Flags.CONFIRM_REQUIRED, so Workbench and the bajaux invokeAction ask before running it. The Go
# rule's own cases are go/internal/lint/rules/confirmrequired/testdata; these pin the wrapper end to end.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LINT="$KIT/toolbelt/lint-confirm-required.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-confirm-required"
}

@test "CR-usage: no argument exits 3 with the usage line" {
  run "$LINT"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: lint-confirm-required.sh [--strict] <src-root>..."* ]]
}

@test "CR1: a destructive action without the flag FAILs on its line, exit 1, naming the action and the fix" {
  run "$LINT" "$FX/missing/src"
  [ "$status" -eq 1 ]
  [[ "${lines[0]}" == "FAIL  lint-confirm-required  $FX/missing/src/com/x/BPanel.java:4  faultReset: "* ]]
  [[ "${lines[1]}" == "FAIL  lint-confirm-required  $FX/missing/src/com/x/BPanel.java:5  clearHistory: "* ]]
  [ "${#lines[@]}" -eq 2 ]
  [[ "$output" == *"declare flags = Flags.CONFIRM_REQUIRED"* ]]
  [[ "$output" == *"// lint-confirm-required: waived <reason>"* ]]
}

@test "CR2: CONFIRM_REQUIRED alone or combined with | passes (exit 0, no output); --strict changes nothing" {
  run "$LINT" "$FX/flagged/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "$LINT" --strict "$FX/flagged/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "CR3: an annotation split over several lines is read whole: the flag on a later line passes, a missing one FAILs on the @ line" {
  run "$LINT" "$FX/multiline/src"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "$output" == "FAIL  lint-confirm-required  $FX/multiline/src/com/x/BWide.java:10  resetBad: "* ]]
}

@test "CR4: a waiver with a reason, on the line or the comment line above, passes" {
  run "$LINT" "$FX/waiver/src"
  [[ "$output" != *"resetSame"* ]]
  [[ "$output" != *"resetAbove"* ]]
}

@test "CR5: a bare waiver marker is a FAIL of its own, naming the missing reason" {
  run "$LINT" "$FX/waiver/src"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "$output" == "FAIL  lint-confirm-required  $FX/waiver/src/com/x/BWaive.java:8  resetBare: waiver without a reason"* ]]
}

@test "CR6: a non-destructive name, and a name that holds a destructive word only as a substring, is clean" {
  run "$LINT" "$FX/names/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "CR7: @NiagaraAction text in a comment, a Javadoc line or a string literal is not an annotation" {
  run "$LINT" "$FX/literals/src"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "CR-empty: a root with no *.java is an ERROR, exit 3" {
  mkdir -p "$BATS_TEST_TMPDIR/empty"
  run "$LINT" "$BATS_TEST_TMPDIR/empty"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-confirm-required  $BATS_TEST_TMPDIR/empty  no *.java sources found (wrong path or empty source tree)" ]
}
