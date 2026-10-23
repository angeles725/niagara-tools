#!/usr/bin/env bats
# RED-FIRST pins for lint-servlet.sh (campaign 8 PR12, B813). Static security lint over BWebServlet
# subclasses under <src>. Real shapes: DashboardServlet.java:168/274/354-358/421-429, BChiServlet.java:613.
#
# Checks (row grammar `<check>  PASS|FAIL|WARN|SKIP  <subject>  <detail>`):
#   auth          FAIL  a doGet/doPost that writes/resolves with no getRemoteUser()/op.getUser() gate
#   input-400     FAIL  parseDouble/parseInt on a request param with no try/catch that returns 400
#   unbounded-set WARN  BComponent.set*/set on the request thread with no facet MIN/MAX enforcement
#   cache-nofinger WARN Cache-Control max-age>0 on an rc asset with no fingerprint
#   log-in-handler WARN LOG.info inside a request handler (per-request log spam)
#   exit 0 no FAIL (WARN-only still 0) / 1 any FAIL / 3 usage.
#
# RED today: lint-servlet.sh does not exist -> every pin fails for the right reason (tool absent).
# NAMED MUTATIONS (post-green): drop the auth check -> LSV1 no longer FAILs; drop the input-400 check
# -> LSV2 no longer FAILs; remove the dot-dir prune -> LSV-prune's stale NoAuth is flagged (D9b).

load helpers/symroot   # assert_symroot_same <dir> <cmd...> (issue #329)

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LS="$KIT/toolbelt/lint-servlet.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-servlet"
  ONE="$BATS_TEST_TMPDIR/one"; mkdir -p "$ONE"
}
only() { rm -rf "$ONE"; mkdir -p "$ONE"; cp "$FX/$1" "$ONE/"; }

@test "LSV1: a doPost that writes with no auth gate FAILs (auth), exit 1" {
  only NoAuth.java
  run "$LS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"auth"* ]] && [[ "$output" == *"NoAuth"* ]]
}

@test "LSV1c: a handler with an op.getUser() gate does NOT FAIL auth" {
  only WithAuth.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "LSV2: parseDouble on a request param with no try/catch->400 FAILs (input-400)" {
  only BadParse.java
  run "$LS" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"input-400"* ]]
}

@test "LSV3: a set* on the request thread with no MIN/MAX enforcement WARNs (unbounded-set), exit 0" {
  only UnboundedSet.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"unbounded-set"* ]]
}

@test "LSV5: Cache-Control max-age>0 on an unfingerprinted rc asset WARNs (cache-nofinger)" {
  only CacheNoFinger.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"cache-nofinger"* ]]
}

@test "LSV6: LOG.info inside a request handler WARNs (log-in-handler)" {
  only LogInHandler.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"log-in-handler"* ]]
}

@test "LSV4: X-Requested-With guard without CsrfUtil/csrfToken WARNs (csrf-xrw-only), exit 0" {
  only CsrfXrwOnly.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"csrf-xrw-only"* ]]
}

@test "LSV-usage: no <src> argument -> exit 3" {
  run "$LS"
  [ "$status" -eq 3 ]
}

@test "LSV-prune: a NoAuth copy under .deploy-baseline/ is NOT flagged (D9b dot-dir prune)" {
  rm -rf "$ONE"; mkdir -p "$ONE/.deploy-baseline"
  cp "$FX/WithAuth.java" "$ONE/"                       # a clean live servlet -> the tool scans something
  cp "$FX/NoAuth.java" "$ONE/.deploy-baseline/Stale.java"   # a stale unsafe copy -> must be pruned
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
  [[ "$output" != *"Stale"* ]]
}

@test "LSV2b: catch around parseDouble returning silent default WARNs (catch-no-400), exit 0" {
  only CatchNo400.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]] && [[ "$output" == *"catch-no-400"* ]]
}

@test "LSV2b-clean: catch around parseDouble with 400 response does NOT warn (catch-no-400)" {
  only CatchWith400.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"catch-no-400"* ]]
}

# shellcheck disable=SC2016
@test "LSV7: doGet with getWriter() but no X-Content-Type-Options is an INFO row (api-response-headers), exit 0" {
  # Station header providers reach every module servlet (TridiumSecurityFilter on /*), so a missing
  # per-servlet header is information, not a finding (register C13, corpus B1349 C1).
  only ApiRespNoHeader.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"INFO  api-response-headers"* ]]
  [[ "$output" != *"WARN  api-response-headers"* ]]
}

# shellcheck disable=SC2016
@test "LSV7-strict: --strict does not escalate the api-response-headers INFO row" {
  only ApiRespNoHeader.java
  run "$LS" "$ONE" --strict
  [ "$status" -eq 0 ]
  [[ "$output" == *"INFO  api-response-headers"* ]]
  [[ "$output" != *"FAIL  api-response-headers"* ]]
}

# shellcheck disable=SC2016
@test "LSV7-clean: doGet that sets X-Content-Type-Options does NOT warn (api-response-headers)" {
  only ApiRespWithHeader.java
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"api-response-headers"* ]]
}

@test "LSV-symroot: a symlinked <src-dir> is walked like find -H, same rows as the directory (issue #329)" {
  # Mutation: LSV-symroot -- os.Lstat instead of os.Stat on the root in go/internal/lint/walk.go Walk lists no file under a symlinked root.
  only NoAuth.java
  assert_symroot_same "$ONE" "$LS" "$ONE"
  [ "$status" -eq 1 ] && [[ "$output" == *"FAIL"*"NoAuth"* ]]
}

@test "LSV-empty: a <src-dir> with no *.java is an env error (exit 3, ERROR row), never a clean pass" {
  # Declared difference 2026-10-23 (Go engine, #351): the bash lint exited 0 with no row.
  rm -rf "$ONE"; mkdir -p "$ONE"; echo x > "$ONE/a.txt"
  run "$LS" "$ONE"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-servlet  $ONE  no *.java sources found (wrong path or empty source tree)" ]
}

@test "LSV-dotroot: a <src-dir> whose name starts with a dot is scanned, not pruned with it" {
  # Declared difference 2026-10-23: find -name '.*' -prune also matched the start point, so the bash lint
  # read no file under such a root and exited 0.
  mkdir -p "$BATS_TEST_TMPDIR/.src"; cp "$FX/NoAuth.java" "$BATS_TEST_TMPDIR/.src/"
  run "$LS" "$BATS_TEST_TMPDIR/.src"
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  auth  NoAuth.java:4  "* ]]
}

@test "LSV4b: an X-Requested-With named only in a Javadoc is no guard, so no csrf-xrw-only row (truth audit T8)" {
  # The chihuahua BChiServlet.java:41 row anchored on a Javadoc line; the guard lives in another file.
  rm -rf "$ONE"; mkdir -p "$ONE"
  printf '%s\n' '/**' ' * Security: /api/* requires X-Requested-With: XMLHttpRequest' ' */' \
    'public class Doc extends BWebServlet {' '  void x() { }' '}' > "$ONE/Doc.java"
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"csrf-xrw-only"* ]]
}

@test "LSV2c: a parse call in a comment or a string literal is no input-400 FAIL (code only)" {
  rm -rf "$ONE"; mkdir -p "$ONE"
  printf '%s\n' 'public class C extends BWebServlet {' '  // double v = Double.parseDouble(raw);' \
    '  /* Integer.parseInt(raw) */' '  String hint = "use Integer.parseInt(";' '}' > "$ONE/C.java"
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"input-400"* ]]
}
