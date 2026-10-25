#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# gen-changelog.bats — scripts/gen-changelog.sh (go/internal/changelog fold): the release step that folds every
# changelog.d/<YYYY-MM-DD>-<slug>.md fragment into CHANGELOG.md's `## [Unreleased]` section.
#   --check  every fragment is well formed and the [Unreleased] header exists (CI runs it)
#   --write  append the fragments, date+slug order, at the end of [Unreleased], then delete them
# Exits: 0 ok · 1 a malformed fragment or no [Unreleased] header (nothing written or deleted) · 3 usage/env.
# Hermetic: a fixture repository root in $BATS_TEST_TMPDIR; no git.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  GEN="$REPO/scripts/gen-changelog.sh"
  R="$BATS_TEST_TMPDIR/root"
  mkdir -p "$R/changelog.d"
  printf '# Changelog\n\n## [Unreleased]\n\n### Added — kept\n\n- an entry already here\n\n---\n\n## [v0.1.0] - 2026-01-01\n\n- old\n' \
    > "$R/CHANGELOG.md"
}

frag() { printf '%s' "$2" > "$R/changelog.d/$1"; }

@test "GC1: --write folds the fragments in date+slug order at the end of [Unreleased] and deletes them" {
  # Mutation: GC1 -- inserting right after the header (not before the section's ---) puts b before the kept entry.
  frag 2026-10-25-b.md $'### Fixed — b\n\n- b fixed\n'
  frag 2026-10-24-z.md $'### Added — z\n\n- z added\n'
  frag 2026-10-25-a.md $'### Changed — a\n\n- a changed\n'
  run bash "$GEN" --write "$R"
  [ "$status" -eq 0 ]
  [ ! -e "$R/changelog.d/2026-10-25-a.md" ]
  [ ! -e "$R/changelog.d/2026-10-25-b.md" ]
  [ ! -e "$R/changelog.d/2026-10-24-z.md" ]
  run cat "$R/CHANGELOG.md"
  [ "$output" = "# Changelog

## [Unreleased]

### Added — kept

- an entry already here

### Added — z

- z added

### Changed — a

- a changed

### Fixed — b

- b fixed

---

## [v0.1.0] - 2026-01-01

- old" ]
}

@test "GC2: --write into an empty [Unreleased] section keeps one blank line around the fold" {
  printf '# Changelog\n\n## [Unreleased]\n\n---\n\n## [v0.1.0] - 2026-01-01\n' > "$R/CHANGELOG.md"
  frag 2026-10-25-a.md $'### Added — a\n\n- a\n'
  run bash "$GEN" --write "$R"
  [ "$status" -eq 0 ]
  run cat "$R/CHANGELOG.md"
  [ "$output" = $'# Changelog\n\n## [Unreleased]\n\n### Added — a\n\n- a\n\n---\n\n## [v0.1.0] - 2026-01-01' ]
}

@test "GC3: --check passes valid fragments and changes nothing" {
  frag 2026-10-25-a.md $'### Added — a\n\n- a\n'
  cp "$R/CHANGELOG.md" "$BATS_TEST_TMPDIR/before"
  run bash "$GEN" --check "$R"
  [ "$status" -eq 0 ]
  cmp "$R/CHANGELOG.md" "$BATS_TEST_TMPDIR/before"
  [ -e "$R/changelog.d/2026-10-25-a.md" ]
}

@test "GC4: a malformed fragment fails --check and --write; --write writes and deletes nothing" {
  # Mutation: GC4 -- folding the valid fragments before reporting the bad one leaves CHANGELOG.md changed.
  frag 2026-10-25-a.md $'### Added — a\n\n- a\n'
  frag 2026-10-25-bad.md $'no heading here\n'
  run bash "$GEN" --check "$R"
  [ "$status" -eq 1 ]
  [[ "$output" == *"2026-10-25-bad.md"* ]]
  cp "$R/CHANGELOG.md" "$BATS_TEST_TMPDIR/before"
  run bash "$GEN" --write "$R"
  [ "$status" -eq 1 ]
  cmp "$R/CHANGELOG.md" "$BATS_TEST_TMPDIR/before"
  [ -e "$R/changelog.d/2026-10-25-a.md" ]
}

@test "GC5: an empty fragment and a badly named file fail --check" {
  frag 2026-10-25-empty.md ''
  run bash "$GEN" --check "$R"
  [ "$status" -eq 1 ]
  rm "$R/changelog.d/2026-10-25-empty.md"
  frag notes.md $'### Added — a\n\n- a\n'
  run bash "$GEN" --check "$R"
  [ "$status" -eq 1 ]
  [[ "$output" == *"notes.md"* ]]
}

@test "GC6: no changelog.d/ is valid and --write leaves CHANGELOG.md unchanged" {
  rm -r "$R/changelog.d"
  cp "$R/CHANGELOG.md" "$BATS_TEST_TMPDIR/before"
  run bash "$GEN" --check "$R"
  [ "$status" -eq 0 ]
  run bash "$GEN" --write "$R"
  [ "$status" -eq 0 ]
  cmp "$R/CHANGELOG.md" "$BATS_TEST_TMPDIR/before"
}

@test "GC7: a CHANGELOG.md with no [Unreleased] header fails (1); no CHANGELOG.md or no mode is usage/env (3)" {
  printf '# Changelog\n\n## [v0.1.0]\n' > "$R/CHANGELOG.md"
  run bash "$GEN" --check "$R"
  [ "$status" -eq 1 ]
  rm "$R/CHANGELOG.md"
  run bash "$GEN" --check "$R"
  [ "$status" -eq 3 ]
  run bash "$GEN" "$R"
  [ "$status" -eq 3 ]
}

@test "GC8: the repository's own changelog.d/ and CHANGELOG.md pass --check (default root)" {
  run bash "$GEN" --check
  [ "$status" -eq 0 ]
}

@test "GC-ci: ci.yml runs gen-changelog --check and the changelog gate" {
  # Mutation: GC-ci -- deleting either run line from ci.yml fails this test.
  grep -qE '^ +run: bash scripts/gen-changelog\.sh --check$' "$REPO/.github/workflows/ci.yml"
  grep -qE '^ +run: bash scripts/changelog-gate\.sh --range ' "$REPO/.github/workflows/ci.yml"
}
