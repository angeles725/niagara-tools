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
  [ "$(cd "$REPO" && for i in 1 2; do "$SHARD" "$i" 2; done | sort | uniq -u | wc -l)" -eq "$(find "$REPO/tests" -maxdepth 1 -name '*.bats' | wc -l)" ]
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

