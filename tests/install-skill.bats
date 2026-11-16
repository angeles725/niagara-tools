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

# Build the Go binaries once with the caller's environment: the cases run the launcher under
# HOME=/nonexistent, where an on-demand `go build` has no build cache and fails (exit 3).
setup_file() {
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via REPO_ROOT
  . "$REPO_ROOT/go/go-tool.sh"
  go_tool_bin "$REPO_ROOT/go" install-skill >/dev/null
  go_tool_bin "$REPO_ROOT/go" kit-state >/dev/null
}

setup() {
  TEST_HOME="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TEST_HOME"
  TARGET="$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  # A build-n4-module install materializes the kit state of the kit it resolves; by default point it at
  # a stub kit with no kit-state.sh (a warning, no write), so no test writes into the real checkout.
  BUILD_N4_KIT="$BATS_TEST_TMPDIR/stubkit"; export BUILD_N4_KIT
  mkdir -p "$BUILD_N4_KIT"; : > "$BUILD_N4_KIT/METHODOLOGY.md"
}

# copy_kit — a temp kit checkout with the real sources and kit-state, for the materialize tests.
copy_kit() {
  KTOP="$BATS_TEST_TMPDIR/ktop"; BUILD_N4_KIT="$KTOP/build-n4-module-kit"; export BUILD_N4_KIT
  mkdir -p "$BUILD_N4_KIT/toolbelt" "$BUILD_N4_KIT/retros"
  ln -s "$REPO_ROOT/go" "$KTOP/go"
  cp "$REPO_ROOT/build-n4-module-kit/METHODOLOGY.md" "$REPO_ROOT/build-n4-module-kit/BUILD-STATE.src.md" "$BUILD_N4_KIT/"
  cp -R "$REPO_ROOT/build-n4-module-kit/ledger.d" "$BUILD_N4_KIT/"
  cp "$REPO_ROOT"/build-n4-module-kit/retros/*.md "$BUILD_N4_KIT/retros/"
  rm -f "$BUILD_N4_KIT/retros/INDEX.md"
  cp "$REPO_ROOT/build-n4-module-kit/toolbelt/kit-state.sh" "$BUILD_N4_KIT/toolbelt/"
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
  cp -r "$REPO_ROOT/scripts/lib" "$TREE/scripts/lib"
  ln -s "$REPO_ROOT/go" "$TREE/go"   # the launcher execs the Go command of the module next to scripts/
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

# ---------------------------------------------------------------------------
# IS6-IS9 — the generated kit state (R10-D1): BUILD-STATE.md and retros/INDEX.md are not committed, so
# a build-n4-module install leaves both in the kit checkout the launcher reads, and the launcher tells
# the reader to run kit-state. [ev: retro ledger-generated-not-committed]
# Mutation: IS6 -- dropping the materialize call installs the launcher over a kit with no BUILD-STATE.md.
# ---------------------------------------------------------------------------
@test "IS6: install leaves both generated files in the kit checkout it resolves; a rerun regenerates them" {
  copy_kit
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  [[ "$output" == *"kit-state: wrote $BUILD_N4_KIT/BUILD-STATE.md"* ]]
  [[ "$output" == *"kit-state: wrote $BUILD_N4_KIT/retros/INDEX.md"* ]]
  [ "$(grep -c '^module: ' "$BUILD_N4_KIT/BUILD-STATE.md")" -ge 2 ]   # module envelopes + the kit block
  grep -qF '| Retro file | Module | Date | review-status | deltas |' "$BUILD_N4_KIT/retros/INDEX.md"
  printf '<!-- ledger: kit -->\n<!-- date: 2099-01-01 -->\nlog: 2099-01-01 · after install\n' > "$BUILD_N4_KIT/ledger.d/2099-01-01-later.md"
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  [[ "$output" == *"already current"* ]]
  grep -qF '2099-01-01 · after install' "$BUILD_N4_KIT/BUILD-STATE.md"
}

@test "IS7: the tracked and the installed launcher point the reader at kit-state, not at a committed file" {
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  # shellcheck disable=SC2016  # literal backticks and $KIT, as the launcher spells them
  for f in "$TRACKED" "$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"; do
    grep -qF 'Orient from `$KIT/toolbelt/kit-state.sh build-state`' "$f"
    if grep -qF 'Orient from `$KIT/BUILD-STATE.md`' "$f"; then false; fi
  done
}

@test "IS8: --skill mcp-n4 never touches the build kit state" {
  copy_kit
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  [ ! -e "$BUILD_N4_KIT/BUILD-STATE.md" ] && [ ! -e "$BUILD_N4_KIT/retros/INDEX.md" ]
}

@test "IS9: a kit-state failure is exit 3 naming the kit; the launcher is still installed" {
  copy_kit
  printf '<!-- ledger: kit -->\n  - hand item\n' > "$BUILD_N4_KIT/ledger.d/2099-01-01-bad.md"
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 3 ]
  [[ "$output" == *"kit-state --write failed in $BUILD_N4_KIT"* ]]
  cmp -s "$TRACKED" "$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
  [ ! -e "$BUILD_N4_KIT/BUILD-STATE.md" ]
}

# ---------------------------------------------------------------------------
# IS5 — what the launcher needs before it reaches the Go command: a Go toolchain when the command has to be
# built, and nothing else. The digests are computed in Go, so no sha256 tool is required.
# Mutation: IS5b -- restoring the up-front `sha256_require` exits 3 under a PATH with no sha256 tool.
# ---------------------------------------------------------------------------
@test "IS5: no Go toolchain and no built command -> exit 3 naming go, nothing installed" {
  # A copy of the launcher beside a Go module with no bin/: the command must be built, and PATH has no go.
  FAKE="$BATS_TEST_TMPDIR/repo"; BIN="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$FAKE/scripts" "$FAKE/go" "$BIN"
  cp "$SCRIPT" "$FAKE/scripts/"
  cp "$REPO_ROOT/go/go-tool.sh" "$REPO_ROOT/go/go.mod" "$FAKE/go/"
  ln -s "$(command -v dirname)" "$BIN/dirname"
  run env HOME=/nonexistent PATH="$BIN" /bin/bash "$FAKE/scripts/install-skill.sh" --home "$TEST_HOME"
  [ "$status" -eq 3 ]
  [ "$output" = "install-skill: go is not on PATH; it is needed to build $FAKE/go/bin/install-skill" ]
  [ ! -e "$TEST_HOME/.claude/skills/build-n4-module" ]
}

@test "IS5b: no sha256sum and no shasum on PATH -> the launcher still installs (digests are computed in Go)" {
  BIN="$BATS_TEST_TMPDIR/bin"; mkdir -p "$BIN"
  for t in dirname find; do ln -s "$(command -v "$t")" "$BIN/$t"; done
  run env HOME=/nonexistent PATH="$BIN" /bin/bash "$SCRIPT" --home "$TEST_HOME"
  [ "$status" -eq 0 ]
  [[ "$output" != *"sha256"* ]]
  cmp -s "$TRACKED" "$TEST_HOME/.claude/skills/build-n4-module/SKILL.md"
}

# ---------------------------------------------------------------------------
# HS1-HS4 — --harness <claude|pi|gentle-shell|all> (retro 2026-11-15 H1, H2), the empty --home (S3) and a
# symlinked target (S4). PI_CODING_AGENT_DIR is unset so pi resolves under the throwaway home.
# ---------------------------------------------------------------------------
@test "HS1: --harness pi installs the launcher and the prompt file under the pi agent dir" {
  mkdir -p "$TEST_HOME/.pi/agent"
  run env -u PI_CODING_AGENT_DIR HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4 --harness pi
  [ "$status" -eq 0 ]
  cmp -s "$REPO_ROOT/mcp-n4-kit/skill/SKILL.md" "$TEST_HOME/.pi/agent/skills/mcp-n4/SKILL.md"
  grep -qF "Read the mcp-n4 skill at $TEST_HOME/.pi/agent/skills/mcp-n4/SKILL.md" "$TEST_HOME/.pi/agent/prompts/mcp-n4.md"
  [ ! -e "$TEST_HOME/.claude" ]
}

@test "HS2: --harness all skips an absent harness with an absent row, exit 0; the default stays claude only" {
  mkdir -p "$TEST_HOME/.gentle-shell/agent"
  run env -u PI_CODING_AGENT_DIR HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4
  [ "$status" -eq 0 ]
  [ ! -e "$TEST_HOME/.gentle-shell/agent/skills" ]
  run env -u PI_CODING_AGENT_DIR HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4 --harness all
  [ "$status" -eq 0 ]
  [[ "$output" == *"absent: pi ($TEST_HOME/.pi/agent not found), nothing installed"* ]]
  [ -f "$TEST_HOME/.gentle-shell/agent/prompts/mcp-n4.md" ]
  [ ! -e "$TEST_HOME/.pi" ]
}

@test "HS3: an empty --home is a usage error (exit 2), never the real home" {
  run env HOME="$TEST_HOME" bash "$SCRIPT" --home "" --skill mcp-n4
  [ "$status" -eq 2 ]
  [[ "$output" == *"--home requires a non-empty directory"* ]]
  [ ! -e "$TEST_HOME/.claude" ]
}

@test "HS4: --force refuses a symlinked SKILL.md (exit 3) and leaves the pointee unchanged" {
  mkdir -p "$TEST_HOME/.claude/skills/mcp-n4" "$BATS_TEST_TMPDIR/elsewhere"
  printf 'not ours\n' > "$BATS_TEST_TMPDIR/elsewhere/SKILL.md"
  ln -s "$BATS_TEST_TMPDIR/elsewhere/SKILL.md" "$TEST_HOME/.claude/skills/mcp-n4/SKILL.md"
  run env HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4 --force
  [ "$status" -eq 3 ]
  [[ "$output" == *"refusing to write through a symbolic link"* ]]
  [ "$(cat "$BATS_TEST_TMPDIR/elsewhere/SKILL.md")" = "not ours" ]
}

# ---------------------------------------------------------------------------
# HS5-HS7 — review follow-ups of #692 (retro 2026-11-15 T4).
# ---------------------------------------------------------------------------
@test "HS5: an explicit --home wins over PI_CODING_AGENT_DIR" {
  mkdir -p "$TEST_HOME/.pi/agent" "$BATS_TEST_TMPDIR/other-agent"
  run env PI_CODING_AGENT_DIR="$BATS_TEST_TMPDIR/other-agent" HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4 --harness pi
  [ "$status" -eq 0 ]
  [ -f "$TEST_HOME/.pi/agent/skills/mcp-n4/SKILL.md" ]
  [ ! -e "$BATS_TEST_TMPDIR/other-agent/skills" ]
}

@test "HS6: a symlinked prompts/ directory is refused (exit 3) before anything is written" {
  mkdir -p "$TEST_HOME/.pi/agent" "$BATS_TEST_TMPDIR/elsewhere"
  ln -s "$BATS_TEST_TMPDIR/elsewhere" "$TEST_HOME/.pi/agent/prompts"
  run env -u PI_CODING_AGENT_DIR HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4 --harness pi
  [ "$status" -eq 3 ]
  [[ "$output" == *"refusing to write through a symbolic link: $TEST_HOME/.pi/agent/prompts"* ]]
  [ ! -e "$TEST_HOME/.pi/agent/skills" ]
  [ -z "$(ls -A "$BATS_TEST_TMPDIR/elsewhere")" ]
}

@test "HS7: an empty --home prints the usage line" {
  run env HOME="$TEST_HOME" bash "$SCRIPT" --home ""
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage: install-skill.sh "* ]]
}

# HS8 — T4(3) residual: a file that cannot be placed is refused before the first write.
@test "HS8: a prompts/ that is a file is refused (exit 3) before SKILL.md is written" {
  mkdir -p "$TEST_HOME/.pi/agent"
  printf 'a file\n' > "$TEST_HOME/.pi/agent/prompts"
  run env -u PI_CODING_AGENT_DIR HOME=/nonexistent bash "$SCRIPT" --home "$TEST_HOME" --skill mcp-n4 --harness pi
  [ "$status" -eq 3 ]
  [[ "$output" == *"refusing to install: not a directory: $TEST_HOME/.pi/agent/prompts (move it away, then re-run)"* ]]
  [ ! -e "$TEST_HOME/.pi/agent/skills" ]
}
