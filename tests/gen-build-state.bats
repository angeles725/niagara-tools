#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/gen-build-state.sh — the GENERATED parts of the kit block of
# BUILD-STATE.md. Every kit PR used to edit the same `last_change:` line, prepend `open_issues:` items at
# the same spot and append a session-log line at the same spot, so parallel PRs conflicted there and
# "keep both sides" resolutions lost or duplicated entries. Each change now adds one file under
# build-n4-module-kit/ledger.d/ and the three parts are generated from those files.
# Exits: 0 written / fresh · 1 stale, ledger fragment or BUILD-STATE structure violation · 3 usage/env.
# [ev: retro ledger-fragments]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  GEN="$KIT/toolbelt/gen-build-state.sh"
  SWEEP="$KIT/toolbelt/sweep-build-state.sh"
  FIX="$(cd "$BATS_TEST_DIRNAME" && pwd)/fixtures/gen-build-state"
  FK="$BATS_TEST_TMPDIR/kit"
  mkdir -p "$FK/retros"
  cp "$FIX/BUILD-STATE.md" "$FK/"
  cp -R "$FIX/ledger.d" "$FK/"
}

_frag() { # _frag <file> <date> <log text> [open_issue text]
  {
    printf '<!-- ledger: kit -->\n<!-- date: %s -->\n<!-- last-change-note: %s note -->\n' "$2" "$1"
    printf 'log: %s · %s\n' "$2" "$3"
    [ -z "${4:-}" ] || printf 'open_issue: %s\n' "$4"
  } > "$FK/ledger.d/$1"
}

@test "GBS-usage: an unknown flag or extra args exit 3" {
  run bash "$GEN" --bogus "$FK"
  [ "$status" -eq 3 ]
  run bash "$GEN" "$FK" "$FK"
  [ "$status" -eq 3 ]
  run bash "$GEN" "$BATS_TEST_TMPDIR/nope"
  [ "$status" -eq 3 ]
}

@test "GBS-real: the committed BUILD-STATE.md is fresh against ledger.d/" {
  run bash "$GEN" --check
  [ "$status" -eq 0 ]
  [[ "$output" == *"is fresh"* ]]
}

@test "GBS-fixture: the archive-only fixture regenerates byte for byte" {
  run bash "$GEN" --check "$FK"
  [ "$status" -eq 0 ]
  run bash "$GEN" "$FK"
  [ "$status" -eq 0 ]
  cmp "$FIX/BUILD-STATE.md" "$FK/BUILD-STATE.md"
}

@test "GBS-stale: a new fragment makes --check exit 1 naming the file; regenerating appends the log and prepends the issue" {
  # Mutation: GBS-stale -- dropping the compare makes --check vacuously clean.
  _frag 2026-10-24-new.md 2026-10-24 "new change" "OWED new half"
  run bash "$GEN" --check "$FK"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BUILD-STATE.md is stale vs ledger.d/"* ]]
  run bash "$GEN" "$FK"
  [ "$status" -eq 0 ]
  grep -qxF 'last_change: 2026-10-24              # 2026-10-24-new.md note' "$FK/BUILD-STATE.md"
  [ "$(grep -A1 -xF 'open_issues:' "$FK/BUILD-STATE.md" | tail -1)" = "  - OWED new half" ]
  [ "$(grep -B1 -xF '<!-- /build-state.v1 -->' "$FK/BUILD-STATE.md" | tail -2 | head -1)" = "2026-10-24 · new change" ]
  run bash "$GEN" --check "$FK"
  [ "$status" -eq 0 ]
}

@test "GBS-handedit: a hand edit of a generated part is stale (the fragment is the only source)" {
  sed -i 's/^last_change: .*/last_change: 2030-01-01/' "$FK/BUILD-STATE.md"
  run bash "$GEN" --check "$FK"
  [ "$status" -eq 1 ]
}

@test "GBS-malformed: a malformed fragment is exit 1 naming the file and line; nothing is written" {
  printf '<!-- ledger: kit -->\n<!-- date: 2026-10-24 -->\nlog: 2026-10-24 · x\n  - hand item\n' > "$FK/ledger.d/2026-10-24-bad.md"
  cp "$FK/BUILD-STATE.md" "$BATS_TEST_TMPDIR/before"
  run bash "$GEN" "$FK"
  [ "$status" -eq 1 ]
  [[ "$output" == *"2026-10-24-bad.md: line 4:"* ]]
  cmp "$BATS_TEST_TMPDIR/before" "$FK/BUILD-STATE.md"
}

@test "GBS-sweep: sweep-build-state fails on a stale kit block when ledger.d/ sits beside BUILD-STATE.md" {
  printf '| Retro file | review-status |\n|---|---|\n' > "$FK/retros/INDEX.md"
  run bash "$SWEEP" "$FK/BUILD-STATE.md" "$FK/retros" "$FK/retros/INDEX.md"
  [ "$status" -eq 0 ]
  _frag 2026-10-24-new.md 2026-10-24 "new change"
  run bash "$SWEEP" "$FK/BUILD-STATE.md" "$FK/retros" "$FK/retros/INDEX.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *"stale vs ledger.d"* ]]
}
