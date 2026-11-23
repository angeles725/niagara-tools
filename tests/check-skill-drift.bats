#!/usr/bin/env bats
# Tests for scripts/check-skill-drift.sh — does the installed skill launcher match the tracked copy?
# [ev: retro client-source-of-truth Δ3] — the close gate fails on drift; the remedy is
# scripts/install-skill.sh --force. Every test passes --home so the real $HOME is never read.

# The script is the launcher of go/cmd/check-skill-drift: build the binary once, with the real environment, so a case that
# changes HOME never pays the build with a fresh Go cache. [ev: retro go-toolbelt-small-port Δ4]
setup_file() {
  local root; root="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via root
  . "$root/go/go-tool.sh"
  go_tool_bin "$root/go" check-skill-drift >/dev/null
}

setup() {
  TMPDIR_T="$(mktemp -d)"
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  SD="$REPO/scripts/check-skill-drift.sh"
  H="$TMPDIR_T/home"
}
teardown() { rm -rf "$TMPDIR_T"; }

_install() { "$REPO/scripts/install-skill.sh" --home "$H" "$@" >/dev/null; }

@test "SD1: installed launcher byte-identical to skill/SKILL.md -> PASS, exit 0" {
  _install
  run "$SD" --home "$H"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  skill-drift"*"build-n4-module"* ]]
}

@test "SD2: installed launcher diverged -> FAIL naming the install-skill.sh --force remedy, exit 1" {
  _install
  printf 'stale line\n' >> "$H/.claude/skills/build-n4-module/SKILL.md"
  run "$SD" --home "$H"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  skill-drift"* ]]
  [[ "$output" == *"install-skill.sh --force"* ]]
}

@test "SD3: launcher not installed -> FAIL naming install-skill.sh, exit 1" {
  run "$SD" --home "$H"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  skill-drift"*"not installed"* ]]
  [[ "$output" == *"install-skill.sh"* ]]
}

@test "SD4: --skill mcp-n4 checks the mcp-n4 launcher, not build-n4-module" {
  _install --skill mcp-n4
  run "$SD" --skill mcp-n4 --home "$H"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  skill-drift"*"mcp-n4"* ]]
}

@test "SD5: an unknown --skill exits 2" {
  run "$SD" --skill nope --home "$H"
  [ "$status" -eq 2 ]
}

@test "SD6: the skill table is read from install-skill.sh (single source), never duplicated" {
  run grep -c 'mcp-n4-kit/skill/SKILL.md' "$SD"
  [ "$output" = "0" ]
}

@test "SD7: check-skill-drift.sh never writes the installed copy (read-only check)" {
  _install
  printf 'stale line\n' >> "$H/.claude/skills/build-n4-module/SKILL.md"
  before=$(sha256sum "$H/.claude/skills/build-n4-module/SKILL.md")
  run "$SD" --home "$H"
  after=$(sha256sum "$H/.claude/skills/build-n4-module/SKILL.md")
  [ "$before" = "$after" ]
}

@test "SD8: no Go toolchain and no built command -> exit 3 naming go, never a PASS" {
  # A copy of the launcher beside a Go module with no bin/: the command must be built, and PATH has no go.
  _install
  mkdir -p "$TMPDIR_T/repo/scripts" "$TMPDIR_T/repo/go" "$TMPDIR_T/bin"
  cp "$SD" "$REPO/scripts/install-skill.sh" "$TMPDIR_T/repo/scripts/"
  cp "$REPO/go/go-tool.sh" "$REPO/go/go.mod" "$TMPDIR_T/repo/go/"
  ln -s "$(command -v dirname)" "$TMPDIR_T/bin/dirname"
  run env PATH="$TMPDIR_T/bin" /bin/bash "$TMPDIR_T/repo/scripts/check-skill-drift.sh" --home "$H"
  [ "$status" -eq 3 ]
  [ "$output" = "check-skill-drift: go is not on PATH; it is needed to build $TMPDIR_T/repo/go/bin/check-skill-drift" ]
}

@test "SD8b: no sha256sum and no shasum on PATH -> the check still runs (digests are computed in Go)" {
  _install
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via REPO
  . "$REPO/go/go-tool.sh"
  go_tool_bin "$REPO/go" check-skill-drift >/dev/null   # built with the full PATH: the narrowed one has no go
  mkdir -p "$TMPDIR_T/bin"
  for t in dirname find; do ln -s "$(command -v "$t")" "$TMPDIR_T/bin/$t"; done
  run env PATH="$TMPDIR_T/bin" /bin/bash "$SD" --home "$H"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  skill-drift"*"build-n4-module"* ]]
}

@test "SD9: --help prints the header only, never a code line" {
  run env HOME="$H" "$SD" --help
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | grep -c 'set -u')" -eq 0 ]
}

@test "SD10: --harness pi — a modified skill copy or prompt file is a FAIL row naming --harness pi, exit 1" {
  mkdir -p "$H/.pi/agent"
  PI_CODING_AGENT_DIR="" _install --skill mcp-n4 --harness pi
  run env -u PI_CODING_AGENT_DIR "$SD" --skill mcp-n4 --harness pi --home "$H"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS  skill-drift  mcp-n4[pi]  prompt "* ]]
  printf 'stale\n' >> "$H/.pi/agent/prompts/mcp-n4.md"
  run env -u PI_CODING_AGENT_DIR "$SD" --skill mcp-n4 --harness pi --home "$H"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  skill-drift  mcp-n4[pi]  prompt "*"differs from its template"*"--harness pi"* ]]
}

@test "SD11: --all prints a non-failing SKIP row for an absent harness; the default checks claude only" {
  _install --skill mcp-n4
  run env -u PI_CODING_AGENT_DIR "$SD" --skill mcp-n4 --all --home "$H"
  [ "$status" -eq 0 ]
  [[ "$output" == *"SKIP  skill-drift  mcp-n4[pi]  absent: $H/.pi/agent not found"* ]]
  [[ "$output" == *"SKIP  skill-drift  mcp-n4[gentle-shell]  absent: "* ]]
  run "$SD" --skill mcp-n4 --home "$H"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "$output" | wc -l)" -eq 1 ]
}

@test "SD12: an empty --home is a usage error (exit 2)" {
  run env HOME="$H" "$SD" --home ""
  [ "$status" -eq 2 ]
  [[ "$output" == *"--home requires a non-empty directory"* ]]
}
