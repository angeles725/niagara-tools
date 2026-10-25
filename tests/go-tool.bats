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

# --- What counts as a source: every file under <go-dir> except bin/, testdata/ and *_test.go. `go build`
# never compiles a *_test.go file or reads testdata/, so editing them must not cost a rebuild; an embedded
# asset (//go:embed) is part of the binary, so editing it must. [ev: retro r10-go-tool-walk-scope]

@test "GT10: an edited *_test.go file does not rebuild the binary" {
  # Mutation: GT10 -- dropping the *_test.go exclusion rebuilds (and needs go) after a test-only edit.
  _bin
  [ "$status" -eq 0 ]
  touch -d '2000-01-01' "$MOD/cmd/hello/main.go" "$MOD/go.mod"
  printf 'package main\n' > "$MOD/cmd/hello/main_test.go"
  _bin "$BATS_TEST_TMPDIR/nogo"
  [ "$status" -eq 0 ]
  [ "$output" = "$MOD/bin/hello" ]
}

@test "GT11: an edited file under testdata/ does not rebuild the binary" {
  # Mutation: GT11 -- dropping the testdata prune rebuilds (and needs go) after a fixture edit.
  _bin
  [ "$status" -eq 0 ]
  mkdir -p "$MOD/cmd/hello/testdata"
  printf 'package fixture\n' > "$MOD/cmd/hello/testdata/case.go"
  printf 'fixture\n' > "$MOD/cmd/hello/testdata/case.txt"
  _bin "$BATS_TEST_TMPDIR/nogo"
  [ "$status" -eq 0 ]
  [ "$output" = "$MOD/bin/hello" ]
}

@test "GT12: an edited //go:embed asset rebuilds the binary" {
  # Mutation: GT12 -- watching only *.go and go.mod keeps serving the old embedded asset.
  printf 'package main\n\nimport (\n\t_ "embed"\n\t"fmt"\n)\n\n//go:embed msg.txt\nvar msg string\n\nfunc main() { fmt.Print(msg) }\n' \
    > "$MOD/cmd/hello/main.go"
  printf 'a1\n' > "$MOD/cmd/hello/msg.txt"
  _bin
  [ "$status" -eq 0 ]
  [ "$("$MOD/bin/hello")" = a1 ]
  touch -d '2000-01-01' "$MOD/bin/hello"
  touch -d '1999-01-01' "$MOD/cmd/hello/main.go" "$MOD/go.mod"
  printf 'a2\n' > "$MOD/cmd/hello/msg.txt"
  _bin
  [ "$status" -eq 0 ]
  [ "$("$MOD/bin/hello")" = a2 ]
}

@test "GT13: a non-test .go file edited beside a test file rebuilds the binary" {
  _bin
  printf 'package main\n' > "$MOD/cmd/hello/main_test.go"
  touch -d '2000-01-01' "$MOD/bin/hello"
  touch -d '1999-01-01' "$MOD/cmd/hello/main_test.go" "$MOD/go.mod"
  _src v2
  _bin
  [ "$status" -eq 0 ]
  [ "$("$MOD/bin/hello")" = v2 ]
}

@test "GT14: no non-test Go file under go/ embeds from a testdata/ directory (the walk prunes them)" {
  # The prune is safe only while this holds: a pattern naming testdata, or an embedded directory that holds
  # one, would put a pruned file into a binary.
  local f line pat hits=0 n=0 dir
  local -a pats
  while IFS= read -r f; do
    dir="${f%/*}"
    while IFS= read -r line; do
      read -r -a pats <<< "${line#*//go:embed}"
      for pat in "${pats[@]}"; do
        n=$((n + 1))
        pat="${pat#all:}"
        if [[ "$pat" == *testdata* ]] || { [ -d "$dir/$pat" ] && [ -n "$(find "$dir/$pat" -name testdata -print -quit)" ]; }; then
          printf 'embeds testdata: %s: %s\n' "$f" "$pat" >&2
          hits=$((hits + 1))
        fi
      done
    done < <(grep -E '^[[:space:]]*//go:embed[[:space:]]' "$f")
  done < <(grep -rlE --include='*.go' '^[[:space:]]*//go:embed[[:space:]]' "$ROOT/go" | grep -v '_test\.go$')
  [ "$n" -gt 0 ]
  [ "$hits" -eq 0 ]
}
