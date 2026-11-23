#!/usr/bin/env bats
# nre-tool.bats — the launcher of nre-tool (issue #851). The unit cases (parsing, raw, faults) live in
# go/internal/nretool; here a fake install whose bin/nre.exe is a stub proves the launcher builds the binary and runs
# the right nre class from the install's bin directory.

setup() {
  T="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/nre-tool.sh"
  H="$BATS_TEST_TMPDIR/niagara"
  mkdir -p "$H/bin"
  # shellcheck disable=SC2016  # the stub is a script body: its $ expands when IT runs
  printf '%s\n' '#!/bin/sh' 'printf "%s\n" "$PWD" "$@" > "$NRE_LOG"' \
    'printf "INFO [nre] Booting\r\n\r\nDependencies for %s: \r\n\r\n    baja\r\n    web-rt\r\n" "$2"' > "$H/bin/nre.exe"
  chmod +x "$H/bin/nre.exe"
  export NRE_LOG="$BATS_TEST_TMPDIR/nre.log"
}

@test "NT1: deps lists the module parts, one per line, from the install's DependencyChecker" {
  run "$T" deps "$H" ModuleA
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf 'baja\nweb-rt')" ]
  [ "$(cat "$NRE_LOG")" = "$(printf '%s\n' "$H/bin" devkit:com.tridium.devkit.util.DependencyChecker ModuleA)" ]
}

@test "NT2: no nre under the home is a usage error, exit 3" {
  run "$T" deps "$BATS_TEST_TMPDIR" ModuleA
  [ "$status" -eq 3 ]
  [[ "$output" == *"no bin/nre.exe or bin/nre under"* ]]
}
