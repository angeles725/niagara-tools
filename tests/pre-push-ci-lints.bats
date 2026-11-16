#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# The CI-lints step of .githooks/pre-push and the retro gate's refusal text, end to end: the real hook in a
# throwaway repository whose scripts/ci-local.sh is a copy of this one (go/ and build-n4-module-kit/ are
# links to this checkout). The fixture workflow shellchecks scripts/*.sh only and the cases change no .sh,
# so no case downloads the pinned shellcheck. The host budget reads fixtures (CI_LOCAL_* below).
#   C3: #684 failed CI on a bats line the kit's lint refuses; the push is now refused locally.
#   C8: #675 failed CI on a here-document line bats 1.10 reads as a duplicate case; refused locally too.
#   C6: the retro refusal names the directory it read and the two ways out.
# (retro 2026-11-15 coordination-stretch C3) (retro 2026-11-15 coordination-stretch C6) (retro 2026-11-15 coordination-stretch C8)

setup_file() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime
  . "$REPO/go/go-tool.sh"
  local t; for t in pre-push ci-steps host-budget n4lint; do go_tool_bin "$REPO/go" "$t" >/dev/null; done
}

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  HOOK="$REPO/.githooks/pre-push"
  R="$BATS_TEST_TMPDIR/r"
  mkdir -p "$R/scripts/lib" "$R/.github/workflows" "$R/tests" "$BATS_TEST_TMPDIR/cg/init.scope"
  cp "$REPO/scripts/ci-local.sh" "$R/scripts/"
  cp "$REPO"/scripts/lib/*.sh "$R/scripts/lib/"
  ln -s "$REPO/go" "$R/go"
  ln -s "$REPO/build-n4-module-kit" "$R/build-n4-module-kit"
  printf 'name: CI\non: pull_request\njobs:\n  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - name: shellcheck\n        run: shellcheck scripts/*.sh\n' \
    > "$R/.github/workflows/ci.yml"
  printf '16\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.current"
  printf '48129\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.max"
  mkdir -p "$BATS_TEST_TMPDIR/gocache"
  export CI_LOCAL_CGROUP_ROOT="$BATS_TEST_TMPDIR/cg"
  export CI_LOCAL_HOST_BUDGET_ARGS="--proc-root $REPO/tests/fixtures/host-budget/proc --gocache $BATS_TEST_TMPDIR/gocache --disk-min-free 1"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 NIAGARA_CLIENT_DENYLIST="$BATS_TEST_TMPDIR/no-denylist"
  export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid
  git -C "$R" init -q -b main
  git -C "$R" add -A
  git -C "$R" commit -q -m base
  git -C "$R" update-ref refs/remotes/origin/main HEAD
  BASE="$(git -C "$R" rev-parse HEAD)"
}

# push <commit message> — commit the work tree and run the hook on a push of it over the base.
push() {
  git -C "$R" add -A
  git -C "$R" commit -q -m "$1"
  local sha; sha="$(git -C "$R" rev-parse HEAD)"
  cd "$R" || return 1
  run bash "$HOOK" origin https://example.invalid/r.git <<< "refs/heads/x $sha refs/heads/x $BASE"
}

@test "PCL1: a bats case with a bare mid-test negation (the #684 line 783 form) is refused before it leaves" {
  # Mutation: PCL1 -- dropping the ciLints call from go/internal/prepush gate.go lets this push through.
  printf '%s\n' '@test "a" {' '  ! cmp -s a b' '  true' '}' > "$R/tests/x.bats"
  push "test: x"
  [ "$status" -eq 1 ]
  [[ "$output" == *"bats-negation"*"tests/x.bats:2"* ]]
  [[ "$output" == *"pre-push: the changed files fail the CI lints (scripts/ci-local.sh --changed --range $BASE.."* ]]
  # the #684 fix of the same line passes
  printf '%s\n' '@test "a" {' '  ! cmp -s a b || return 1' '  true' '}' > "$R/tests/x.bats"
  push "test: x fixed"
  [ "$status" -eq 0 ]
}

@test "PCL2: a here-document line that starts with the test keyword (the #675 form) is refused" {
  # Mutation: PCL2 -- dropping the lint-bats-heredoc check from lib/ci-local-changed.sh lets this push through.
  printf '%s\n' '@test "a" {' "  cat > f <<'SH'" '@test "inner" { :; }' 'SH' '  true' '}' > "$R/tests/y.bats"
  push "test: y"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  bats-heredoc-test  tests/y.bats:3"* ]]
}

@test "PCL3: the retro refusal names the directory it read and the two ways out" {
  # Mutation: PCL3 -- the old text ("See BUILD-LOOP.md §7." alone) does not say a repository-level retros/ is not read.
  mkdir -p "$R/retros"
  printf 'x\n' > "$R/scripts/notes.txt"
  printf '# a retro outside the kit\n' > "$R/retros/2026-11-15-elsewhere.md"
  push "chore: notes

Changelog: none (fixture)"
  [ "$status" -eq 1 ]
  [[ "$output" == *"retros are read only from build-n4-module-kit/retros/ and fragments only from build-n4-module-kit/ledger.d/"* ]]
  [[ "$output" == *"(1) a pending retro under build-n4-module-kit/retros/ plus a build-n4-module-kit/ledger.d/ fragment"* ]]
  [[ "$output" == *"(2) a 'Retro: none (trivial: <reason>)' trailer in a commit of the range"* ]]
}
