#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# ci-local-gates.bats — `scripts/ci-local.sh --changed` runs the tree gates of go/internal/cisteps/gates.go for the
# files changed (CG1-CG7, issues #983 item 5 and #973 item 3). The cardwidth gate (the 120-character limit on kit docs)
# ran only in CI, so a METHODOLOGY.md line over 120 broke PR #978 after the push; the summary did not count the gates.
# The fixture is a throwaway repository: this checkout's HEAD without go/, a go/ of links to this checkout's (its
# README.md copied: cardwidth walks the fixture's tree and counts it) and this work tree's ci-local files. Its workflow
# has one shellcheck step and one stand-in for ci.yml's gen-changelog step, so nothing is downloaded.

load helpers/ci-local-golinks

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
  ci_local_link_go "$REPO" "$R"
  rm -rf "$R/scripts/lib"
  cp "$REPO/scripts/ci-local.sh" "$R/scripts/"
  cp -R "$REPO/scripts/lib" "$R/scripts/"
  printf 'name: CI\non: pull_request\njobs:\n  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - name: shellcheck\n        run: shellcheck scripts/*.sh .githooks/*\n      - name: gen-changelog --check (stand-in)\n        run: echo stand-in gate failed; exit 1\n' \
    > "$R/.github/workflows/ci.yml"
  printf '16\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.current"
  printf '48129\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.max"
  export CI_LOCAL_CGROUP_ROOT="$BATS_TEST_TMPDIR/cg"
  export CI_LOCAL_HOST_BUDGET_ARGS="--proc-root $REPO/tests/fixtures/host-budget/proc --gocache $BATS_TEST_TMPDIR/gocache --disk-min-free 1"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid
  git -C "$R" init -q -b main
  git -C "$R" add -A
  git -C "$R" commit -q -m base
  git -C "$R" update-ref refs/remotes/origin/main HEAD
  BASE="$(git -C "$R" rev-parse HEAD)"
}

# long_line — one prose line of 199 characters (a card line the limit refuses: every space is breakable).
long_line() { local i; for ((i = 0; i < 40; i++)); do printf 'word '; done; printf 'end\n'; }

@test "CG1: a kit .md line over 120 characters fails --changed with the cardwidth message, locally and as pushed" {
  # Mutation: CG1 -- dropping the cardwidth gate from gates.go lets a #978-style long line through to CI.
  long_line >> "$R/build-n4-module-kit/BUILD-LOOP.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"build-n4-module-kit/BUILD-LOOP.md must stay wrapped: 1 overlong line(s)"* ]]
  [[ "$output" == *"FAIL go test ./internal/cardwidth"* ]]
  git add -A
  git commit -q -m "docs: long line"
  run bash scripts/ci-local.sh --changed --range "$BASE..$(git rev-parse HEAD)"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL go test ./internal/cardwidth"* ]]
}

@test "CG2: a wrapped kit .md passes the cardwidth gate" {
  printf '\nA short line.\n' >> "$R/build-n4-module-kit/BUILD-LOOP.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS go test ./internal/cardwidth"* ]]
  [[ "$output" == *"ci-local: --changed: 0 check(s) failed"* ]]
}

@test "CG3: a change with no .md does not run the cardwidth gate" {
  # Mutation: CG3 -- a **/* trigger makes every push pay for the cardwidth test.
  printf 'x\n' > "$R/scripts/notes.txt"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" != *"cardwidth"* ]]
}

@test "CG4: a change over 400 lines prints the pr-size WARN and still exits 0; a small one prints its count" {
  # Mutation: CG4 -- counting the soft pr-size gate as a failure blocks a push the size:exception label would allow.
  printf 'x\n' > "$R/scripts/notes.txt"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"pr-size: total 0 changed lines"* ]]
  seq 1 450 > "$R/scripts/big.txt"
  git add -A
  git commit -q -m "chore: big"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"pr-size: total 451 changed lines"*"exceeds the 400-line limit"* ]]
  [[ "$output" == *"WARN pr-size"* ]]
}

@test "CG5: the summary counts every failed gate" {
  # Mutation: CG5 -- a gate that does not count leaves the summary at 0 check(s) failed beside a FAIL line.
  long_line >> "$R/build-n4-module-kit/BUILD-LOOP.md"
  mkdir -p "$R/changelog.d"
  printf '%s\n' '### Fixed' '- a fixture fragment.' > "$R/changelog.d/2026-11-24-fixture.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL go test ./internal/cardwidth"* ]]
  [[ "$output" == *"FAIL gen-changelog --check"* ]]
  [[ "$output" == *"stand-in gate failed"* ]]
  [[ "$output" == *"ci-local: --changed: 2 check(s) failed"* ]]
}

@test "CG6: a gate whose ci.yml step the workflow lacks is skipped with its name, not failed" {
  mkdir -p "$R/changelog.d"
  printf '%s\n' '### Fixed' '- a fixture fragment.' > "$R/changelog.d/2026-11-24-b.md"
  sed -i '/stand-in/,$d' "$R/.github/workflows/ci.yml"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"SKIP gen-changelog --check (this workflow has no step gen-changelog*)"* ]]
}

@test "CG7: a pr-size that cannot measure (exit 3, exit 2) is a WARN with its exit code and --changed exits 0" {
  # Mutation: CG7 -- downgrading only exit 1 lets any other pr-size exit block a push the soft gate promises never to block.
  printf 'x\n' > "$R/scripts/notes.txt"
  cd "$R"
  PR_SIZE_LIMIT=abc run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"pr-size: PR_SIZE_LIMIT must be a positive integer"* ]]
  [[ "$output" == *"WARN pr-size"*"(exit 3: the size was not measured)"* ]]
  [[ "$output" != *"FAIL pr-size"* ]]
  [[ "$output" == *"ci-local: --changed: 0 check(s) failed"* ]]
  # a stub that exits 2, committed to the base so that no .sh counts as changed
  printf '#!/usr/bin/env bash\necho "pr-size: stub cannot measure" >&2\nexit 2\n' > scripts/pr-size.sh
  git add scripts/pr-size.sh
  git commit -q -m "stub pr-size"
  git update-ref refs/remotes/origin/main HEAD
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"pr-size: stub cannot measure"* ]]
  [[ "$output" == *"WARN pr-size"*"(exit 2: the size was not measured)"* ]]
  [[ "$output" != *"FAIL pr-size"* ]]
  [[ "$output" == *"ci-local: --changed: 0 check(s) failed"* ]]
}

@test "CG8: run from a git hook (GIT_DIR and GIT_INDEX_FILE exported), a gate's own git calls see their own repository" {
  # Mutation: CG8 -- without clearing git's local env vars, a gate's own git calls (a go test fixture repository)
  # resolve to the pushing repository: the pre-push replay fails where CI passes, and fixture commits land in it.
  cd "$R"
  # shellcheck disable=SC2016  # the $(...) is the workflow step's own shell, not this test's
  printf 'name: CI\non: pull_request\njobs:\n  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - name: shellcheck\n        run: shellcheck scripts/*.sh .githooks/*\n      - name: gen-changelog --check (stand-in)\n        run: d="$(mktemp -d)"; git -C "$d" init -q; test "$(git -C "$d" rev-parse --absolute-git-dir)" = "$d/.git"\n' \
    > .github/workflows/ci.yml
  git add -A
  git commit -q -m wf
  git update-ref refs/remotes/origin/main HEAD
  mkdir -p changelog.d
  printf -- '- x\n' > changelog.d/2026-11-24-cg8.md
  GIT_DIR="$R/.git" GIT_INDEX_FILE="$R/.git/index" GIT_WORK_TREE="$R" run bash scripts/ci-local.sh --changed
  [[ "$output" == *"gen-changelog --check"* ]]
  [[ "$output" != *"FAIL gen-changelog"* ]]
  [[ "$output" == *"ci-local: --changed: 0 check(s) failed"* ]]
}

# gate_probe <mode> <command>: gate_run of the sourced lib on its own (a stand-in repo root and work dir), then the
# failed-gate count; the lib is sourced in a child so the counter survives `run`.
gate_probe() {
  run bash -c '
    set -u
    ROOT="$1" WORK="$2" RANGE="" CHANGED_FAILED=0; NAMES=(); BODIES=()
    . "$3/scripts/lib/ci-local-gates.sh"
    gate_run "probe gate" "$4" "$5"
    echo "failed=$CHANGED_FAILED"' _ "$R" "$BATS_TEST_TMPDIR/w" "$REPO" "$1" "$2"
}

@test "CG9: a hard gate whose command is missing or whose go test did not build says environment, not a size finding, and still counts (issue #973)" {
  # Mutation: CG9 -- printing every non-zero exit as `FAIL (exit N)` sends the reader to the file-size rule for a missing go binary
  mkdir -p "$BATS_TEST_TMPDIR/w"
  gate_probe hard 'no-such-command-for-ci-local'
  [ "$status" -eq 0 ]
  [[ "$output" == *"FAIL probe gate (environment: command not found, exit 127)"* ]]
  [[ "$output" == *"failed=1"* ]]
  gate_probe hard 'printf "FAIL\tpkg [build failed]\n"; exit 1'
  [[ "$output" == *"FAIL probe gate (environment: go test [build failed], exit 1)"* ]]
  [[ "$output" == *"failed=1"* ]]
  gate_probe hard 'printf "FAIL\tpkg [setup failed]\n"; exit 1'
  [[ "$output" == *"FAIL probe gate (environment: go test [setup failed], exit 1)"* ]]
  [[ "$output" == *"failed=1"* ]]
}

@test "CG10: a hard gate's ordinary failure and a soft gate keep their previous lines (issue #973)" {
  mkdir -p "$BATS_TEST_TMPDIR/w"
  gate_probe hard 'echo size finding; exit 1'
  [[ "$output" == *"FAIL probe gate (exit 1)"* ]]
  [[ "$output" != *"environment"* ]]
  [[ "$output" == *"failed=1"* ]]
  gate_probe soft 'no-such-command-for-ci-local'
  [[ "$output" == *"WARN probe gate (exit 127: the size was not measured)"* ]]
  [[ "$output" == *"failed=0"* ]]
}
