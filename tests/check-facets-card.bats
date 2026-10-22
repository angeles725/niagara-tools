#!/usr/bin/env bats
# shellcheck disable=SC2016  # the backticks in the fixture text are literal Markdown, not command substitution
# check-facets-card.sh — the BFacets key card in types/logic-authoring.md must list exactly the BFacets key
# constants, in source order (register C06, rac Δ8). The card was hand-copied once and drifted: it named
# makeDouble and 17 keys that are not constants. The pinned list is tests/fixtures/facets-card/BFacets-4.15.6.20.keys
# (generated from the 4.15.6.20 source; the regenerate command is in its header); a .java source works too.
# [ev: retro devguide415-runtime-api-corrections Δ8]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  CHK="$KIT/toolbelt/check-facets-card.sh"
  KEYS="$BATS_TEST_DIRNAME/fixtures/facets-card/BFacets-4.15.6.20.keys"
  CARD="$KIT/types/logic-authoring.md"
}

# card_with <rows...>: a minimal doc whose card section holds the given table rows
card_with() {
  printf '%s\n' '# doc' '## BFacets key reference card' '' '| Constant | Key | Value | Purpose |' '|---|---|---|---|' "$@" \
    '' '**Not `BFacets` keys**: `nullText`' '' '## Next section' '| `NOT_IN_CARD` | `"later"` | x | y |' > "$BATS_TEST_TMPDIR/card.md"
}

@test "FC1: the live kit card matches the pinned 4.15.6.20 BFacets constants, in order" {
  run "$CHK" "$CARD" "$KEYS"
  [ "$status" -eq 0 ]
  [[ "$output" == *"34 keys match"* ]]
}

@test "FC-missing: a card that drops a constant fails naming it" {
  card_with '| `MIN` | `"min"` | a | b |'
  printf '%s\n' 'MIN=min' 'MAX=max' > "$BATS_TEST_TMPDIR/k.keys"
  run "$CHK" "$BATS_TEST_TMPDIR/card.md" "$BATS_TEST_TMPDIR/k.keys"
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISSING MAX=max"* ]]
}

@test "FC-extra: a card row that is not a constant fails naming it; rows after the section are ignored" {
  card_with '| `MIN` | `"min"` | a | b |' '| `NULL_TEXT` | `"nullText"` | a | b |'
  printf '%s\n' 'MIN=min' > "$BATS_TEST_TMPDIR/k.keys"
  run "$CHK" "$BATS_TEST_TMPDIR/card.md" "$BATS_TEST_TMPDIR/k.keys"
  [ "$status" -eq 1 ]
  [[ "$output" == *"EXTRA NULL_TEXT=nullText"* ]]
  [[ "$output" != *"NOT_IN_CARD"* ]]
}

@test "FC-order: the same keys in another order fail (the card follows declaration order)" {
  card_with '| `MAX` | `"max"` | a | b |' '| `MIN` | `"min"` | a | b |'
  printf '%s\n' 'MIN=min' 'MAX=max' > "$BATS_TEST_TMPDIR/k.keys"
  run "$CHK" "$BATS_TEST_TMPDIR/card.md" "$BATS_TEST_TMPDIR/k.keys"
  [ "$status" -eq 1 ]
  [[ "$output" == *"ORDER"* ]]
}

@test "FC-java: a BFacets.java source is read for its String key constants (private arrays ignored)" {
  card_with '| `MIN` | `"min"` | a | b |' '| `ORD_RELATIVIZE` | `"ordRelativize"` | a | b |'
  printf '%s\n' 'public final class BFacets {' '  public static final String MIN = "min";' \
    '  public static final String ORD_RELATIVIZE= "ordRelativize";' \
    '  private static final String[] NUMERIC_KEYS = { MIN };' '}' > "$BATS_TEST_TMPDIR/BFacets.java"
  run "$CHK" "$BATS_TEST_TMPDIR/card.md" "$BATS_TEST_TMPDIR/BFacets.java"
  [ "$status" -eq 0 ]
}

@test "FC-empty: no card rows or no source keys is an env error (exit 3), never a clean match" {
  card_with
  run "$CHK" "$BATS_TEST_TMPDIR/card.md" "$KEYS"
  [ "$status" -eq 3 ]
  run "$CHK" "$CARD" /dev/null
  [ "$status" -eq 3 ]
  run "$CHK" "$CARD"
  [ "$status" -eq 3 ]
}
