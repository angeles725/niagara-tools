#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# changelog-gate-unreleased.bats — scripts/changelog-gate.sh (go/internal/changelog): `## [Unreleased]` of
# CHANGELOG.md is written only by the release fold (scripts/gen-changelog.sh --write). A range that adds a
# non-blank line inside it FAILS unless the range is a release: it deletes a changelog.d/ fragment (the fold) or
# changes VERSION (the bump). Two open PRs that both add lines there conflict; fragments never do.
# Hermetic: every test builds a throwaway git repository in $BATS_TEST_TMPDIR (generic names only).

load helpers/changelog-gate

unreleased_line() { # add one entry inside the empty [Unreleased] section
  sed -i 's/^## \[Unreleased\]$/## [Unreleased]\n\n### Changed — x\n\n- x changed/' "$R/CHANGELOG.md"
}

@test "CG-u1: a go/ change plus a line inside [Unreleased] FAILS and points to changelog.d/" {
  # Mutation: CG-u1 -- accepting an [Unreleased] line as a carrier again passes this range.
  printf 'package x // changed\n' > "$R/go/x.go"; unreleased_line; commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"[Unreleased] gains a line outside a release"* ]]
  [[ "$output" == *"changelog.d/<YYYY-MM-DD>-<slug>.md"* ]]
}

@test "CG-u2: a line inside [Unreleased] FAILS even when the range is otherwise not applicable" {
  # Mutation: CG-u2 -- checking [Unreleased] only after the not-applicable exit passes this range.
  printf '# doc changed\n' > "$R/docs/a.md"; unreleased_line; commit "docs: change a"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"[Unreleased] gains a line outside a release"* ]]
}

@test "CG-u3: a valid fragment does not excuse a line inside [Unreleased] (FAILS)" {
  printf 'package x // changed\n' > "$R/go/x.go"; unreleased_line
  fragment 2026-11-15-change-x.md $'### Changed — x\n\n- x changed\n'
  commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"[Unreleased] gains a line outside a release"* ]]
}

@test "CG-u4: the missing-entry fix no longer offers an [Unreleased] line" {
  printf 'package x // changed\n' > "$R/go/x.go"; commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"changelog.d/"* ]]
  [[ "$output" != *"add a line under CHANGELOG.md"* ]]
}

release_sequence() { # FEAT: a go/ change with a fragment; FOLD: gen-changelog --write; BUMP: CONTRIBUTING § 5
  printf 'package x // changed\n' > "$R/go/x.go"
  fragment 2026-11-15-change-x.md $'### Changed — x\n\n- x changed\n'
  commit "feat: change x"; FEAT=$(git -C "$R" rev-parse HEAD)
  bash "$REPO/scripts/gen-changelog.sh" --write "$R" >/dev/null || { echo "the fold failed"; return 1; }
  [ ! -e "$R/changelog.d/2026-11-15-change-x.md" ]
  grep -q '^- x changed$' "$R/CHANGELOG.md"
  commit "chore: fold changelog.d"; FOLD=$(git -C "$R" rev-parse HEAD)
  printf '# Changelog\n\n## [Unreleased]\n\n---\n\n## [v0.2.0] - 2026-11-15\n\n### Changed — x\n\n- x changed\n\n---\n\n## [v0.1.0] - 2026-01-01\n\n### Added — first\n\n- the first tool\n' \
    > "$R/CHANGELOG.md"
  printf '0.2.0\n' > "$R/VERSION"
  commit "chore(release): v0.2.0"
}

@test "CG-u5: the real release sequence PASSES as one range and as separate fold and bump ranges" {
  # Mutation: CG-u5 -- a VERSION-only discriminator refuses the fold range FEAT..FOLD; a fragment-deletion-only
  # one refuses the bump FOLD..HEAD, whose diff adds the `---` separator inside the emptied [Unreleased].
  # BASE..HEAD (the feature, the fold and the bump in one push) is left out: its fragment is added and deleted
  # inside the range, so the gate sees no carrier; that predates this rule and a release PR never holds a feature.
  release_sequence
  local rng
  for rng in "$FEAT..$FOLD" "$FOLD..HEAD" "$FEAT..HEAD"; do
    gate "$rng"
    [ "$status" -eq 0 ] || { echo "refused $rng: $output"; return 1; }
  done
}

@test "CG-u6: a release commit that also changes go/ PASSES on its folded [Unreleased] entry" {
  printf 'package x // changed\n' > "$R/go/x.go"
  fragment 2026-11-15-change-x.md $'### Changed — x\n\n- x changed\n'
  commit "docs: fragment only"; local mid; mid=$(git -C "$R" rev-parse HEAD)
  run bash "$REPO/scripts/gen-changelog.sh" --write "$R"
  [ "$status" -eq 0 ]
  printf 'package x // changed again\n' > "$R/go/x.go"
  commit "feat: change x and fold"
  gate "$mid..HEAD"
  [ "$status" -eq 0 ]
  [[ "$output" == *"[Unreleased]"* ]]
}

@test "CG-u7: a valid Changelog: none trailer does not excuse a line inside [Unreleased] (FAILS)" {
  printf 'package x // changed\n' > "$R/go/x.go"; unreleased_line
  commit $'feat: change x\n\nChangelog: none (internal only)'
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"[Unreleased] gains a line outside a release"* ]]
}

@test "CG-u8: .githooks/pre-push refuses a line inside [Unreleased] next to a valid fragment" {
  printf 'package x // changed\n' > "$R/go/x.go"; unreleased_line
  fragment 2026-11-15-change-x.md $'### Changed — x\n\n- x changed\n'
  commit $'feat: change x\n\nRetro: none (trivial: probe)'
  hook
  [ "$status" -ne 0 ]
  [[ "$output" == *"[Unreleased] gains a line outside a release"* ]]
}

@test "CG-u9: a VERSION bump with a go/ change and an [Unreleased] line PASSES as a release" {
  printf 'package x // changed\n' > "$R/go/x.go"; unreleased_line; printf '0.2.0\n' > "$R/VERSION"
  commit "chore(release): v0.2.0"
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"from the release fold"* ]]
}

@test "CG-u10: deleting a changelog.d/ file that is not a fragment name does not make a release (FAILS)" {
  # Mutation: CG-u10 -- counting any deleted changelog.d/ path as the fold passes this range.
  mkdir -p "$R/changelog.d"; printf 'x\n' > "$R/changelog.d/README"; commit "docs: stray file"
  local base; base=$(git -C "$R" rev-parse HEAD)
  rm "$R/changelog.d/README"; printf 'package x // changed\n' > "$R/go/x.go"; unreleased_line
  commit "feat: change x"
  gate "$base..HEAD"
  [ "$status" -eq 1 ]
  [[ "$output" == *"[Unreleased] gains a line outside a release"* ]]
}
