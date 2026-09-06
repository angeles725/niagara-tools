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
