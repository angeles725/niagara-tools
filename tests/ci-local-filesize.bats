#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# ci-local-filesize.bats — `scripts/ci-local.sh --changed` runs the Go file-size gate when a go/**/*.go changed
# (CLF1-CLF6, issue #969). The gate (go/internal/filesize) only ran in CI's Go job, so a file over the limit showed up
# after review and push. The rule is research-sdd's debt list: a production file over 200 lines needs a line in
# go/internal/filesize/modularity-debt.txt (a count of 400 or less is a ceiling, a count over 400 is exact, nothing is
# admitted above 400); a file at 200 lines or less needs none.
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
  cp "$REPO"/go/internal/filesize/*.go "$R/go/internal/filesize/"
  printf '# no debt\n' > "$R/go/internal/filesize/modularity-debt.txt"
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

# debt <line>... — the fixture's debt list, one entry per argument.
debt() {
  { printf '# no debt\n'; printf '%s\n' "$@"; } > "$R/go/internal/filesize/modularity-debt.txt"
}

@test "CLF1: a changed Go file of 201 lines and no debt line fails --changed with the line to add" {
  # Mutation: CLF1 -- dropping run_filesize from the --changed branch leaves a Go change unchecked until CI.
  go_file internal/x/mid.go 201
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"internal/x/mid.go: 201 lines, over 200 and not listed: split it by responsibility"* ]]
  [[ "$output" == *"internal/x/mid.go 201 accepted <reason>"* ]]
  [[ "$output" == *"FAIL go test ./internal/filesize"* ]]
  [[ "$output" == *"ci-local: --changed: 1 check(s) failed"* ]]
  [[ "$output" != *"PASS go test ./internal/filesize"* ]]
}

@test "CLF2: with its line the 201-line file passes, and raising it needs the line raised in the same change" {
  # Mutation: CLF2 -- ignoring run_filesize's status passes a file that outgrew its recorded count.
  debt "internal/x/mid.go 300 accepted the reason it stays whole"
  go_file internal/x/mid.go 300
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS go test ./internal/filesize"* ]]
  go_file internal/x/mid.go 301
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"internal/x/mid.go: grew from 300 to 301 lines: split it by responsibility, or raise its line"* ]]
  [[ "$output" == *"internal/x/mid.go 301 accepted the reason it stays whole"* ]]
  debt "internal/x/mid.go 301 accepted the reason it stays whole"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
}

@test "CLF3: a changed Go file of 200 lines passes with no debt line" {
  go_file internal/x/ok.go 200
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS go test ./internal/filesize"* ]]
  [[ "$output" != *"not listed"* ]]
}

@test "CLF4: a change with no Go file does not run the file-size gate (unchanged)" {
  printf 'x\n' > "$R/scripts/notes.txt"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"no shellcheck, bats-lint or gofmt file changed; running the tree gates"* ]]
  [[ "$output" != *"filesize"* ]]
}

@test "CLF5: a Go file over 400 lines cannot be listed and a listed file at 200 lines needs its line deleted" {
  go_file internal/x/huge.go 401
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"internal/x/huge.go: 401 lines, over 400: a file over 400 lines cannot be listed, split it by responsibility"* ]]
  go_file internal/x/huge.go 200
  debt "internal/x/huge.go 300 accepted the reason it stays whole"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"internal/x/huge.go: 200 lines, no longer over 200: delete this line from"* ]]
}

@test "CLF6: a legacy count over 400 is exact: the file that shrank needs its line lowered" {
  debt "internal/x/old.go 500 legacy recorded when the ratchet was created"
  go_file internal/x/old.go 480
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"internal/x/old.go: 480 lines, below its recorded 500"* ]]
  [[ "$output" == *"internal/x/old.go 480 legacy recorded when the ratchet was created"* ]]
}
