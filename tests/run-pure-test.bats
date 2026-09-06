#!/usr/bin/env bats
# Guards toolbelt/run-pure-test.sh — the standalone WSL pure-test runner.
# These BITE: a fixture test that FAILS must make the runner exit non-zero (the runner
# must not swallow a JUnit failure), a passing one must exit 0, and a missing/broken
# environment must report the right exit code. All fixtures are tiny → sub-second.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  RUN="$KIT/toolbelt/run-pure-test.sh"
  # Skip locally (don't fake a pass) when JUnit isn't fetched; fail in CI so a missing
  # pre-fetch step is loud instead of silent.
  if ! find "$HOME/.gradle" -name 'junit-4.13.2.jar' 2>/dev/null | grep -q .; then
    if [ -n "${CI:-}" ]; then
      echo "CI: junit-4.13.2.jar not in ~/.gradle — ci.yml pre-fetch step is required" >&2
      false
    else
      skip "junit-4.13.2 not in ~/.gradle cache (run one gradle build to fetch it)"
    fi
  fi

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

@test "P6: an empty ~/.gradle cache (no junit) exits 3 with an actionable message" {
  # GREEN path needs junit in ~/.gradle; this asserts the RED path (empty cache) is a
  # clean env error, never a false pass. Point HOME at an empty dir so the script's
  # `find $HOME/.gradle` finds nothing.
  make_test "Adder.add(2,2) == 4"
  run env HOME="$BATS_TEST_TMPDIR/emptyhome" "$RUN" "$RT" com.example.demo.AdderTest
  [ "$status" -eq 3 ]
  [[ "$output" == *"junit-4.13.2.jar not in"* ]]
}

# ===========================================================================
# S24 (C10): run-pure-test.sh must run the JVM with cwd = the module-rt-dir, so a
# SOURCE-STRUCTURAL test (one that reads `src/…` via Paths.get, e.g. FreezeAlarm-
# WiringTest / ConfigLoginWiringTest) resolves its relative paths regardless of
# where the runner was invoked. On df8c7ec the runner runs `java …` from the
# caller's cwd (toolbelt/run-pure-test.sh:62, no `cd "$rt"`), so from a non-profile
# cwd the structural test's `Paths.get("src/…")` misses and it FAILs.
# RED-for-the-right-reason: S24-cwd FAILs on df8c7ec (runner uses the caller cwd).
# ---------------------------------------------------------------------------
@test "S24-cwd: a source-structural test (Paths.get(\"src/…\")) passes via run-pure-test.sh from a NON-profile cwd" {
  cat > "$RT/src/$PKG/Widget.java" <<'JAVA'
package com.example.demo;
final class Widget { static int v() { return 7; } }
JAVA
  cat > "$RT/srcTest/test/com/example/demo/WidgetStructTest.java" <<'JAVA'
package com.example.demo;
import org.junit.Test; import static org.junit.Assert.*;
import java.nio.file.*;
public class WidgetStructTest {
  @Test public void reads_its_own_source_relative_to_the_module_root() {
    // resolves ONLY when the JVM cwd is the module-rt-dir (like the real wiring tests)
    assertTrue("src/ must resolve from the module root", Files.exists(Paths.get("src/com/example/demo/Widget.java")));
  }
}
JAVA
  # invoke the runner from a NON-profile cwd; on the fix it cd's into $RT, so Paths.get("src/…") resolves
  mkdir -p "$BATS_TEST_TMPDIR/elsewhere"; cd "$BATS_TEST_TMPDIR/elsewhere"
  run "$RUN" "$RT" com.example.demo.WidgetStructTest
  [ "$status" -eq 0 ]
  [[ "$output" == *"OK (1 test)"* ]]
}

@test "S24-cwd-regression: the same runner call from the module-rt-dir itself still passes (unchanged)" {
  cat > "$RT/src/$PKG/Widget2.java" <<'JAVA'
package com.example.demo;
final class Widget2 { static int v() { return 9; } }
JAVA
  cat > "$RT/srcTest/test/com/example/demo/Widget2StructTest.java" <<'JAVA'
package com.example.demo;
import org.junit.Test; import static org.junit.Assert.*;
import java.nio.file.*;
public class Widget2StructTest {
  @Test public void t() { assertTrue(Files.exists(Paths.get("src/com/example/demo/Widget2.java"))); }
}
JAVA
  cd "$RT"
  run "$RUN" "$RT" com.example.demo.Widget2StructTest
  [ "$status" -eq 0 ]
}
