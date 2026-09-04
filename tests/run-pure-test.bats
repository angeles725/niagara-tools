#!/usr/bin/env bats
# Guards toolbelt/run-pure-test.sh — the standalone WSL pure-test runner.
# These BITE: a fixture test that FAILS must make the runner exit non-zero (the runner
# must not swallow a JUnit failure), a passing one must exit 0, and a missing/broken
# environment must report the right exit code. All fixtures are tiny → sub-second.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  RUN="$KIT/toolbelt/run-pure-test.sh"
  # Skip the whole suite (don't fake a pass) if JUnit isn't fetched on this machine.
  find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | grep -q . \
    || skip "junit-4.13.2 not in ~/.gradle cache (run one gradle build to fetch it)"

  RT="$BATS_TEST_TMPDIR/mod-rt"
  PKG=com/example/demo
  mkdir -p "$RT/src/$PKG" "$RT/srcTest/test/$PKG"
  # A ZERO-Baja pure decision core.
  cat > "$RT/src/$PKG/Adder.java" <<'JAVA'
package com.example.demo;
final class Adder { static int add(int a, int b) { return a + b; } }
JAVA
}

# writes a test whose sole assertion is $1 (a Java boolean expr) then compiles+runs it
make_test() {
  cat > "$RT/srcTest/test/com/example/demo/AdderTest.java" <<JAVA
package com.example.demo;
import org.junit.Test; import static org.junit.Assert.*;
public class AdderTest { @Test public void t() { assertTrue($1); } }
JAVA
}

@test "P1: a PASSING pure test exits 0 and reports OK" {
  make_test "Adder.add(2,2) == 4"
  run "$RUN" "$RT" com.example.demo.AdderTest
  [ "$status" -eq 0 ]
  [[ "$output" == *"OK (1 test)"* ]]
}

@test "P2: a FAILING pure test exits NON-ZERO (the runner must not swallow it)" {
  make_test "Adder.add(2,2) == 5"
  run "$RUN" "$RT" com.example.demo.AdderTest
  [ "$status" -ne 0 ]
  [[ "$output" == *"FAILURES"* ]] || [[ "$output" == *"Tests run: 1,  Failures: 1"* ]]
}

@test "P3: a missing test source exits 3 (environment), not 0" {
  run "$RUN" "$RT" com.example.demo.DoesNotExistTest
  [ "$status" -eq 3 ]
  [[ "$output" == *"test source not found"* ]]
}

@test "P4: wrong arg count exits 2 (usage)" {
  run "$RUN" "$RT"
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage"* ]]
}

@test "P5: a non-existent module-rt-dir exits 3 (environment)" {
  run "$RUN" "$BATS_TEST_TMPDIR/nope" com.example.demo.AdderTest
  [ "$status" -eq 3 ]
}
