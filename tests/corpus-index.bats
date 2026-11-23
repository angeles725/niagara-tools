#!/usr/bin/env bats
# The corpus index holds an entry for every corpus block a kit guide cites in an `[ev: corpus ...]` tag, and its
# title states the range it reaches. [ev: retro doc-coverage-audit Δ20]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  CARDS=("$KIT"/types/*.md "$KIT/BUILD-LOOP.md" "$KIT/METHODOLOGY.md" "$KIT/skill/SKILL.md" "$KIT/ORCHESTRATION.md" "$KIT/build-verify.md" "$KIT/SOURCES.md" "$KIT/README.md")
  cat "${CARDS[@]}" | tr '\n' ' ' | grep -o '\[ev: corpus [^]]*\]' | grep -o '\bB[0-9]\{1,4\}\b' | sort -u > "$BATS_TEST_TMPDIR/cited"
  grep -o '\bB[0-9]\{1,4\}\b' "$KIT/corpus-index.md" | sort -u > "$BATS_TEST_TMPDIR/indexed"
}

@test "CI-ALL: every block a guide cites in an [ev: corpus] tag has an entry in corpus-index.md" {
  run comm -23 "$BATS_TEST_TMPDIR/cited" "$BATS_TEST_TMPDIR/indexed"
  [ "$status" -eq 0 ]
  [ -z "$output" ] || { echo "cited, not in the index: $(echo "$output" | tr '\n' ' ')"; false; }
}

@test "CI-RANGE: the title names the highest block the guides cite" {
  top=$(sed 's/^B//' "$BATS_TEST_TMPDIR/cited" | sort -n | tail -n 1)
  run head -n 1 "$KIT/corpus-index.md"
  [[ "$output" == *"–B${top})"* ]] || { echo "title: $output; highest cited block: B$top"; false; }
}

@test "CI-SENTINEL: the cited set is not empty (the other two tests would pass on nothing)" {
  [ "$(wc -l < "$BATS_TEST_TMPDIR/cited")" -gt 100 ]
}
