#!/usr/bin/env bats
# BS: scripts/bats-shard.sh splits tests/*.bats into N CI shards. Files in the weight table are placed
# greedily longest-first onto the lightest shard; unlisted files go round-robin by sorted name. Every
# file must land in exactly one shard (a new file can never be silently skipped), and the split must be
# deterministic. These tests drive the script against a scratch tests dir plus a scratch weight table.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  SHARD="$REPO/scripts/bats-shard.sh"
  T="$BATS_TEST_TMPDIR/tests"
  W="$BATS_TEST_TMPDIR/weights"
  mkdir -p "$T"
  for f in a b c d e f g; do : > "$T/$f.bats"; done
  : > "$T/notes.txt"
  printf '%s\n' '# seconds file' '10 a.bats' '6 b.bats' '5 c.bats' '1 d.bats' '9 gone.bats' > "$W"
}

# all_shards COUNT — print every shard's list, one file per line.
all_shards() {
  local i
  for ((i = 1; i <= $1; i++)); do "$SHARD" --dir "$T" --weights "$W" "$i" "$1" || return; done
}

@test "BS1: the shards partition the .bats files exactly (2 and 3 shards)" {
  # Mutation: BS1 -- skipping the round-robin pass drops every unlisted file from all shards
  local n
  for n in 2 3; do
    run all_shards "$n"
    [ "$status" -eq 0 ]
    [ "$(printf '%s\n' "$output" | sort)" = "$(printf '%s\n' "$T"/*.bats | sort)" ]
    [ -z "$(printf '%s\n' "$output" | sort | uniq -d)" ]
  done
}

@test "BS2: weighted files go longest-first onto the lightest shard" {
  # a=10 -> s1; b=6 -> s2; c=5 -> s2 (6<10); d=1 -> s1 (10<11); unlisted e,f,g round-robin s1,s2,s1.
  run "$SHARD" --dir "$T" --weights "$W" 1 2
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\n' "$T/a.bats" "$T/d.bats" "$T/e.bats" "$T/g.bats")" ]
  run "$SHARD" --dir "$T" --weights "$W" 2 2
  [ "$output" = "$(printf '%s\n' "$T/b.bats" "$T/c.bats" "$T/f.bats")" ]
}

@test "BS3: the split is deterministic (repeat runs and file creation order do not change it)" {
  local first
  first="$(all_shards 2)"
  [ "$(all_shards 2)" = "$first" ]
  rm "$T"/*.bats
  for f in g f e d c b a; do : > "$T/$f.bats"; done
  [ "$(all_shards 2)" = "$first" ]
}

@test "BS4: a file missing from the weight table still lands in exactly one shard" {
  : > "$T/zz-new.bats"
  run all_shards 2
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c '/zz-new\.bats$')" -eq 1 ]
}

@test "BS5: --check passes for the real tests dir and the built-in weight table" {
  run "$SHARD" --check 2
  [ "$status" -eq 0 ]
  [[ "$output" == PASS* ]]
  # the union of the two shards is exactly tests/*.bats, each file once
  [ "$(cd "$REPO" && for i in 1 2; do "$SHARD" "$i" 2; done | sort)" = "$(cd "$REPO" && printf '%s\n' tests/*.bats | sort)" ]
}

@test "BS6: a bad shard index or count is a usage error (exit 2)" {
  for args in "0 2" "3 2" "1 0" "x 2" "1"; do
    # shellcheck disable=SC2086
    run "$SHARD" --dir "$T" --weights "$W" $args
    [ "$status" -eq 2 ]
  done
}

@test "BS7: a tests dir with no .bats files is an environment error (exit 3)" {
  rm "$T"/*.bats
  run "$SHARD" --dir "$T" --weights "$W" 1 2
  [ "$status" -eq 3 ]
}

@test "BS8: a shard lists its weighted files heaviest first, so bats -j starts the longest at once" {
  # Mutation: BS8 -- printing rows in name order starts the 50 s file last in shard 1
  printf '%s\n' '50 g.bats' '10 a.bats' '6 b.bats' > "$W"
  run "$SHARD" --dir "$T" --weights "$W" 1 2
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "$T/g.bats" ]
  run "$SHARD" --dir "$T" --weights "$W" 2 2
  [ "${lines[0]}" = "$T/a.bats" ]
  [ "${lines[1]}" = "$T/b.bats" ]
}

@test "BS9: ci.yml's bats job runs one job per shard through bats-shard.sh, and a failing shard fails the workflow" {
  # Mutation: BS9 -- a matrix of three shards under a `--check 2` default splits the suite two ways in CI and three ways in a local replay
  local ci="$REPO/.github/workflows/ci.yml" job n
  job="$(awk '/^  bats:$/ { on = 1; next } on && /^  [a-z0-9-]+:$/ { exit } on' "$ci")"
  n="$(sed -nE 's/^ +shard: \[(.*)\]$/\1/p' <<<"$job")"
  [ "$n" = "1, 2" ]
  # shellcheck disable=SC2016  # the patterns are literal ci.yml text, never expanded
  local pins=('strategy.job-total' '${SHARDS:-2}' 'scripts/bats-shard.sh --check' 'scripts/bats-shard.sh "$SHARD" "$SHARDS"')
  local pin
  for pin in "${pins[@]}"; do [[ "$job" == *"$pin"* ]] || { echo "ci.yml bats job lacks: $pin"; return 1; }; done
  # fail-fast is off so both shards report, but neither step nor job may swallow a failure
  [[ "$job" == *'fail-fast: false'* ]]
  [[ "$job" != *continue-on-error* ]]
  # every shard job sets up Java 8 (run-pure-test.sh) and keeps the global skip gate
  [[ "$job" == *'java-version: '*8* ]]
  [[ "$job" == *'scripts/check-bats-skips.sh'* ]]
}

@test "BS10: --check exits 1 and names a file the shard listing loses (a name with a space), nothing else" {
  # Mutation: BS10 -- dropping the exit 1 of the failure branch lets a file in no shard pass the CI check
  : > "$T/has space.bats"
  run "$SHARD" --dir "$T" --weights "$W" --check 2
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL bats-shard in no shard: has space.bats" ]
}

@test "BS11: --check exits 1 and names an empty shard (more shards than files, issue #981)" {
  # Mutation: BS11 -- not checking each shard for files lets a CI job start with nothing to run
  rm "$T"/*.bats
  : > "$T/only.bats"
  run "$SHARD" --dir "$T" --weights "$W" --check 2
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL bats-shard empty shard: 2" ]
  run "$SHARD" --dir "$T" --weights "$W" --check 1
  [ "$status" -eq 0 ]
}
