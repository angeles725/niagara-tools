#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/build.sh
# Toolchain-free: covers only the argument-parsing + env-check layer (before gradle / JDK /
# niagara_home are actually invoked). No real gradle, JDK, or Niagara home is required.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  BUILD="$KIT/toolbelt/build.sh"
}

@test "BUILD-usage: no args exits 2" {
  run bash "$BUILD"
  [ "$status" -eq 2 ]
}

@test "BUILD-unknown-flag: unknown flag exits 2" {
  run bash "$BUILD" --unknown-flag
  [ "$status" -eq 2 ]
}

@test "BUILD-missing-root: missing module-root dir exits 10" {
  run bash "$BUILD" /nonexistent/path/does-not-exist SomeMod
  [ "$status" -eq 10 ]
}

@test "BUILD-no-preflight-accepted: --no-preflight parsed (no positional args still exits 2)" {
  run bash "$BUILD" --no-preflight
  [ "$status" -eq 2 ]
}

@test "BUILD-no-report-accepted: --no-report parsed (no positional args still exits 2)" {
  run bash "$BUILD" --no-report
  [ "$status" -eq 2 ]
}
