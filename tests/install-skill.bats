#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# RED-FIRST tests for scripts/install-skill.sh (Campaign 6 PR6 T6.2/T6.3).
# Contract: openspec/changes/build-n4-module-campaign6/{spec.md R6-5, design.md §D5/§5}.
#
# install-skill.sh [--home <dir>] [--dry-run] [--force]
#   Source of truth: build-n4-module-kit/skill/SKILL.md (tracked copy)
#   Target:         <home>/.claude/skills/build-n4-module/SKILL.md
#
# Exit codes:
#   0  installed or already current
#   1  installed copy diverged and --force absent
#   2  usage error
#   3  env error (target dir not creatable)
#
# Every test passes --home "$BATS_TEST_TMPDIR/home" — no test touches real $HOME.
# Suite is identical under HOME=/nonexistent (no $HOME coupling in the installer).
#
# Named mutation (IS1): installer drops the last line of the copy -> cmp fails.
#
# IS2: second run exits 0 "already current".
# IS3: modify the installed copy -> exit 1 without --force; exit 0 + parity with --force.
# IS4: --dry-run writes nothing (target absent) and exits 0.

REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"  # no git: works from an export too
SCRIPT="$REPO_ROOT/scripts/install-skill.sh"
TRACKED="$REPO_ROOT/build-n4-module-kit/skill/SKILL.md"

setup() {
  TEST_HOME="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TEST_HOME"
  TARGET="$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
}

# ---------------------------------------------------------------------------
# IS1 — install and verify byte parity with the tracked copy
# ---------------------------------------------------------------------------
@test "IS1: install copies SKILL.md byte-identical to tracked copy" {
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  TARGET="$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  [ -f "$TARGET" ]
  cmp -s "$TRACKED" "$TARGET"
}

# ---------------------------------------------------------------------------
# IS2 — second run exits 0 "already current"
# ---------------------------------------------------------------------------
@test "IS2: second install exits 0 (already current)" {
  # First install
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  # Second run must also exit 0
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
}

# ---------------------------------------------------------------------------
# IS3a — modified copy exits 1 without --force
# ---------------------------------------------------------------------------
@test "IS3a: diverged copy exits 1 without --force" {
  # Install first
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  TARGET="$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  # Corrupt the installed copy
  printf '\nEXTRA LINE\n' >> "$TARGET"
  # Without --force must exit 1
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 1 ]
}

# ---------------------------------------------------------------------------
# IS3b — modified copy + --force exits 0 and parity restored
# ---------------------------------------------------------------------------
@test "IS3b: diverged copy + --force exits 0 and restores parity" {
  # Install first
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  TARGET="$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  # Corrupt the installed copy
  printf '\nEXTRA LINE\n' >> "$TARGET"
  # With --force must exit 0 and restore parity
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --force
  [ "$status" -eq 0 ]
  cmp -s "$TRACKED" "$TARGET"
}

# ---------------------------------------------------------------------------
# IS4 — --dry-run writes nothing and exits 0 (target absent afterward)
# ---------------------------------------------------------------------------
@test "IS4: --dry-run writes nothing and exits 0" {
  TARGET="$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  [ ! -f "$TARGET" ]  # must not exist before
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --dry-run
  [ "$status" -eq 0 ]
  [ ! -f "$TARGET" ]  # must not exist after
}

# ---------------------------------------------------------------------------
# --skill <build-n4-module|mcp-n4> (mcp-n4-kit T5)
#
# These tests run a COPY of the script inside a throwaway tree holding stub
# skill sources, so they do not depend on the real mcp-n4 skill file.
# Layout of the throwaway tree: <tree>/scripts/install-skill.sh and
#   <tree>/build-n4-module-kit/skill/SKILL.md
#   <tree>/mcp-n4-kit/skill/SKILL.md
# ---------------------------------------------------------------------------
make_tree() {
  TREE="$BATS_TEST_TMPDIR/tree"
  mkdir -p "$TREE/scripts" "$TREE/build-n4-module-kit/skill" "$TREE/mcp-n4-kit/skill"
  cp "$SCRIPT" "$TREE/scripts/install-skill.sh"
  printf 'stub build skill\n' > "$TREE/build-n4-module-kit/skill/SKILL.md"
  printf 'stub mcp skill\n' > "$TREE/mcp-n4-kit/skill/SKILL.md"
  TSCRIPT="$TREE/scripts/install-skill.sh"
}

@test "SK1: --skill mcp-n4 installs the mcp-n4 launcher byte-identical" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  cmp -s "$TREE/mcp-n4-kit/skill/SKILL.md" "$TEST_HOME/.claude/skills/mcp-n4/SKILL.md"
  [ ! -e "$TEST_HOME/.claude/skills/build-n4-module" ]
}

@test "SK2: --skill build-n4-module is the explicit form of the default" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill build-n4-module
  [ "$status" -eq 0 ]
  cmp -s "$TREE/build-n4-module-kit/skill/SKILL.md" "$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  [ ! -e "$TEST_HOME/.claude/skills/mcp-n4" ]
}

@test "SK3: without --skill the default stays build-n4-module" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  [ -f "$TEST_HOME/.claude/skills/build-n4-module/SKILL.md" ]
  [ ! -e "$TEST_HOME/.claude/skills/mcp-n4" ]
}

@test "SK4: unknown --skill name exits 2 and writes nothing" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill nope
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown skill"* ]]
  [ ! -e "$TEST_HOME/.claude" ]
}

@test "SK5: --skill without an argument exits 2" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill
  [ "$status" -eq 2 ]
}

@test "SK6: mcp-n4 diverged copy exits 1 without --force, 0 and parity with --force" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  printf 'EXTRA\n' >> "$TEST_HOME/.claude/skills/mcp-n4/SKILL.md"
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 1 ]
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4 --force
  [ "$status" -eq 0 ]
  cmp -s "$TREE/mcp-n4-kit/skill/SKILL.md" "$TEST_HOME/.claude/skills/mcp-n4/SKILL.md"
}

@test "SK7: mcp-n4 second run is already current; --dry-run writes nothing" {
  make_tree
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4 --dry-run
  [ "$status" -eq 0 ]
  [ ! -e "$TEST_HOME/.claude" ]
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  [[ "$output" == *"installed"* ]]
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  [[ "$output" == *"already current"* ]]
}

@test "SK10: the skills table is the single source of the names, paths and messages" {
  make_tree
  sed -i 's|^SKILLS="|SKILLS="extra:extra-kit/skill/SKILL.md |' "$TSCRIPT"
  mkdir -p "$TREE/extra-kit/skill"
  printf 'stub extra skill\n' > "$TREE/extra-kit/skill/SKILL.md"
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill extra
  [ "$status" -eq 0 ]
  cmp -s "$TREE/extra-kit/skill/SKILL.md" "$TEST_HOME/.claude/skills/extra/SKILL.md"
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill nope
  [ "$status" -eq 2 ]
  [[ "$output" == *"extra or build-n4-module or mcp-n4"* ]]
  [[ "$output" == *"extra|build-n4-module|mcp-n4"* ]]
}

@test "SK8: a missing tracked source for the chosen skill exits 3" {
  make_tree
  rm "$TREE/mcp-n4-kit/skill/SKILL.md"
  run env HOME=/nonexistent bash "$TSCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 3 ]
}

@test "SK9: the tracked mcp-n4 launcher installs byte-identical from the real repo" {
  REAL="$REPO_ROOT/mcp-n4-kit/skill/SKILL.md"
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  cmp -s "$REAL" "$TEST_HOME/.claude/skills/mcp-n4/SKILL.md"
  grep -q '^name: mcp-n4$' "$REAL"
}
