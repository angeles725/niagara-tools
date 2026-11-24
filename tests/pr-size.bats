#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# pr-size.sh — the PR size gate (decision L14): additions + deletions of base...head, excluding generated, lock and
# vendored files, fail over PR_SIZE_LIMIT (default 400). The fixture is a throwaway git repository.
# pr-size.yml is pinned structurally (PS12-PS13): the trigger types, the label that waives the gate, and SHA-pinned actions.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  GATE="$REPO/scripts/pr-size.sh"
  WF="$REPO/.github/workflows/pr-size.yml"
  R="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$R"
  git -C "$R" init -q -b main
  git -C "$R" config user.email t@example.invalid
  git -C "$R" config user.name tester
  git -C "$R" config commit.gpgsign false
  printf 'base\n' > "$R/README.md"
  git -C "$R" add -A && git -C "$R" commit -qm "chore: base"
  BASE=$(git -C "$R" rev-parse HEAD)
  unset PR_SIZE_LIMIT
}

lines() { # lines <n> [prefix]: n distinct lines
  local i
  for ((i = 1; i <= $1; i++)); do printf '%s%d\n' "${2:-l}" "$i"; done
}

head_commit() { # head_commit: commit every change in the repo, set HEAD_SHA
  git -C "$R" add -A && git -C "$R" commit -qm "feat: change"
  HEAD_SHA=$(git -C "$R" rev-parse HEAD)
}

set_base() { # set_base <rev>: the next gate run diffs from <rev>
  BASE=$(git -C "$R" rev-parse "$1")
}

gate() { # gate [args...]: run the gate inside the repo, default BASE HEAD_SHA
  cd "$R" || return 1
  if [ "$#" -eq 0 ]; then run bash "$GATE" "$BASE" "$HEAD_SHA"; else run bash "$GATE" "$@"; fi
}

@test "PS1: a PR under the limit passes and prints per-file counts and the total" {
  lines 30 > "$R/a.txt"; lines 20 > "$R/b.txt"
  head_commit
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"30"*"a.txt"* ]]
  [[ "$output" == *"20"*"b.txt"* ]]
  [[ "$output" == *"total 50 changed lines (+50 -0)"* ]]
  [[ "$output" == *"limit 400"* ]]
}

@test "PS2: exactly 400 lines pass, 401 fail with exit 1 naming the total and the label" {
  # Mutation: PS2-off-by-one -- failing at >= 400 instead of > 400 rejects a PR exactly at the budget
  lines 400 > "$R/a.txt"
  head_commit
  gate
  [ "$status" -eq 0 ]
  printf 'extra\n' >> "$R/a.txt"
  head_commit
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"total 401 changed lines (+401 -0)"* ]]
  [[ "$output" == *"exceeds the 400-line limit"* ]]
  [[ "$output" == *"size:exception"* ]]
}

@test "PS3: deletions count as well as additions" {
  # Mutation: PS3-additions-only -- counting only additions lets a 500-line deletion through
  lines 500 > "$R/old.txt"
  head_commit
  set_base HEAD
  git -C "$R" rm -q old.txt
  lines 5 > "$R/new.txt"
  head_commit
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"total 505 changed lines (+5 -500)"* ]]
}

@test "PS4: go.sum, *.lock, package-lock.json, testdata/** and vendor/** are listed as excluded and not counted" {
  # Mutation: PS4-no-exclude -- counting generated, lock and vendored files makes a dependency bump fail the gate
  mkdir -p "$R/go/internal/x/testdata" "$R/vendor/dep" "$R/tool"
  lines 1000 > "$R/go/go.sum"
  lines 1000 > "$R/Cargo.lock"
  lines 1000 > "$R/tool/package-lock.json"
  lines 1000 > "$R/go/internal/x/testdata/golden.txt"
  lines 1000 > "$R/vendor/dep/lib.go"
  lines 12 > "$R/real.txt"
  head_commit
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"total 12 changed lines (+12 -0)"* ]]
  [[ "$output" == *"excluded"*"go/go.sum"* ]]
  [[ "$output" == *"excluded"*"go/internal/x/testdata/golden.txt"* ]]
  [[ "$output" == *"5 file(s) excluded"* ]]
}

@test "PS5: a file merely named like an excluded one (testdata.go, a go.sum.md) is still counted" {
  lines 250 > "$R/testdata.go"
  lines 250 > "$R/go.sum.md"
  head_commit
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"total 500 changed lines"* ]]
}

@test "PS6: a changelog.d fragment counts toward the limit" {
  mkdir -p "$R/changelog.d"
  lines 395 > "$R/a.txt"
  lines 10 > "$R/changelog.d/2026-11-24-x.md"
  head_commit
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"total 405 changed lines"* ]]
}

@test "PS7: PR_SIZE_LIMIT overrides the default in both directions" {
  # Mutation: PS7-hardcoded-limit -- ignoring PR_SIZE_LIMIT makes the limit untunable and untestable
  lines 20 > "$R/a.txt"
  head_commit
  PR_SIZE_LIMIT=10 gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"exceeds the 10-line limit"* ]]
  PR_SIZE_LIMIT=50 gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"limit 50"* ]]
  PR_SIZE_LIMIT=1000 gate
  [ "$status" -eq 0 ]
}

@test "PS8: a limit that is not a positive integer is exit 3" {
  lines 5 > "$R/a.txt"
  head_commit
  PR_SIZE_LIMIT=abc gate
  [ "$status" -eq 3 ]
  [[ "$output" == *"PR_SIZE_LIMIT"* ]]
  PR_SIZE_LIMIT=0 gate
  [ "$status" -eq 3 ]
  PR_SIZE_LIMIT='' gate
  [ "$status" -eq 0 ]
}

@test "PS9: missing arguments, an unknown ref and a non-repository are exit 3" {
  lines 5 > "$R/a.txt"
  head_commit
  gate "$BASE"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: pr-size.sh <base> <head>"* ]]
  gate "$BASE" no-such-ref
  [ "$status" -eq 3 ]
  [[ "$output" == *"no-such-ref"* ]]
  cd "$BATS_TEST_TMPDIR" && run env GIT_CEILING_DIRECTORIES="$BATS_TEST_TMPDIR" bash "$GATE" "$BASE" "$HEAD_SHA"
  [ "$status" -eq 3 ]
}

@test "PS10: the diff is merge-base relative, so commits that landed on the base after the branch point are not counted" {
  git -C "$R" checkout -q -b pr
  lines 10 > "$R/a.txt"
  head_commit
  git -C "$R" checkout -q main
  lines 600 > "$R/other.txt"
  git -C "$R" add -A && git -C "$R" commit -qm "feat: lands on main meanwhile"
  set_base main
  gate "$BASE" "$HEAD_SHA"
  [ "$status" -eq 0 ]
  [[ "$output" == *"total 10 changed lines (+10 -0)"* ]]
}

@test "PS11: a pure rename counts zero lines, a rename with an edit counts the edit; binary files count zero" {
  lines 600 > "$R/big.txt"
  head_commit
  set_base HEAD
  mkdir -p "$R/moved"
  git -C "$R" mv big.txt moved/big.txt
  printf 'x\n' >> "$R/moved/big.txt"
  printf '\000\001\002' > "$R/blob.bin"
  head_commit
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"total 1 changed lines (+1 -0)"* ]]
  [[ "$output" == *"moved/big.txt"* ]]
  [[ "$output" == *"blob.bin"*"binary"* ]]
}

@test "PS12: pr-size.yml runs on the five pull_request types, separate from ci.yml, and waives only for size:exception" {
  [ -f "$WF" ]
  grep -q '^  pull_request:$' "$WF"
  grep -q 'types: \[opened, synchronize, reopened, labeled, unlabeled\]' "$WF"
  grep -q "contains(github.event.pull_request.labels.\*.name, 'size:exception')" "$WF"
  grep -q 'scripts/pr-size.sh' "$WF"
  grep -q 'fetch-depth: 0' "$WF"
  run grep -c 'pr-size' "$REPO/.github/workflows/ci.yml"
  [ "$output" = "0" ]
}

@test "PS13: every action pr-size.yml uses is pinned by a 40-hex commit SHA" {
  # Mutation: PS13-tag-pin -- a mutable tag lets an action's owner change what runs with the PR's token
  run grep -E '^\s*-?\s*uses:' "$WF"
  [ "$status" -eq 0 ]
  while IFS= read -r l; do
    [[ "$l" =~ uses:\ [A-Za-z0-9._/-]+@[0-9a-f]{40}\ *(#.*)?$ ]] || { echo "unpinned: $l"; return 1; }
  done <<<"$output"
}
