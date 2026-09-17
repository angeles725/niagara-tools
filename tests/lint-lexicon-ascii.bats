#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-lexicon-ascii.sh
# Regression: Niagara reads lexicon files as Latin-1; UTF-8 multi-byte accents
# arrive on-station as mojibake. All lexicon content must be ASCII-only.
# [ev: retro 2026-09-17-umbrelladashboard-module-creation]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LA="$KIT/toolbelt/lint-lexicon-ascii.sh"
  # Create a minimal module tree with a lexicon
  mkdir -p "$TMPDIR_T/Mod/Mod-rt"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "LA1: a lexicon with only ASCII content exits 0 (clean)" {
  printf 'foo=bar\nbaz=qux 123\n' > "$TMPDIR_T/Mod/Mod-rt/module.lexicon"
  run "$LA" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "LA2: a lexicon with a UTF-8 multi-byte accented character exits 1 (non-ASCII bytes detected)" {
  # "Presión" in UTF-8 encodes "ó" as 0xc3 0xb3
  printf 'pressure=Presi\xc3\xb3n\n' > "$TMPDIR_T/Mod/Mod-rt/module.lexicon"
  run "$LA" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"non-ASCII"* ]]
  # Named mutation: remove the non-ASCII byte scan -> LA2's FAIL vanishes (exit 0).
}

@test "LA3: FAIL row reports the file path relative to module root and the line number" {
  printf 'ok=yes\nbad=caf\xc3\xa9\n' > "$TMPDIR_T/Mod/Mod-rt/module.lexicon"
  run "$LA" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  # Must contain the relative path and line number 2
  [[ "$output" == *"Mod-rt/module.lexicon:2"* ]]
}

@test "LA4: multiple lexicon files under separate profiles are all scanned" {
  mkdir -p "$TMPDIR_T/Mod/Mod-ux"
  printf 'a=ascii\n' > "$TMPDIR_T/Mod/Mod-rt/module.lexicon"
  # ux lexicon has non-ASCII
  printf 'b=\xc3\xa9\n' > "$TMPDIR_T/Mod/Mod-ux/module.lexicon"
  run "$LA" "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Mod-ux/module.lexicon"* ]]
}

@test "LA5: module root with no *.lexicon files exits 0 (no files to check — not an error)" {
  run "$LA" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
}

@test "LA6: no argument exits 3 with usage message" {
  run "$LA"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage"* ]]
}

@test "LA7: non-existent module root exits 3" {
  run "$LA" "/nonexistent/path/that/does/not/exist"
  [ "$status" -eq 3 ]
}

@test "LA8: a dot-directory (e.g. .deploy-baseline) is pruned and its lexicons not scanned (D9b)" {
  # Baseline snapshots should not be double-counted
  mkdir -p "$TMPDIR_T/Mod/Mod-rt/.deploy-baseline"
  printf 'ok=yes\n' > "$TMPDIR_T/Mod/Mod-rt/module.lexicon"
  printf 'bad=\xc3\xa9\n' > "$TMPDIR_T/Mod/Mod-rt/.deploy-baseline/module.lexicon"
  run "$LA" "$TMPDIR_T/Mod"
  # The .deploy-baseline lexicon has non-ASCII; the real one is clean.
  # With D9b pruning, only the real one is scanned -> exit 0.
  [ "$status" -eq 0 ]
}
