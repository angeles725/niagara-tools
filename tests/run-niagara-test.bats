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

@test "RNT3: --testng on a results file whose setup failed and skipped its tests is FAIL, exit 1" {
  printf '<testng-results skipped="1" failed="0" ignored="0" total="1" passed="0"><suite name="S"><test name="T">%s%s</test></suite></testng-results>\n' \
    '<class name="test.com.example.ModuleA.BExampleStationTest"><test-method status="FAIL" name="setupStation" is-config="true"/>' \
    '<test-method status="SKIP" name="reads_value"/></class>' > "$BATS_TEST_TMPDIR/testng-results.xml"
  run "$R" verdict --testng "$BATS_TEST_TMPDIR/testng-results.xml"
  [ "$status" -eq 1 ] || { echo "$output"; return 1; }
  [[ "$output" == *"FAIL  run-niagara-test  testng: test-method setupStation FAIL (config)"* ]]
}

@test "RNT4: run refuses an install that is not on a Windows drive with exit 2, before copying anything" {
  mkdir -p "$BATS_TEST_TMPDIR/home/bin" && : > "$BATS_TEST_TMPDIR/home/bin/test.exe"
  run "$R" run --install "$BATS_TEST_TMPDIR/home" --gradle-root "$BATS_TEST_TMPDIR" --part ModuleA-rt --module ModuleA
  [ "$status" -eq 2 ] || { echo "$output"; return 1; }
  [[ "$output" == *"--install $BATS_TEST_TMPDIR/home: not on a Windows drive"* ]]
}

@test "RNT5: run takes only low for --module-verification: any other value is exit 2, before copying anything" {
  run "$R" run --install 'C:\Niagara\N4' --gradle-root "$BATS_TEST_TMPDIR" --part ModuleA-rt --module ModuleA --module-verification medium
  [ "$status" -eq 2 ] || { echo "$output"; return 1; }
  [[ "$output" == *'--module-verification "medium": the only value is low'* ]]
}

@test "RNT6: run refuses an --exclude-jar glob that is not a jar basename with exit 2, before copying anything" {
  run "$R" run --install 'C:\Niagara\N4' --gradle-root "$BATS_TEST_TMPDIR" --part ModuleA-rt --module ModuleA --exclude-jar 'mods\moduleA-*.jar'
  [ "$status" -eq 2 ] || { echo "$output"; return 1; }
  [[ "$output" == *'--exclude-jar "mods\\moduleA-*.jar" is not a jar basename glob'* ]]
}

@test "RNT7: run takes --restore-gradle-root as a flag without a value: a value after it is exit 2, before copying anything" {
  run "$R" run --install 'C:\Niagara\N4' --gradle-root "$BATS_TEST_TMPDIR" --part ModuleA-rt --module ModuleA --restore-gradle-root yes
  [ "$status" -eq 2 ] || { echo "$output"; return 1; }
  [[ "$output" == *'run: unexpected argument "yes"'* ]]
}

@test "RNT8: run --log-dir needs a value: a bare --log-dir is exit 2, before copying anything" {
  run "$R" run --install 'C:\Niagara\N4' --gradle-root "$BATS_TEST_TMPDIR" --part ModuleA-rt --module ModuleA --log-dir
  [ "$status" -eq 2 ] || { echo "$output"; return 1; }
  [[ "$output" == *'flag needs an argument: -log-dir'* ]]
}

@test "RNT9: --testng fails a results file with ignored test-methods unless --allow-ignored-count waives exactly that count" {
  # Mutation: RNT9 -- dropping the ignored check (judgeIgnored) turns the first run green
  printf '<testng-results skipped="0" failed="0" ignored="1" total="2" passed="1"><suite name="S"><test name="T">%s</test></suite></testng-results>\n' \
    '<class name="test.com.example.ModuleA.BExampleStationTest"><test-method status="PASS" name="reads_value"/></class>' > "$BATS_TEST_TMPDIR/testng-results.xml"
  run "$R" verdict --testng "$BATS_TEST_TMPDIR/testng-results.xml"
  [ "$status" -eq 1 ] || { echo "$output"; return 1; }
  [[ "$output" == *"FAIL  run-niagara-test  testng: 1 ignored test-method (@Test(enabled = false) or excluded"* ]]
  run "$R" verdict --testng "$BATS_TEST_TMPDIR/testng-results.xml" --allow-ignored-count 1 --allow-ignored-reason 'needs a live device'
  [ "$status" -eq 0 ] || { echo "$output"; return 1; }
  [[ "$output" == *"PASS  run-niagara-test  testng: 1 ignored test-method allowed (--allow-ignored-count 1): needs a live device"* ]]
  run "$R" verdict --testng "$BATS_TEST_TMPDIR/testng-results.xml" --allow-ignored-count 1
  [ "$status" -eq 2 ] || { echo "$output"; return 1; }
  [[ "$output" == *"--allow-ignored-count needs --allow-ignored-reason <why>"* ]]
}
