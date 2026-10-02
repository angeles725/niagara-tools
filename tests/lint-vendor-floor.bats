#!/usr/bin/env bats
# Pins for lint-vendor-floor.sh — vendored browser libraries (rc/vendor/**/*.js) parsed at the
# Chromium 83 panel floor (acorn ecmaVersion 2020) plus a token scan for APIs above the floor.
# [ev: retro dashboard-frontend-standard Δ16]
#
# Rows:  FAIL  vendor-floor  <file>:<line>  syntax: ...   (does not parse at ecmaVersion 2020)
#        WARN  vendor-floor  <file>:<line>  api: ...      (API above the floor; may be feature-guarded)
#        SKIP  vendor-floor  <rc-dir>  unavailable: ...   (node or acorn missing -> exit 4, never a pass)
# Exit:  0 no FAIL · 1 any FAIL · 3 usage/env · 4 tool unavailable
#
# The acorn-dependent pins skip with a reason when acorn is not resolvable (CI has no npm install);
# the degrade path (VF3/VF3b) is forced through KIT_NODE / KIT_ACORN and runs everywhere node does.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LVF="$KIT/toolbelt/lint-vendor-floor.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/vendor-floor"
}

_need_node() { command -v node >/dev/null 2>&1 || skip "node not installed"; }
_need_acorn() {
  _need_node
  if [ -n "${KIT_ACORN:-}" ] || [ -d "$KIT/toolbelt/eslint/node_modules/acorn" ]; then return 0; fi
  node -e 'require.resolve("acorn")' >/dev/null 2>&1 \
    || skip "acorn not installed (npm install --prefix build-n4-module-kit/toolbelt/eslint)"
}

@test "VF1: no argument -> usage exit 3" {
  run "$LVF"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage"* ]]
}

@test "VF2: a non-directory argument -> exit 3" {
  run "$LVF" "$FX/does-not-exist"
  [ "$status" -eq 3 ]
}

@test "VF3: node missing -> typed SKIP row naming node and exit 4 (never a silent pass)" {
  KIT_NODE=/nonexistent/node run "$LVF" "$FX/syntax"
  [ "$status" -eq 4 ]
  [[ "$output" == *"SKIP"* ]] && [[ "$output" == *"vendor-floor"* ]] && [[ "$output" == *"unavailable"* ]]
  [[ "$output" == *"node"* ]]
}

@test "VF3b: acorn missing -> typed SKIP row naming acorn and exit 4" {
  _need_node
  KIT_ACORN=/nonexistent/acorn run "$LVF" "$FX/syntax"
  [ "$status" -eq 4 ]
  [[ "$output" == *"SKIP"* ]] && [[ "$output" == *"acorn"* ]]
}

@test "VF4: an rc dir with no vendor/ -> exit 0 before any tool is needed" {
  KIT_NODE=/nonexistent/node run "$LVF" "$FX/novendor"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]] && [[ "$output" != *"SKIP"* ]]
}

@test "VF5: logical assignment (ES2021) in a vendored lib -> FAIL syntax row, exit 1" {
  _need_acorn
  run "$LVF" "$FX/syntax"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]] && [[ "$output" == *"vendor-floor"* ]]
  [[ "$output" == *"bad-2.0.0.min.js:2"* ]] && [[ "$output" == *"syntax"* ]]
}

@test "VF6: APIs above the floor -> one WARN api row each, exit stays 0" {
  _need_acorn
  run "$LVF" "$FX/api"
  [ "$status" -eq 0 ]
  [[ "$output" == *"api-3.0.0.js:2"* ]] && [[ "$output" == *"replaceAll"* ]]
  [[ "$output" == *"api-3.0.0.js:3"* ]] && [[ "$output" == *"hasOwn"* ]]
  [[ "$output" == *"api-3.0.0.js:4"* ]] && [[ "$output" == *"findLast"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 3 ]
}

@test "VF7: an ES2020 lib (?. ?? class BigInt) with API names only in comments/strings -> clean exit 0" {
  _need_acorn
  run "$LVF" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]] && [[ "$output" != *"FAIL"* ]]
}

@test "VF8: --strict promotes the API WARN rows to FAIL (exit 1)" {
  _need_acorn
  run "$LVF" "$FX/api" --strict
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL')" -eq 3 ]
}
