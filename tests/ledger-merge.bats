#!/usr/bin/env bats
# ledger-merge.bats — two kit branches that each add a ledger.d/ fragment and a retro merge cleanly
# with a plain `git merge`. Before R10-D1 every such PR regenerated the committed BUILD-STATE.md
# (last_change line, open_issues items, session log) and retros/INDEX.md (new rows at the end), so
# the second PR to land always conflicted there and needed an extra `git merge origin/main`.
# GitHub's mergeability ignores local merge drivers, so the fix is that the generated files are not
# committed at all: ledger.d/ + retros/ + BUILD-STATE.src.md are the only sources, the .gitignore
# keeps BUILD-STATE.md and retros/INDEX.md out of every commit. [ev: retro ledger-generated-not-committed]
#
# The temp repo copies the real kit layout (the .gitignore, the BUILD-STATE source, ledger.d/, retros/
# and the generator wrappers) and runs the generators on each branch the way a contributor does.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$REPO/build-n4-module-kit"
  W="$BATS_TEST_TMPDIR/w"
  K="$W/build-n4-module-kit"
  mkdir -p "$K/toolbelt" "$K/retros"
  g init -q -b main
  cp "$REPO/.gitignore" "$W/"
  ln -s "$REPO/go" "$W/go" # the wrappers exec <toplevel>/go via go/go-tool.sh
  cp "$KIT"/BUILD-STATE*.md "$K/"
  cp -R "$KIT/ledger.d" "$K/"
  cp "$KIT"/retros/*.md "$K/retros/"
  cp "$KIT/toolbelt/gen-build-state.sh" "$KIT/toolbelt/gen-retro-index.sh" "$K/toolbelt/"
  regen
  g add -A
  g commit -qm base
}

g() { git -C "$W" -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false "$@"; }

regen() {
  bash "$K/toolbelt/gen-build-state.sh" "$K" >/dev/null
  bash "$K/toolbelt/gen-retro-index.sh" "$K" >/dev/null
}

# change <branch> <slug>: one PR's worth of kit state — a ledger fragment, a retro, regenerated, committed.
change() {
  g checkout -q -b "$1" main
  printf '<!-- ledger: kit -->\n<!-- date: 2099-01-01 -->\n<!-- last-change-note: %s -->\nlog: 2099-01-01 · %s landed\nopen_issue: OWED %s follow-up\n' \
    "$2" "$2" "$2" > "$K/ledger.d/2099-01-01-$2.md"
  printf '<!-- review-status: pending -->\n<!-- module: kit -->\n<!-- deltas: 1 -->\n# %s\n' "$2" > "$K/retros/2099-01-01-$2.md"
  regen
  g add -A
  g commit -qm "feat: $2"
}

@test "LM1: two branches each adding a ledger fragment and a retro merge with plain git merge" {
  change pr-alpha alpha-change
  change pr-beta beta-change
  g checkout -q pr-alpha
  run g merge --no-edit pr-beta
  [ "$status" -eq 0 ]
  [ -z "$(g diff --name-only --diff-filter=U)" ]
}

@test "LM2: the merged tree regenerates both changes; neither generated file is tracked" {
  change pr-alpha alpha-change
  change pr-beta beta-change
  g checkout -q pr-alpha
  g merge -q --no-edit pr-beta
  regen
  grep -qF '2099-01-01 · alpha-change landed' "$K/BUILD-STATE.md"
  grep -qF '2099-01-01 · beta-change landed' "$K/BUILD-STATE.md"
  grep -qF '| 2099-01-01-alpha-change.md |' "$K/retros/INDEX.md"
  grep -qF '| 2099-01-01-beta-change.md |' "$K/retros/INDEX.md"
  [ -z "$(g ls-files build-n4-module-kit/BUILD-STATE.md build-n4-module-kit/retros/INDEX.md)" ]
}
