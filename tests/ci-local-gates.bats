#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# ci-local-gates.bats — `scripts/ci-local.sh --changed` runs the tree gates of go/internal/cisteps/gates.go for the
# files changed (CG1-CG6, issues #983 item 5 and #973 item 3). The cardwidth gate (the 120-character limit on kit docs)
# ran only in CI, so a METHODOLOGY.md line over 120 broke PR #978 after the push; the summary did not count the gates.
# The fixture is a throwaway repository: this checkout's HEAD without go/, a go/ of links to this checkout's (its
# README.md copied: cardwidth walks the fixture's tree and counts it) and this work tree's ci-local files. Its workflow
# has one shellcheck step and one stand-in for ci.yml's gen-changelog step, so nothing is downloaded.

setup_file() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime
  . "$REPO/go/go-tool.sh"
  local t; for t in ci-steps host-budget kit-state; do go_tool_bin "$REPO/go" "$t" >/dev/null; done
}

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  R="$BATS_TEST_TMPDIR/r"
  mkdir -p "$R/go" "$BATS_TEST_TMPDIR/cg/init.scope" "$BATS_TEST_TMPDIR/gocache"
  git -C "$REPO" archive HEAD -- ':(exclude)go' | tar -x -C "$R"
  ln -s "$REPO"/go/* "$R/go/"
  rm "$R/go/README.md"
  cp "$REPO/go/README.md" "$R/go/"
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
