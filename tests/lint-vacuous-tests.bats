#!/usr/bin/env bats
# lint-vacuous-tests.bats — a srcTest/ @Test method that cannot fail (only trivially true assertions, no assertion)
# or never runs (its first statement always skips it) is a FAIL row, unless allowed by name with a reason; the
# rule's golden cases live in go/internal/lint/rules/vacuoustests. [ev: retro r10-lint-vacuous-tests Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-vacuous-tests.sh"
  ROOT="$BATS_TEST_TMPDIR/ModuleA"
  T="$ROOT/ModuleA-rt/srcTest/com/example/test"
  mkdir -p "$T"
}

USAGE='usage: lint-vacuous-tests.sh [--allow <Class.method>]... [--allow-reason <why>] <gradle-root>'

# test_class <name> <method body>... writes one @Test method per body into $T/<name>.java.
test_class() {
  local name=$1 i=0 body
  shift
  {
    printf '%s\n' 'package com.example.test;' 'import org.testng.annotations.Test;' "public class $name {"
    for body in "$@"; do
      i=$((i + 1))
      printf '%s\n' '  @Test' "  public void m$i() { $body }"
    done
    printf '%s\n' '  private void check(int v) { org.testng.Assert.assertTrue(v > 0); }' '}'
  } > "$T/$name.java"
}

@test "VT-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "$USAGE" ]
}

@test "VT1: assertTrue(true) and a same-literal assertEquals are a trivially-true FAIL; a real assertion is clean" {
  # Mutation: VT1 -- dropping the trivially-true check reads assertTrue(true) as a real assertion (exit 0)
  test_class BAlphaTest 'assertTrue(true);' 'assertEquals("a", "a");' 'assertEquals(compute(), 2);'
  run "$L" "$ROOT"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "FAIL  trivially-true  $T/BAlphaTest.java:5  BAlphaTest.m1: its only assertions are trivially true, so it cannot fail" ]
  [ "${lines[1]}" = "FAIL  trivially-true  $T/BAlphaTest.java:7  BAlphaTest.m2: its only assertions are trivially true, so it cannot fail" ]
}

@test "VT2: a test with no assertion FAILs; one that asserts through a same-class helper is clean" {
  # Mutation: VT2 -- not resolving helpers flags the check(...) test as no-assertion
  test_class BBetaTest 'compute();' 'check(compute());'
  run "$L" "$ROOT"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "${lines[0]}" == "FAIL  no-assertion  $T/BBetaTest.java:5  BBetaTest.m1: no assertion "* ]]
}

@test "VT3: assumeTrue(false), assumeFalse(true) and a first-statement SkipException are an always-skipped FAIL" {
  # Mutation: VT3 -- dropping the always-skipped check reads assumeTrue(false) as a test that runs
  test_class BGammaTest 'org.junit.Assume.assumeTrue(false);' 'assumeFalse(true);' 'throw new SkipException("later");' \
    'assumeTrue(isLinux()); check(1);'
  run "$L" "$ROOT"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 3 ]
  [ "${lines[0]}" = "FAIL  always-skipped  $T/BGammaTest.java:5  BGammaTest.m1: its first statement always skips it (org.junit.Assume.assumeTrue(false)), so it never runs" ]
  [ "${lines[1]}" = "FAIL  always-skipped  $T/BGammaTest.java:7  BGammaTest.m2: its first statement always skips it (assumeFalse(true)), so it never runs" ]
  [ "${lines[2]}" = "FAIL  always-skipped  $T/BGammaTest.java:9  BGammaTest.m3: its first statement always skips it (throw new SkipException), so it never runs" ]
}

@test "VT4: --allow names a flagged method (PASS row with the reason); a stale --allow FAILs; no reason is exit 3" {
  # Mutation: VT4 -- skipping the stale check passes an --allow that names a test which is not flagged
  test_class BAlphaTest 'assertTrue(true);' 'check(2);'
  run "$L" --allow BAlphaTest.m1 --allow-reason 'placeholder until the station test' "$ROOT"
  [ "$status" -eq 0 ]
  [ "$output" = "PASS  allowed  $T/BAlphaTest.java:5  BAlphaTest.m1: trivially-true, allowed: placeholder until the station test" ]
  run "$L" --allow BAlphaTest.m1 --allow BAlphaTest.m2 --allow-reason why "$ROOT"
  [ "$status" -eq 1 ]
  [ "${lines[1]}" = "FAIL  stale-allow  --allow BAlphaTest.m2 names no flagged test method (remove it)" ]
  run "$L" --allow BAlphaTest.m1 "$ROOT"
  [ "$status" -eq 3 ]
}

@test "VT5-static: a test asserting only through a static-imported srcTest helper is clean; a non-asserting one FAILs" {
  # Mutation: VT5-static -- not resolving static imports flags the static-imported await(...) test as no-assertion
  printf '%s\n' 'package com.example.test;' 'import static org.testng.Assert.assertTrue;' 'public class ModuleAFixture {' \
    '  static void await(boolean c) { assertTrue(c); }' '}' > "$T/ModuleAFixture.java"
  printf '%s\n' 'package com.example.test;' 'public class ModuleANoop {' \
    '  static void await(boolean c) { System.out.println(c); }' '}' > "$T/ModuleANoop.java"
  printf '%s\n' 'package com.example.test;' 'import static com.example.test.ModuleAFixture.*;' \
    'import org.testng.annotations.Test;' 'public class BDeltaTest {' '  @Test' '  public void m1() { await(compute() > 0); }' '}' \
    > "$T/BDeltaTest.java"
  printf '%s\n' 'package com.example.test;' 'import static com.example.test.ModuleANoop.*;' \
    'import org.testng.annotations.Test;' 'public class BEpsilonTest {' '  @Test' '  public void m1() { await(compute() > 0); }' '}' \
    > "$T/BEpsilonTest.java"
  run "$L" "$ROOT"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "${lines[0]}" == "FAIL  no-assertion  $T/BEpsilonTest.java:6  BEpsilonTest.m1: no assertion "* ]]
}

@test "VT-env: an unparseable source and a root with no srcTest/ *.java are exit 3" {
  printf '%s\n' 'class BBadTest {' '  /* never closed' > "$T/BBadTest.java"
  run "$L" "$ROOT"
  [ "$status" -eq 3 ]
  [ "$output" = "lint-vacuous-tests: cannot scan $T/BBadTest.java: it ends inside a /* comment" ]
  rm -r "$ROOT/ModuleA-rt/srcTest"
  mkdir -p "$ROOT/ModuleA-rt/src"
  printf 'class A { }\n' > "$ROOT/ModuleA-rt/src/A.java"
  run "$L" "$ROOT"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-vacuous-tests  $ROOT  no *.java under a srcTest/ (wrong path: pass a gradle or module root)" ]
}
