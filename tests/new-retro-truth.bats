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
