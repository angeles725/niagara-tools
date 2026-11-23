#!/usr/bin/env bats
# Guards toolbelt/run-pure-test.sh — the standalone WSL pure-test runner.
# These BITE: a fixture test that FAILS must make the runner exit non-zero (the runner
# must not swallow a JUnit failure), a passing one must exit 0, and a missing/broken
# environment must report the right exit code. All fixtures are tiny → sub-second.

# The script is the launcher of go/cmd/run-pure-test: build the binary once, with the real environment, so a test
# that changes HOME never has to (P6 runs it under an empty HOME).
setup_file() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via ROOT
  . "$ROOT/go/go-tool.sh"
  go_tool_bin "$ROOT/go" run-pure-test >/dev/null
}

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

@test "P7: a test that does not COMPILE exits 3 (environment), never 1 — a compile error is not a RED (audit A1)" {
  # Mutation: P7 -- letting set -e propagate javac's status reports a compile error as exit 1, the "test FAILED" bite.
  make_test "Adder.nope(2,2) == 4"
  run "$RUN" "$RT" com.example.demo.AdderTest
  [ "$status" -eq 3 ]
  [[ "$output" == *"does not compile"* ]]
}

@test "P2-exact: a FAILING pure test exits exactly 1 (audit A1)" {
  make_test "Adder.add(2,2) == 5"
  run "$RUN" "$RT" com.example.demo.AdderTest
  [ "$status" -eq 1 ]
}

# #944: the kit's servlet-free request/response recorder (toolbelt/java/n4kit/http), put on the javac source path
# by the launcher, so StaticHandler-style header logic is unit-tested with no servlet API jar. The module code
# reaches it through the two small interfaces; its servlet doGet is a thin adapter that implements them.
write_static_headers() {
  cat > "$RT/src/$PKG/StaticHeaders.java" <<'JAVA'
package com.example.demo;
import n4kit.http.HttpRequestView;
import n4kit.http.HttpResponseSink;
final class StaticHeaders {
  static void apply(HttpRequestView req, HttpResponseSink out, String path) {
    String ae = req.getHeader("Accept-Encoding");
    boolean gz = path.endsWith(".js") && ae != null && ae.contains("gzip");
    out.setStatus(200);
    out.setContentType("application/javascript");
    out.setHeader("Cache-Control", "public, max-age=31536000, immutable");
    out.addHeader("Vary", "Accept-Encoding");
    if (gz) out.setHeader("Content-Encoding", "gzip");
  }
}
JAVA
}

@test "H1: the recorder keeps the status and the headers a StaticHandler-style helper sets, with no servlet API" {
  write_static_headers
  cat > "$RT/srcTest/test/com/example/demo/StaticHeadersTest.java" <<'JAVA'
package com.example.demo;
import n4kit.http.HttpRecorder;
import org.junit.Test; import static org.junit.Assert.*;
public class StaticHeadersTest {
  @Test public void gzip_twin_headers() {
    HttpRecorder r = HttpRecorder.get("/rc/dist/app.js").withHeader("accept-encoding", "gzip, br");
    StaticHeaders.apply(r, r, "/rc/dist/app.js");
    assertEquals(200, r.status());
    assertEquals("gzip", r.header("Content-Encoding"));
    assertEquals("public, max-age=31536000, immutable", r.header("cache-control"));
    assertEquals("Accept-Encoding", r.header("VARY"));
    assertEquals("application/javascript", r.contentType());
  }
  @Test public void plain_file_has_vary_and_no_content_encoding() {
    HttpRecorder r = HttpRecorder.get("/rc/dist/app.js");
    StaticHeaders.apply(r, r, "/rc/dist/app.js");
    assertNull(r.header("Content-Encoding"));
    assertEquals("Accept-Encoding", r.header("Vary"));
  }
  @Test public void set_replaces_add_appends_and_an_unset_status_is_200() {
    HttpRecorder r = HttpRecorder.get("/x");
    assertEquals(200, r.status());
    r.setHeader("A", "1"); r.setHeader("a", "2"); r.addHeader("A", "3");
    assertEquals(java.util.Arrays.asList("2", "3"), r.headers("A"));
    assertEquals("2", r.header("A"));
    r.setStatus(304);
    assertEquals(304, r.status());
    assertTrue(r.headers("Absent").isEmpty());
    assertNull(r.header("Absent"));
  }
}
JAVA
  run "$RUN" "$RT" com.example.demo.StaticHeadersTest
  [ "$status" -eq 0 ]
  [[ "$output" == *"OK (3 tests)"* ]]
}

@test "H2: a header assertion on the recorder BITES: a wrong Vary is exit 1, not a pass" {
  write_static_headers
  cat > "$RT/srcTest/test/com/example/demo/StaticHeadersTest.java" <<'JAVA'
package com.example.demo;
import n4kit.http.HttpRecorder;
import org.junit.Test; import static org.junit.Assert.*;
public class StaticHeadersTest {
  @Test public void wrong_vary() {
    HttpRecorder r = HttpRecorder.get("/a.js");
    StaticHeaders.apply(r, r, "/a.js");
    assertEquals("Origin", r.header("Vary"));
  }
}
JAVA
  run "$RUN" "$RT" com.example.demo.StaticHeadersTest
  [ "$status" -eq 1 ]
  [[ "$output" == *"expected:<[Origin]> but was:<[Accept-Encoding]>"* ]]
}
