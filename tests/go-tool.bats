#!/usr/bin/env bats
# go-tool.bats — go/go-tool.sh `go_tool_bin`, the build-on-demand helper behind every thin wrapper
# over a go/cmd/<tool> binary (toolbelt/gen-retro-index.sh, scripts/ci-local.sh -> ci-steps).
# Why: the rebuild paths of the first wrapper were untested (RDD review of #288); a stale binary
# that is silently reused, or a missing toolchain that falls through, runs old logic.
# [ev: retro go-ci-steps]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  H="$ROOT/go/go-tool.sh"
  MOD="$BATS_TEST_TMPDIR/mod"
  mkdir -p "$MOD/cmd/hello" "$BATS_TEST_TMPDIR/nogo"
  printf 'module example.com/t\n\ngo 1.21\n' > "$MOD/go.mod"
  _src v1
  # A PATH with the helper's tools but no go.
  local t
  for t in bash find head mkdir mv rm; do ln -s "$(command -v "$t")" "$BATS_TEST_TMPDIR/nogo/$t"; done
}

_src() { printf 'package main\n\nimport "fmt"\n\nfunc main() { fmt.Println("%s") }\n' "$1" > "$MOD/cmd/hello/main.go"; }
# _bin [PATH] — run go_tool_bin for the fixture module, optionally under another PATH.
# shellcheck disable=SC2016  # $1/$2 expand in the inner bash
_bin() { run env PATH="${1:-$PATH}" bash -c '. "$1" && go_tool_bin "$2" hello' _ "$H" "$MOD"; }

@test "GT1: a missing binary is built and its path printed" {
  _bin
  [ "$status" -eq 0 ]
  [ "$output" = "$MOD/bin/hello" ]
  [ "$("$MOD/bin/hello")" = v1 ]
  run find "$MOD/bin" -name '*.tmp.*'
  [ -z "$output" ]
}

@test "GT2: a binary newer than every source is reused without a toolchain" {
  _bin
  [ "$status" -eq 0 ]
  _bin "$BATS_TEST_TMPDIR/nogo"
  [ "$status" -eq 0 ]
  [ "$output" = "$MOD/bin/hello" ]
}

@test "GT3: a source newer than the binary rebuilds it" {
  # Mutation: GT3 -- testing only `[ -x "$bin" ]` keeps running the old binary after a source edit.
  _bin
  touch -d '2000-01-01' "$MOD/bin/hello"
  _src v2
  _bin
  [ "$status" -eq 0 ]
  [ "$("$MOD/bin/hello")" = v2 ]
}

@test "GT4: a go.mod newer than the binary rebuilds it" {
  _bin
  touch -d '2000-01-01' "$MOD/bin/hello"
  _src v3
  touch -d '2000-01-01' "$MOD/cmd/hello/main.go"
  touch "$MOD/go.mod"
  _bin
  [ "$status" -eq 0 ]
  [ "$("$MOD/bin/hello")" = v3 ]
}

@test "GT5: a stale binary and no go on PATH is exit 3, and the old binary stays" {
  _bin
  touch -d '2000-01-01' "$MOD/bin/hello"
  _src v2
  _bin "$BATS_TEST_TMPDIR/nogo"
  [ "$status" -eq 3 ]
  [[ "$output" == *"hello: go is not on PATH; it is needed to build $MOD/bin/hello"* ]]
  [ "$("$MOD/bin/hello")" = v1 ]
}

@test "GT6: no go.mod is exit 3 naming the directory" {
  rm "$MOD/go.mod"
  _bin
  [ "$status" -eq 3 ]
  [[ "$output" == *"hello: no Go module at $MOD"* ]]
}

@test "GT7: a failed build is exit 3, leaves no temporary file and keeps the old binary" {
  _bin
  touch -d '2000-01-01' "$MOD/bin/hello"
  printf 'package main\n\nfunc main() { syntax error }\n' > "$MOD/cmd/hello/main.go"
  _bin
  [ "$status" -eq 3 ]
  [[ "$output" == *"hello: go build failed in $MOD"* ]]
  [ "$("$MOD/bin/hello")" = v1 ]
  run find "$MOD/bin" -name '*.tmp.*'
  [ -z "$output" ]
}

@test "GT8: build output goes to stderr, so stdout carries only the binary path" {
  # shellcheck disable=SC2016  # $1/$2 expand in the inner bash
  run env bash -c '. "$1" && go_tool_bin "$2" hello 2>/dev/null' _ "$H" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "$MOD/bin/hello" ]
}

@test "GT9: a find error in the staleness check is exit 3, never a silent reuse of the binary" {
  # Mutation: GT9 -- a find piped into `head` loses its exit status and reuses a stale binary.
  [ "$(id -u)" -ne 0 ] || skip "root ignores directory permissions"
  _bin
  [ "$status" -eq 0 ]
  mkdir "$MOD/cmd/locked"
  chmod 000 "$MOD/cmd/locked"
  _bin
  chmod 755 "$MOD/cmd/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"hello: cannot check whether $MOD/bin/hello is current"* ]]
}
