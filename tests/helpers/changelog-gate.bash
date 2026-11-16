#!/usr/bin/env bash
# changelog-gate.bash — shared bats helpers for the changelog gate suites: a throwaway git repository with a
# CHANGELOG.md holding an empty `## [Unreleased]` and one release, and runners for the gate and the pre-push hook.
# Loaded with `load helpers/changelog-gate` from tests/changelog-gate.bats and tests/changelog-gate-unreleased.bats.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  GATE="$REPO/scripts/changelog-gate.sh"
  R="$BATS_TEST_TMPDIR/repo"
  mkdir -p "$R/go" "$R/docs"
  git -C "$R" init -q -b main
  git -C "$R" config user.email t@example.invalid
  git -C "$R" config user.name tester
  git -C "$R" config commit.gpgsign false
  printf '# Changelog\n\n## [Unreleased]\n\n---\n\n## [v0.1.0] - 2026-01-01\n\n### Added — first\n\n- the first tool\n' \
    > "$R/CHANGELOG.md"
  printf 'package x\n' > "$R/go/x.go"
  printf '# doc\n' > "$R/docs/a.md"
  git -C "$R" add -A && git -C "$R" commit -qm "chore: base"
  BASE=$(git -C "$R" rev-parse HEAD)
}

commit() { # commit <message>: commit every change in the repo
  git -C "$R" add -A && git -C "$R" commit -qm "$1"
}

gate() { # gate [range]: run the gate over BASE..HEAD (or the given range) inside the repo
  local range="${1:-$BASE..HEAD}"
  cd "$R" && run bash "$GATE" --range "$range"
}

fragment() { # fragment <name> <content>
  mkdir -p "$R/changelog.d"; printf '%s' "$2" > "$R/changelog.d/$1"
}

hook() { # hook: run .githooks/pre-push in the repo over BASE..HEAD (an existing remote branch)
  cd "$R" && run env NIAGARA_CLIENT_DENYLIST="$BATS_TEST_TMPDIR/no-client-denylist" \
    bash "$REPO/.githooks/pre-push" origin https://example.invalid/repo.git \
    <<<"refs/heads/x $(git -C "$R" rev-parse HEAD) refs/heads/x $BASE"
}
