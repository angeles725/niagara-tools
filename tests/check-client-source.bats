#!/usr/bin/env bats
# Tests for scripts/check-client-source.sh — is the module root the declared source of truth?
# [ev: retro client-source-of-truth Δ1] (git half of the preflight check; toolbelt scripts never
# call version control — kit-links L2 — so this lives in scripts/).
#
# Every repository is created under a temp dir with a throwaway identity; no network, no $HOME config.

setup() {
  TMPDIR_T="$(mktemp -d)"
  CS="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/scripts/check-client-source.sh"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid
  export GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid
}
teardown() { rm -rf "$TMPDIR_T"; }

# _repo <dir> — a repo on branch main with two commits; C1/C2 hold their shas, MODROOT the module root
_repo() {
  mkdir -p "$1/Group/Mod"
  git -C "$1" init -q -b main
  printf 'a\n' > "$1/Group/Mod/a.txt"; git -C "$1" add -A; git -C "$1" commit -q -m one
  C1=$(git -C "$1" rev-parse HEAD)
  printf 'b\n' > "$1/Group/Mod/b.txt"; git -C "$1" add -A; git -C "$1" commit -q -m two
  C2=$(git -C "$1" rev-parse HEAD)
  MODROOT="$1/Group/Mod"
}

@test "CS1: module root outside any git work tree -> WARN, exit 0" {
  mkdir -p "$TMPDIR_T/plain/Mod"
  run "$CS" --declared unknown "$TMPDIR_T/plain/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  source-of-truth"*"not inside a git work tree"* ]]
}

@test "CS2: detached HEAD -> FAIL, exit 1" {
  _repo "$TMPDIR_T/r"
  git -C "$TMPDIR_T/r" checkout -q --detach "$C1"
  run "$CS" --declared "r@main@$C2" "$MODROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  source-of-truth"*"detached"* ]]
}

@test "CS3: on the declared branch at the declared commit -> PASS, exit 0" {
  _repo "$TMPDIR_T/r"
  run "$CS" --declared "r@main@$C2" "$MODROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  source-of-truth"* ]]
  [[ "$output" != *"FAIL"* ]]
}

@test "CS4: HEAD behind the declared commit (stale checkout) -> FAIL, exit 1" {
  _repo "$TMPDIR_T/r"
  git -C "$TMPDIR_T/r" reset -q --hard "$C1"
  run "$CS" --declared "r@main@$C2" "$MODROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  source-of-truth"*"stale"* ]]
}

@test "CS5: declared commit unknown to this checkout -> FAIL, exit 1" {
  _repo "$TMPDIR_T/r"
  run "$CS" --declared "r@main@0123456789abcdef0123456789abcdef01234567" "$MODROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  source-of-truth"*"not in this checkout"* ]]
}

@test "CS6: checkout on a different branch than declared -> FAIL, exit 1" {
  _repo "$TMPDIR_T/r"
  git -C "$TMPDIR_T/r" checkout -q -b feature
  run "$CS" --declared "r@main@$C2" "$MODROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  source-of-truth"*"feature"*"main"* ]]
}

@test "CS7: the module root lives in a different checkout than the declared path -> FAIL, exit 1" {
  _repo "$TMPDIR_T/r"
  run "$CS" --declared "Cliente/other-tree@main@$C2" "$MODROOT"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  source-of-truth"*"other-tree"* ]]
}

@test "CS8: HEAD ahead of the declared commit (new local work) -> PASS with an update note" {
  _repo "$TMPDIR_T/r"
  run "$CS" --declared "r@main@$C1" "$MODROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  source-of-truth"*"ahead"* ]]
}

@test "CS9: source_of_truth unknown -> WARN (honest unknown is not a failure), exit 0" {
  _repo "$TMPDIR_T/r"
  run "$CS" --declared unknown "$MODROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  source-of-truth"*"unknown"* ]]
}

@test "CS10: --build-state <file> --module <M> reads the envelope's source_of_truth field" {
  _repo "$TMPDIR_T/r"
  cat > "$TMPDIR_T/BUILD-STATE.md" <<EOF
<!-- build-state.v1 -->
module: Other
source_of_truth: elsewhere@main@$C1
<!-- /build-state.v1 -->
<!-- build-state.v1 -->
module: Mod
source_of_truth: r@main@$C2   # DECLARED
<!-- /build-state.v1 -->
EOF
  run "$CS" --build-state "$TMPDIR_T/BUILD-STATE.md" --module Mod "$MODROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  source-of-truth"* ]]
}

@test "CS11: a module with no source_of_truth field in BUILD-STATE -> WARN undeclared, exit 0" {
  _repo "$TMPDIR_T/r"
  printf '<!-- build-state.v1 -->\nmodule: Mod\n<!-- /build-state.v1 -->\n' > "$TMPDIR_T/BUILD-STATE.md"
  run "$CS" --build-state "$TMPDIR_T/BUILD-STATE.md" --module Mod "$MODROOT"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  source-of-truth"*"undeclared"* ]]
}

@test "CS12: HEAD behind its local upstream tracking ref -> WARN naming the count (no network)" {
  _repo "$TMPDIR_T/up"
  git clone -q "$TMPDIR_T/up" "$TMPDIR_T/down"
  printf 'c\n' > "$TMPDIR_T/up/Group/Mod/c.txt"; git -C "$TMPDIR_T/up" add -A; git -C "$TMPDIR_T/up" commit -q -m three
  git -C "$TMPDIR_T/down" fetch -q
  run "$CS" --declared "down@main@$C2" "$TMPDIR_T/down/Group/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  source-of-truth"*"behind"*"1"* ]]
}

@test "CS-usage: no module root -> exit 2; a missing directory -> exit 3" {
  run "$CS"
  [ "$status" -eq 2 ]
  run "$CS" --declared unknown "$TMPDIR_T/nope"
  [ "$status" -eq 3 ]
}

@test "CS-malformed: a declared value that is not <path>@<branch>@<commit> -> exit 2" {
  _repo "$TMPDIR_T/r"
  run "$CS" --declared "r-main" "$MODROOT"
  [ "$status" -eq 2 ]
}
