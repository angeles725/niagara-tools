#!/usr/bin/env bats
# lint-lexicon-apostrophe.bats — a lexicon value with a MessageFormat argument and a single apostrophe WARNs (LX1).
# The rule's golden cases live in go/internal/lint/rules/lexiconapostrophe. [ev: retro course-coverage Δ26]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-lexicon-apostrophe.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  mkdir -p "$MOD/Mod-rt"
  LEX="$MOD/Mod-rt/module.lexicon"
  W=" has a MessageFormat argument and a single ' that MessageFormat drops (doesn't renders doesnt): write '' (types/lexicon.md § 3)"
}

@test "LX-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-lexicon-apostrophe.sh [--strict] <module-root>" ]
}

@test "LX-fixture: the MinimalPan module is clean under --strict" {
  run "$L" --strict "$KIT/fixtures/MinimalPan"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LX1: can't {0} WARNs; can''t {0} is clean" {
  # Mutation: LX1 -- treating a doubled '' as a lone apostrophe WARNs "can''t {0}"
  printf '%s\n' "ok=Room {0} can''t cool" "bad=Room {0} can't cool" > "$LEX"
  run "$L" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-lexicon-apostrophe  $LEX:2  LX1 bad$W" ]
  run "$L" --strict "$MOD"
  [ "$status" -eq 1 ]
}

@test "LX1-noargs: an apostrophe in a value with no argument is clean" {
  # Mutation: LX1-noargs -- dropping the argument test WARNs a value with no {n} read by getText(key)
  printf '%s\n' "idMissing=Server response doesn't contain id" > "$LEX"
  run "$L" --strict "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
