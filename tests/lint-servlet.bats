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

@test "LSV2e: a try in another handler, or one closed before the parse, does not guard it (input-400)" {
  # Mutation: LSV2e -- a 5-line look-back for "try {" lets doGet's try guard doPost's Integer.parseInt(
  # Go golden: go/internal/lint/rules/servlet/testdata/truth-try-scope-1.txt
  printf '%s\n' 'package com.x;' 'public class BSrv extends BWebServlet {' '  public void doGet(WebOp op) {' \
    '    try { a(); } catch (Exception e) { op.getResponse().sendError(400); }' '  }' '  public void doPost(WebOp op) {' \
    '    int b = Integer.parseInt(op.get("b"));' '  }' '}' > "$ONE/BSrv.java"
  run "$LS" "$ONE"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  input-400  BSrv.java:7  parseDouble/parseInt without enclosing try/catch -> may not return 400 on bad input" ]
}

@test "LSV2f: the catch that decides catch-no-400 is the one attached to the parse's try that catches it" {
  # Mutation: LSV2f -- taking the first catch within 4 lines of the try's end judges a multi-catch by its IOException clause
  # Go golden: go/internal/lint/rules/servlet/testdata/truth-catch-structure-1.txt
  printf '%s\n' 'package com.x;' 'public class BCat extends BWebServlet {' '  void multi(String s) {' '    try { Long.parseLong(s); }' \
    '    catch (IOException e) { op.getResponse().sendError(400); }' \
    '    catch (IllegalStateException | NumberFormatException e) { v = 0; }' '  }' '}' > "$ONE/BCat.java"
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  catch-no-400  BCat.java:6  catch around parse without 400 response (silent default may write bad value)" ]
}

@test "LSV2g: a parse inside a try-with-resources is in a try (no input-400)" {
  # Mutation: LSV2g -- finding the try by the "try {" regex FAILs a parse inside try (Reader r = open()) { ... }
  # Go golden: go/internal/lint/rules/servlet/testdata/truth-try-resources-1.txt
  printf '%s\n' 'package com.x;' 'public class BRes extends BWebServlet {' '  void body(WebOp op) {' '    try (Reader r = open()) {' \
    '      int a = Integer.parseInt(op.get("a"));' '    } catch (NumberFormatException e) { op.getResponse().sendError(400); }' \
    '  }' '}' > "$ONE/BRes.java"
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
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

@test "LSV-walk-locked: a sub-directory find cannot enter is an env error (exit 3) naming the root, never a clean pass" {
  # Mutation: LSV-walk-locked -- a find | sort process substitution drops find's status and exits 0.
  only WithAuth.java
  mkdir -p "$ONE/locked"; cp "$FX/NoAuth.java" "$ONE/locked/"
  chmod 000 "$ONE/locked"
  if [ -r "$ONE/locked" ]; then chmod 755 "$ONE/locked"; skip "running as root"; fi
  run "$LS" "$ONE"
  chmod 755 "$ONE/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"lint-servlet: cannot list every file under $ONE: "*"locked"* ]]
}

@test "LSV-worktrees: a *-worktrees sub-directory is pruned, so a checkout copy never doubles a finding" {
  # Mutation: LSV-worktrees -- a walk without the *-worktrees prune prints the auth FAIL twice.
  only NoAuth.java
  mkdir -p "$ONE/mod-worktrees/feat"; cp "$FX/NoAuth.java" "$ONE/mod-worktrees/feat/"
  run "$LS" "$ONE"
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c 'FAIL  auth')" -eq 1 ]
}

@test "LSV-empty: a <src-dir> with no *.java is an env error (exit 3, ERROR row), never a clean pass" {
  # The K20 row of lib/scan-files.sh scan_files_nonempty_or_exit, kept by the Go port (Spec.EmptyError).
  rm -rf "$ONE"; mkdir -p "$ONE"; echo x > "$ONE/a.txt"
  run "$LS" "$ONE"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-servlet  $ONE  no files to scan (K20: wrong path or empty scaffold)" ]
}

@test "LSV-dotroot: a <src-dir> whose name starts with a dot is scanned, not pruned with it" {
  # The walk prunes dot sub-directories only (find -mindepth 1); a private find -name '.*' -prune once
  # matched the start point too and read no file under such a root.
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

@test "LSV2d: a \"} catch\" on the try's closing line around a parse with a silent default WARNs (catch-no-400)" {
  rm -rf "$ONE"; mkdir -p "$ONE"
  printf '%s\n' 'public class S extends BWebServlet {' '  void x(String s) {' '    try {' \
    '      v = Double.parseDouble(s);' '    } catch (NumberFormatException e) {' '      v = 0;' '    }' '  }' '}' > "$ONE/S.java"
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  catch-no-400  S.java:5  "* ]]
}

@test "LSV2h: a catch that re-reads the parse input (enum ordinal-or-tag) is no catch-no-400 row; a constant return still WARNs" {
  # Mutation: LSV2h -- dropping the re-read check WARNs the enum tag fallback again (PANCCADIA BDashboardServlet.java:623).
  # [ev: issue #459 catch-no-400]
  rm -rf "$ONE"; mkdir -p "$ONE"
  printf '%s\n' 'public class S extends BWebServlet {' '  BValue e(BEnum e, String tag) {' \
    '    try { return e.getRange().get((int) Double.parseDouble(tag), false); }' \
    '    catch (NumberFormatException nfe) { return e.getRange().get(tag); }' '  }' \
    '  double d(String s) {' '    try { return Double.parseDouble(s.trim()); }' \
    '    catch (NumberFormatException x) { return 0.0; }' '  }' '}' > "$ONE/S.java"
  run "$LS" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"S.java:4  "* ]]
  [[ "$output" == *"WARN  catch-no-400  S.java:8  "* ]]
}
