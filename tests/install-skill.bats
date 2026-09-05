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

SCRIPT="$(git -C "$(dirname "$BATS_TEST_FILENAME")" rev-parse --show-toplevel)/scripts/install-skill.sh"
TRACKED="$(git -C "$(dirname "$BATS_TEST_FILENAME")" rev-parse --show-toplevel)/build-n4-module-kit/skill/SKILL.md"

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
