#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# ci-local-filesize.bats — `scripts/ci-local.sh --changed` runs the Go file-size gate when a go/**/*.go changed
# (CLF1-CLF4, issue #969). The gate (go/internal/filesize) only ran in CI's Go job, so a file over the limit showed up
# after review and push. WARN is a file over 200 lines (printed, never failing); FAIL is a file over 300 lines.
# The fixture is a throwaway repository: this checkout's HEAD without go/, a small go/ of its own (the go.mod, go-tool.sh,
# the prebuilt ci-steps and host-budget binaries and a copy of internal/filesize), and this work tree's ci-local files.

setup_file() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime
  . "$REPO/go/go-tool.sh"
  local t; for t in ci-steps host-budget kit-state; do go_tool_bin "$REPO/go" "$t" >/dev/null; done
}

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  R="$BATS_TEST_TMPDIR/r"
  mkdir -p "$R" "$BATS_TEST_TMPDIR/cg/init.scope" "$BATS_TEST_TMPDIR/gocache"
  git -C "$REPO" archive HEAD -- ':(exclude)go' | tar -x -C "$R"
  mkdir -p "$R/go/internal/filesize" "$R/go/bin"
  cp "$REPO/go/go.mod" "$REPO/go/go-tool.sh" "$R/go/"
  cp "$REPO"/go/internal/filesize/{filesize.go,filesize_test.go} "$R/go/internal/filesize/"
  printf '# no baseline\n' > "$R/go/internal/filesize/baseline.txt"
  rm -rf "$R/scripts/lib"
  cp "$REPO/scripts/ci-local.sh" "$R/scripts/"
  cp -R "$REPO/scripts/lib" "$R/scripts/"
  cp "$REPO/go/bin/ci-steps" "$REPO/go/bin/host-budget" "$REPO/go/bin/kit-state" "$R/go/bin/"
  # newer than every file the cases add, so go_tool_bin never tries to rebuild them from this fixture's small go/
  touch -d 'tomorrow' "$R"/go/bin/*
  printf 'name: CI\non: pull_request\njobs:\n  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - name: shellcheck\n        run: shellcheck scripts/*.sh .githooks/*\n' \
    > "$R/.github/workflows/ci.yml"
  printf '16\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.current"
  printf '48129\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.max"
  export CI_LOCAL_CGROUP_ROOT="$BATS_TEST_TMPDIR/cg"
  export CI_LOCAL_HOST_BUDGET_ARGS="--proc-root $REPO/tests/fixtures/host-budget/proc --gocache $BATS_TEST_TMPDIR/gocache --disk-min-free 1"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid
  git -C "$R" init -q -b main
  printf 'go/bin/\n' >> "$R/.git/info/exclude"
  git -C "$R" add -A
  git -C "$R" commit -q -m base
  git -C "$R" update-ref refs/remotes/origin/main HEAD
}

# go_file <rel> <lines> — a gofmt-clean Go file of exactly <lines> lines under the fixture's go/.
go_file() {
  mkdir -p "$(dirname "$R/go/$1")"
  { printf 'package x\n\n'; local i; for ((i = 2; i < $2; i++)); do printf '// line\n'; done; } > "$R/go/$1"
}

@test "CLF1: a changed Go file of 201 lines prints its WARN line and the change still passes" {
  # Mutation: CLF1 -- dropping run_filesize from the --changed branch leaves a Go change unchecked until CI.
  go_file internal/x/mid.go 201
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS go test ./internal/filesize"* ]]
  [ "$(grep -c '^WARN internal/x/mid.go holds 201 lines' <<< "$output")" -eq 1 ]
}

@test "CLF2: a changed Go file of 300 lines warns and passes, one of 301 lines fails --changed with the split advice" {
  # Mutation: CLF2 -- ignoring run_filesize's status passes a file over 300 lines.
  go_file internal/x/edge.go 300
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN internal/x/edge.go holds 300 lines"* ]]
  go_file internal/x/edge.go 301
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"internal/x/edge.go holds 301 lines, over the limit of 300: split it by responsibility"* ]]
  [[ "$output" == *"FAIL go test ./internal/filesize"* ]]
  [[ "$output" == *"the file-size gate"* ]]
  [[ "$output" != *"WARN internal/x/edge.go"* ]]
}

@test "CLF3: a changed Go file of 200 lines passes with no WARN line" {
  go_file internal/x/ok.go 200
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS go test ./internal/filesize"* ]]
  [[ "$output" != *"WARN "* ]]
}

@test "CLF4: a change with no Go file does not run the file-size gate (unchanged)" {
  printf 'x\n' > "$R/scripts/notes.txt"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"no changed file needs shellcheck, the bats lints or gofmt"* ]]
  [[ "$output" != *"filesize"* ]]
}
