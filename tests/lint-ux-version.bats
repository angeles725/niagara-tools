#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-ux-version.sh
bats_require_minimum_version 1.5.0
# A -ux module's local resources carry ?v=<defaultModuleVersion>: the servlet caches images for 30 days, so a stale
# suffix keeps a kiosk on the old file (types/dashboard.md "Cache-bust every static asset with the module version";
# types/frontend-standard.md §5, §11). The rule's logic has golden cases in
# go/internal/lint/rules/uxversion/testdata; these pin the wrapper contract on committed fixture modules.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  UXV="$KIT/toolbelt/lint-ux-version.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-ux-version"
  UX="Mod/Mod-ux/src/rc"
}

@test "UXV-help: --help and -h print the usage on stdout and exit 0" {
  run "$UXV" --help
  [ "$status" -eq 0 ]
  [[ "$output" == "Usage:  lint-ux-version.sh [--strict] <module-root>"* ]]
  [[ "$output" == *"stale-version|no-version"* ]]
  [[ "$output" == *"Exit:"* ]]
  [[ "$output" != *"Mutation:"* ]]
  run "$UXV" -h
  [ "$status" -eq 0 ]
  [[ "$output" == "Usage:  lint-ux-version.sh"* ]]
}

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "UXV-usage: no argument prints the usage on stderr and exits 3" {
  run --separate-stderr "$UXV"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [ "$stderr" = "usage: lint-ux-version.sh [--strict] <module-root>" ]
}

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "UXV-nondir: a missing root and an unknown option exit 3 with their message on stderr" {
  run --separate-stderr "$UXV" "$FX/nope"
  [ "$status" -eq 3 ]
  [ "$stderr" = "lint-ux-version: not a directory: $FX/nope" ]
  run --separate-stderr "$UXV" --bogus "$FX/clean"
  [ "$status" -eq 3 ]
  [ "$stderr" = "lint-ux-version: unknown option: --bogus" ]
}

@test "UXV-clean: matching suffixes, an external URL and a data: URI exit 0 with no output" {
  run "$UXV" "$FX/clean"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "UXV-external: an external URL and a data: URI are never a reference" {
  # clean/ holds https://cdn.example.org/lib.js and data:image/png with no ?v=; the run stays silent.
  grep -q 'cdn.example.org' "$FX/clean/$UX/index.html"
  run "$UXV" "$FX/clean"
  [ "$status" -eq 0 ]
  [[ "$output" != *"cdn.example.org"* ]]
  [[ "$output" != *"no-version"* ]]
}

@test "UXV-stale: a ?v= that differs from defaultModuleVersion is a FAIL row (exit 1), a missing one a WARN" {
  run "$UXV" "$FX/stale"
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "FAIL  lint-ux-version  $FX/stale/$UX/css/base.css:1  stale-version: ../img/bg.png?v=0.9 has ?v=0.9 but defaultModuleVersion is 1.2.3 (types/frontend-standard.md §11)" ]
  [ "${lines[1]}" = "FAIL  lint-ux-version  $FX/stale/$UX/index.html:2  stale-version: css/base.css?v=1.2.2 has ?v=1.2.2 but defaultModuleVersion is 1.2.3 (types/frontend-standard.md §11)" ]
  [ "${lines[2]}" = "WARN  lint-ux-version  $FX/stale/$UX/index.html:4  no-version: img/logo.png has no ?v=1.2.3 suffix (types/dashboard.md: cache-bust every static asset with the module version)" ]
  [ "${#lines[@]}" -eq 3 ]
}

@test "UXV-stale-subdir: the module directory under the gradle root finds the version above it" {
  run "$UXV" "$FX/stale/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"$FX/stale/Mod/Mod-ux/src/rc/index.html:2  stale-version: css/base.css?v=1.2.2 has ?v=1.2.2 but defaultModuleVersion is 1.2.3"* ]]
}

@test "UXV-missing: a local reference with no ?v= is a WARN row and the exit stays 0" {
  run "$UXV" "$FX/missing"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-ux-version  $FX/missing/$UX/index.html:2  no-version: js/main.js has no ?v=1.2.3 suffix (types/dashboard.md: cache-bust every static asset with the module version)" ]
}

@test "UXV-strict: --strict (before or after the root) prints the WARN as FAIL and exits 1" {
  run "$UXV" --strict "$FX/missing"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-ux-version  $FX/missing/$UX/index.html:2  no-version: js/main.js has no ?v=1.2.3 suffix (types/dashboard.md: cache-bust every static asset with the module version)" ]
  run "$UXV" "$FX/missing" --strict
  [ "$status" -eq 1 ]
  [[ "$output" == "FAIL  lint-ux-version  "* ]]
  run "$UXV" --strict "$FX/clean"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "UXV-waiver: an allow marker with a reason prints a PASS row and exit 0" {
  run "$UXV" "$FX/waived"
  [ "$status" -eq 0 ]
  [ "$output" = "PASS  lint-ux-version  $FX/waived/$UX/index.html:2  stale-version: js/old.js?v=0.1 allowed: pinned legacy build" ]
}

@test "UXV-vendor: rc/vendor and *.min.js are never read" {
  run "$UXV" "$FX/vendor"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "UXV-empty: a root with no *-ux/src/rc file is an ERROR row and exit 3" {
  run "$UXV" "$FX/empty"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-ux-version  $FX/empty  no *-ux/src/rc *.html, *.js or *.css files found (wrong path or no -ux module)" ]
}

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "UXV-noversion: a build.gradle.kts with no defaultModuleVersion is exit 3, never a pass" {
  run --separate-stderr "$UXV" "$FX/noversion"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [ "$stderr" = "lint-ux-version: no defaultModuleVersion(\"X.Y.Z\") found in a build.gradle.kts under or above $FX/noversion" ]
}

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "UXV-ambiguous: children with different versions and none at the root is exit 3, never the first" {
  run --separate-stderr "$UXV" "$FX/ambiguous"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [ "$stderr" = "lint-ux-version: ambiguous defaultModuleVersion: 1.0.0 in $FX/ambiguous/a, 2.0.0 in $FX/ambiguous/b" ]
}

@test "UXV-entity: &amp; in an attribute value is read as & before the query is split" {
  run "$UXV" "$FX/entity"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  lint-ux-version  $FX/entity/$UX/index.html:2  stale-version: js/b.js?x=1&v=0.1 has ?v=0.1 but defaultModuleVersion is 1.2.3 (types/frontend-standard.md §11)" ]
}
