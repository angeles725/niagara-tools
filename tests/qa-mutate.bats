#!/usr/bin/env bats
# Self-test of qa/mutate.sh (campaign-9 verify terrain): the mutation harness must itself bite.
# MU1 refuses a real checkout (HEAD on a branch) · MU2 OBSERVES a known flip on a throwaway detached
# worktree and restores it · MU3 reports ANCHOR-MISSING (exit 1) instead of mutating blindly
# · MU4 NO-OP-MUTATION when the sed changes nothing.
setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  MUT="$REPO/qa/mutate.sh"
  WT="$BATS_TEST_TMPDIR/wt"
  git -C "$REPO" worktree add -q --detach "$WT" HEAD
  T="$BATS_TEST_TMPDIR/t.tsv"
}
teardown() { git -C "$REPO" worktree remove --force "$WT" 2>/dev/null || true; }
row() { printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$@" >> "$T"; }
# known flip: short-circuit lint-write-path.sh right after its `set -` line -> WP2 (uncovered -> exit 1)
# and WP-usage (no arg -> exit 3) both stop failing the way the pins expect.
known_flip_row() {
  row X1 PRX build-n4-module-kit/toolbelt/lint-write-path.sh bats 'bats tests/lint-write-path.bats' \
      'WP2,WP-usage' '^set -' sed 's/^(set -[a-zA-Z]+)$/\1\nexit 0/' 'selftest'
}

@test "MU1: refuses a worktree whose HEAD is on a branch (exit 3, nothing touched)" {
  known_flip_row
  run "$MUT" --worktree "$REPO" --table "$T" --id X1
  [ "$status" -eq 3 ]
  [[ "$output" == *"refusing"* ]]
}

@test "MU2: OBSERVED on a known flip, verbatim log kept, tree restored" {
  known_flip_row
  run "$MUT" --worktree "$WT" --table "$T" --id X1 --out "$BATS_TEST_TMPDIR/out"
  echo "$output"
  [ "$status" -eq 0 ]
  [[ "$output" == *"OBSERVED"*"X1"*"flips 2/2"* ]]
  [ -f "$BATS_TEST_TMPDIR/out/X1.log" ] && grep -qE '^not ok .*WP2' "$BATS_TEST_TMPDIR/out/X1.log"
  [ -z "$(git -C "$WT" status --porcelain)" ]
}

@test "MU3: ANCHOR-MISSING -> exit 1, file untouched" {
  row X2 PRX build-n4-module-kit/toolbelt/lint-write-path.sh bats 'bats tests/lint-write-path.bats' \
      WP2 'THIS_ANCHOR_DOES_NOT_EXIST' sed 's/a/b/' 'selftest'
  run "$MUT" --worktree "$WT" --table "$T" --id X2
  [ "$status" -eq 1 ]
  [[ "$output" == *"ANCHOR-MISSING"*"X2"* ]]
  [ -z "$(git -C "$WT" status --porcelain)" ]
}

@test "MU4: NO-OP-MUTATION when the sed changes nothing -> exit 1" {
  row X3 PRX build-n4-module-kit/toolbelt/lint-write-path.sh bats 'bats tests/lint-write-path.bats' \
      WP2 '^set -' sed 's/ZZZ_NOT_PRESENT/y/' 'selftest'
  run "$MUT" --worktree "$WT" --table "$T" --id X3
  [ "$status" -eq 1 ]
  [[ "$output" == *"NO-OP-MUTATION"*"X3"* ]]
}

@test "MU5: --list prints the C9 table without a worktree" {
  run "$MUT" --table "$REPO/qa/c9-mutations.tsv" --list --pr PR6b
  [ "$status" -eq 0 ]
  [[ "$output" == *"M6b1"* ]] && [[ "$output" == *"M6b3"* ]] && [[ "$output" != *"M7a"* ]]
}
