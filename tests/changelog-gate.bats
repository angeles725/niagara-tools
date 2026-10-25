#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# changelog-gate.bats — scripts/changelog-gate.sh (go/internal/changelog): a range that changes kit behaviour
# (go/, build-n4-module-kit/, tools/) must carry a CHANGELOG entry: a changelog.d/<YYYY-MM-DD>-<slug>.md
# fragment added in the range, or a line added inside CHANGELOG.md's `## [Unreleased]` section, or a
# `Changelog: none (<reason>)` commit trailer. A malformed fragment or trailer fails the range whatever else it
# carries. Exits: 0 pass or not applicable · 1 missing entry or malformed fragment/trailer · 3 usage/env.
# Hermetic: every test builds a throwaway git repository in $BATS_TEST_TMPDIR (generic names only).

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

@test "CG-a: a go/ change with no CHANGELOG entry FAILS, naming the paths and the fix" {
  printf 'package x // changed\n' > "$R/go/x.go"; commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"go/x.go"* ]]
  [[ "$output" == *"changelog.d/"* ]]
  [[ "$output" == *"## [Unreleased]"* ]]
  [[ "$output" == *"Changelog: none ("* ]]
}

@test "CG-b: the same change plus a line added inside [Unreleased] PASSES" {
  printf 'package x // changed\n' > "$R/go/x.go"
  sed -i 's/^## \[Unreleased\]$/## [Unreleased]\n\n### Changed — x\n\n- x changed/' "$R/CHANGELOG.md"
  commit "feat: change x"
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"[Unreleased]"* ]]
}

@test "CG-c: an edit to an OLDER release section only does not count (FAILS)" {
  # Mutation: CG-c -- accepting any CHANGELOG.md change (dropping the [Unreleased] bounds) passes this range.
  printf 'package x // changed\n' > "$R/go/x.go"
  sed -i 's/^- the first tool$/- the first tool, reworded\n- one more old line/' "$R/CHANGELOG.md"
  commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"go/x.go"* ]]
}

@test "CG-c2: a blank line added inside [Unreleased] is not an entry (FAILS)" {
  printf 'package x // changed\n' > "$R/go/x.go"
  sed -i 's/^## \[Unreleased\]$/## [Unreleased]\n/' "$R/CHANGELOG.md"
  commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
}

@test "CG-d: a 'Changelog: none (<reason>)' trailer PASSES and prints the reason" {
  printf 'package x // changed\n' > "$R/go/x.go"
  commit $'test: rename a helper\n\nChangelog: none (test-only rename, no behaviour change)'
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"test-only rename, no behaviour change"* ]]
}

@test "CG-e: a trailer with an empty reason FAILS" {
  printf 'package x // changed\n' > "$R/go/x.go"
  commit $'test: x\n\nChangelog: none ( )'
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"Changelog: none ( )"* ]]
}

@test "CG-e2: a trailer whose reason is longer than the cap FAILS" {
  printf 'package x // changed\n' > "$R/go/x.go"
  commit "test: x

Changelog: none ($(printf 'r%.0s' $(seq 1 130)))"
  gate
  [ "$status" -eq 1 ]
}

@test "CG-e3: a 'Changelog:' line that is not the none (<reason>) form FAILS" {
  printf 'package x // changed\n' > "$R/go/x.go"
  commit $'test: x\n\nChangelog: none'
  gate
  [ "$status" -eq 1 ]
}

@test "CG-f: a range touching only files outside go/, build-n4-module-kit/ and tools/ PASSES (not applicable)" {
  printf '# doc changed\n' > "$R/docs/a.md"; commit "docs: a"
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"not applicable"* ]]
}

@test "CG-f2: the bookkeeping paths (retros/, ledger.d/, BUILD-STATE.src.md, go/bin/) are not behaviour" {
  mkdir -p "$R/build-n4-module-kit/retros" "$R/build-n4-module-kit/ledger.d" "$R/go/bin"
  printf 'r\n' > "$R/build-n4-module-kit/retros/2026-01-02-r.md"
  printf 'l\n' > "$R/build-n4-module-kit/ledger.d/2026-01-02-l.md"
  printf 's\n' > "$R/build-n4-module-kit/BUILD-STATE.src.md"
  printf 'b\n' > "$R/go/bin/tool"
  commit "docs: bookkeeping"
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"not applicable"* ]]
}

@test "CG-f3: build-n4-module-kit/ and tools/ count like go/" {
  mkdir -p "$R/build-n4-module-kit/toolbelt" "$R/tools"
  printf 'echo\n' > "$R/build-n4-module-kit/toolbelt/t.sh"
  printf 'echo\n' > "$R/tools/t.py"
  commit "feat: t"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"build-n4-module-kit/toolbelt/t.sh"* ]]
  [[ "$output" == *"tools/t.py"* ]]
}

@test "CG-g: a malformed range exits 3" {
  gate "not-a-range"
  [ "$status" -eq 3 ]
  gate "$BASE..no-such-ref"
  [ "$status" -eq 3 ]
}

@test "CG-g2: no git on PATH exits 3" {
  local bin="$BATS_TEST_TMPDIR/nogit"
  mkdir -p "$bin"
  bash "$REPO/scripts/changelog-gate.sh" --help >/dev/null 2>&1 || true   # build the binary while go is on PATH
  local t; for t in bash dirname readlink find; do ln -s "$(command -v "$t")" "$bin/$t"; done
  cd "$R" && run env PATH="$bin" bash "$GATE" --range "$BASE..HEAD"
  [ "$status" -eq 3 ]
  [[ "$output" == *"git is not on PATH"* ]]
}

@test "CG-h: a valid changelog.d fragment PASSES and names it" {
  printf 'package x // changed\n' > "$R/go/x.go"
  fragment 2026-10-25-change-x.md $'### Changed — x\n\n- x changed\n'
  commit "feat: change x"
  gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"changelog.d/2026-10-25-change-x.md"* ]]
}

@test "CG-i: an empty fragment FAILS even with a valid [Unreleased] line" {
  printf 'package x // changed\n' > "$R/go/x.go"
  sed -i 's/^## \[Unreleased\]$/## [Unreleased]\n\n### Changed — x\n\n- x changed/' "$R/CHANGELOG.md"
  fragment 2026-10-25-empty.md ''
  commit "feat: change x"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"2026-10-25-empty.md"* ]]
}

@test "CG-j: malformed fragments FAIL (no heading, no bullet, bad name, wrong heading level)" {
  printf 'package x // changed\n' > "$R/go/x.go"; commit "feat: change x"
  local bad
  for bad in $'- a bullet with no heading\n' $'### Added — x\n\nprose but no bullet\n' \
             $'### Improved — x\n\n- not a Keep a Changelog type\n' $'## [v9.9.9]\n\n- a release header\n'; do
    rm -rf "$R/changelog.d"; fragment 2026-10-25-bad.md "$bad"; commit "docs: frag"
    gate; [ "$status" -eq 1 ] || { echo "passed: $bad"; return 1; }
    git -C "$R" reset -q --hard HEAD~1
  done
  fragment NoDate.md $'### Added — x\n\n- fine body, bad name\n'; commit "docs: frag"
  gate
  [ "$status" -eq 1 ]
  [[ "$output" == *"NoDate.md"* ]]
}

@test "CG-k: a malformed fragment FAILS even when the range is otherwise not applicable" {
  fragment 2026-10-25-bad.md $'no heading\n'; commit "docs: frag"
  gate
  [ "$status" -eq 1 ]
}

hook() { # hook: run .githooks/pre-push in the repo over BASE..HEAD (an existing remote branch)
  cd "$R" && run env NIAGARA_CLIENT_DENYLIST="$BATS_TEST_TMPDIR/no-client-denylist" \
    bash "$REPO/.githooks/pre-push" origin https://example.invalid/repo.git \
    <<<"refs/heads/x $(git -C "$R" rev-parse HEAD) refs/heads/x $BASE"
}

@test "CG-hook1: .githooks/pre-push refuses a go/ change with no CHANGELOG entry (retro trailer alone is not enough)" {
  # Mutation: CG-hook -- dropping the changelog-gate line from .githooks/pre-push passes this push.
  printf 'package x // changed\n' > "$R/go/x.go"
  commit $'feat: change x\n\nRetro: none (trivial: fixture)'
  hook
  [ "$status" -eq 1 ]
  [[ "$output" == *"no CHANGELOG entry"* ]]
}

@test "CG-hook2: .githooks/pre-push passes the same push once it carries a fragment" {
  printf 'package x // changed\n' > "$R/go/x.go"
  fragment 2026-10-25-change-x.md $'### Changed — x\n\n- x changed\n'
  commit $'feat: change x\n\nRetro: none (trivial: fixture)'
  hook
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS — fragment changelog.d/2026-10-25-change-x.md"* ]]
}
