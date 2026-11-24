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

# A stub bin/nre.exe: it records its arguments and the stdin it read, prints $NRE_OUT and exits $NRE_EXIT (0).
stub_nre() {
  # shellcheck disable=SC2016  # the stub is a script body: its $ expands when IT runs
  printf '%s\n' '#!/bin/sh' 'printf "%s\n" "$@" > "$NRE_LOG"' 'cat > "$NRE_STDIN"' \
    'printf "%s" "$NRE_OUT"' 'exit "${NRE_EXIT:-0}"' > "$H/bin/nre.exe"
  chmod +x "$H/bin/nre.exe"
  export NRE_STDIN="$BATS_TEST_TMPDIR/stdin.txt" NRE_OUT="$1"
  # the Windows-path conversion is deterministic: a fake wslpath leads PATH
  mkdir -p "$BATS_TEST_TMPDIR/tools"
  # shellcheck disable=SC2016  # the stub is a script body: its $ expands when IT runs
  printf '%s\n' '#!/bin/sh' 'printf "W:%s\n" "$2"' > "$BATS_TEST_TMPDIR/tools/wslpath"
  chmod +x "$BATS_TEST_TMPDIR/tools/wslpath"
  PATH="$BATS_TEST_TMPDIR/tools:$PATH"
}

@test "NT3: list names every supported tool with a purpose, no install needed" {
  run "$T" list
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | awk '{print $1}')" = "$(printf 'deps\npalette\nrecompile')" ]
  [[ "$output" == *"DependencyChecker"* && "$output" == *"PaletteGenerator"* && "$output" == *"RecompileTool"* ]]
}

@test "NT4: an unknown tool is refused with exit 3 and the supported list" {
  run "$T" depcheck "$H" ModuleA
  [ "$status" -eq 3 ]
  [[ "$output" == *"unknown tool depcheck (known: deps, list, palette, recompile)"* ]]
  [ ! -e "$NRE_LOG" ]
}

@test "NT5: palette runs PaletteGenerator on a module part dir, says the file it writes, nre's exit code is the exit" {
  stub_nre ""
  P="$BATS_TEST_DIRNAME/fixtures/nre-tool/ModuleA-rt"
  run "$T" palette "$H" "$P"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PaletteGenerator writes $P/module.palette"* ]]
  [ "$(cat "$NRE_LOG")" = "$(printf '%s\n' devkit:com.tridium.devkit.util.PaletteGenerator "W:$P")" ]
  NRE_EXIT=7 run "$T" palette "$H" "$P"
  [ "$status" -eq 7 ]
}

@test "NT6: palette refuses a directory that is not a module part, and runs nothing" {
  stub_nre ""
  run "$T" palette "$H" "$BATS_TEST_DIRNAME/fixtures/nre-tool"
  [ "$status" -eq 3 ]
  [[ "$output" == *"is not a module part directory (needs module-include.xml and src/)"* ]]
  [ ! -e "$NRE_LOG" ]
}

@test "NT7: recompile passes the target, -alias: and -tsaUrl:, and hands stdin to nre for the password prompt" {
  stub_nre "--- Processed x!"
  B="$BATS_TEST_TMPDIR/a.bog"
  : > "$B"
  run "$T" recompile "$H" "$B" -alias:lab -tsaUrl:http://tsa.example <<< "typed-secret"
  [ "$status" -eq 0 ]
  [ "$(cat "$NRE_LOG")" = "$(printf '%s\n' program:com.tridium.program.ui.RecompileTool "W:$B" -alias:lab -tsaUrl:http://tsa.example)" ]
  [ "$(cat "$NRE_STDIN")" = "typed-secret" ]
}

@test "NT8: recompile refuses a password in argv, without echoing it, and runs nothing" {
  stub_nre "--- Processed x!"
  B="$BATS_TEST_TMPDIR/a.bog"
  : > "$B"
  run "$T" recompile "$H" "$B" -password:hunter2
  [ "$status" -eq 3 ]
  [[ "$output" == *"never passes the password in argv; nre prompts for it"* ]]
  [[ "$output" != *hunter2* ]]
  [ ! -e "$NRE_LOG" ]
}

@test "NT9: a RecompileTool failure fails the step: nre's exit code, or exit 1 when nre exits 0 after an ERROR line" {
  B="$BATS_TEST_TMPDIR/a.bog"
  : > "$B"
  stub_nre ""
  NRE_EXIT=4 run "$T" recompile "$H" "$B" < /dev/null
  [ "$status" -eq 4 ]
  stub_nre "ERROR Recompiling p"
  run "$T" recompile "$H" "$B" < /dev/null
  [ "$status" -eq 1 ]
}
