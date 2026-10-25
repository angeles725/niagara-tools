#!/usr/bin/env bats
# Smoke tests for build-n4-module-kit/toolbelt/run-niagara-test.sh: the wrapper builds and execs the Go tool, and
# its exit code is the verdict. Every row, reason and edge is a test of go/internal/niagaratest. The logs are
# synthetic, built from line shapes of real runs (retro testing-parity-btestng).

setup() {
  R="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/run-niagara-test.sh"
}

@test "RNT1: a log whose every summary ran tests with 0 failures is PASS, exit 0" {
  printf 'Total tests run: 6, Failures: 0, Skips: 0\r\nBUILD SUCCESSFUL\r\n' > "$BATS_TEST_TMPDIR/run.log"
  run "$R" verdict "$BATS_TEST_TMPDIR/run.log"
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
  [[ "$output" == *"PASS  run-niagara-test  verdict: 1 test summary, 6 tests run, 0 failures" ]]
}

@test "RNT2: BUILD SUCCESSFUL with 0 tests on standard input is FAIL, exit 1" {
  # Mutation: RNT2 -- a wrapper that drops the Go tool's exit code (no exec, a trailing exit 0) passes this run
  run bash -c "printf 'Total tests run: 0\nBUILD SUCCESSFUL\n' | '$R' verdict -"
  [ "$status" -eq 1 ] || { echo "$output"; return 1; }
  [[ "$output" == *"FAIL  run-niagara-test  line 1: run 0, failures -, skips -; no tests ran (Total tests run: 0)"* ]]
}
