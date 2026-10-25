#!/usr/bin/env bats
# new-retro-truth.bats — the truth fixes of the Go new-retro over the bash new-retro.sh it replaced (15e792d1),
# run through the toolbelt wrapper. Each test names the golden truth-* case of go/internal/newretro/testdata
# that pins the same behaviour byte for byte; tests/retro-loop.bats keeps the original contract.

setup() {
  NR="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/new-retro.sh"
  TK="$BATS_TEST_TMPDIR/kit"; mkdir -p "$TK/retros"
  printf '<!-- build-state.v1 -->\n## kit\nretro_pending: false\n' > "$TK/BUILD-STATE.src.md"
  cp "$TK/BUILD-STATE.src.md" "$BATS_TEST_TMPDIR/state.before"
}
nr() { run env -C "$TK" KIT="$TK" "$NR" "$@"; }
# nothing_landed: no retro, no index, BUILD-STATE.src.md unchanged
nothing_landed() { [ -z "$(ls -A "$TK/retros")" ] && cmp -s "$BATS_TEST_TMPDIR/state.before" "$TK/BUILD-STATE.src.md"; }

@test "NRT1: a slug is checked whole: a space or a / after its 6th char is refused, exit 3, nothing lands" {
  # Mutation: NR-slug -- checking only the first 6 chars (the bash glob) writes "abcdef ghi" and exits 0
  # (golden truth-slug-space-after-six) and fails "abcdef/../../x" in the write, exit 1 (truth-slug-slash).
  local s
  for s in 'abcdef ghi' 'abcdef/../../x'; do
    nr kit "$s"
    [ "$status" -eq 3 ]
    [ "$output" = "new-retro: slug must be [A-Za-z0-9][A-Za-z0-9._-]+ and >= 6 chars: $s" ]
    nothing_landed
  done
}

@test "NRT2: --date must be a YYYY-MM-DD date and --deltas an integer from 0 to 99, else exit 3 and nothing lands" {
  # Mutation: NR-values -- accepting any value writes kit/x-slugabc.md outside retros/ for --date ../x
  # (golden truth-date-escapes-retros) and fails --deltas -1 late in the index generator, exit 1
  # (truth-deltas-negative); truth-date-not-a-date, truth-deltas-not-a-number and truth-deltas-over-cap pin the
  # other shapes (an overflowing count is pinned in Go, TestDeltasRefusedBeforeRender, never through a RED run).
  local opt val
  while read -r opt val want; do
    nr kit slugabc "$opt" "$val"
    [ "$status" -eq 3 ]
    [ "$output" = "new-retro: $opt must be $want: $val" ]
    nothing_landed
    [ ! -e "$TK/x-slugabc.md" ]
  done <<'ROWS'
--date ../x a YYYY-MM-DD date
--date 2026-13-01 a YYYY-MM-DD date
--deltas -1 an integer from 0 to 99
--deltas x an integer from 0 to 99
--deltas 100 an integer from 0 to 99
ROWS
}

@test "NRT3: an envelope that does not flip is no PASS: no section for the module exits 3, already true says so" {
  # Mutation: NR-nosection -- printing the envelope PASS row without a matching section exits 0 here
  # (goldens truth-no-envelope-section, truth-no-retro-pending-line, truth-flip-already-true).
  nr Typo slugabc
  [ "$status" -eq 3 ]
  [ "$output" = "new-retro: no build-state.v1 section for module Typo in $TK/BUILD-STATE.src.md" ]
  nothing_landed
  printf '<!-- build-state.v1 -->\n## kit\nretro_pending: true\n' > "$TK/BUILD-STATE.src.md"
  nr kit slugabc
  [ "$status" -eq 0 ]
  [[ "$output" == *"envelope  PASS  BUILD-STATE.src.md  retro_pending: already true" ]]
}
