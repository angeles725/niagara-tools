#!/usr/bin/env bash
# check-skill-drift.sh — read-only close-gate check: does the INSTALLED skill launcher match the tracked copy?
#
# An agent follows the launcher installed under <home>/.claude/skills/<skill>/SKILL.md, not the kit's
# tracked skill/SKILL.md; an installed copy rots silently (a kit change to the tracked launcher never
# reaches it). BUILD-LOOP.md §7 runs this at close; a FAIL is fixed with
# `scripts/install-skill.sh --force` (that script installs; this one never writes).
# The skill names and tracked paths are read from install-skill.sh's SKILLS table (single source).
# [ev: retro client-source-of-truth Δ3]
#
# Usage:
#   check-skill-drift.sh [--skill <name>] [--home <dir>]
#     --skill <name>  a skill from install-skill.sh's table (default build-n4-module)
#     --home <dir>    base home directory (default $HOME); tests always pass it
#
# Row format:  PASS|FAIL  skill-drift  <skill>  <detail>
# Exit: 0 installed copy is current · 1 diverged or not installed · 2 usage (unknown skill) ·
#       3 environment (install-skill.sh, its table or the tracked launcher missing; no sha256 tool)
#
# Implementation: the Go module at <repo>/go (go/internal/skillinstall; -h prints the header above, read from
# this file). This wrapper is the stable entry point and keeps the up-front sha256 tool check of the contract
# (exit 3 before anything else; the digests themselves are computed in Go). go/go-tool.sh `go_tool_bin` builds
# go/bin/check-skill-drift when the binary is missing or older than a Go source, then this script execs it.
# Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via SCRIPT_DIR
. "$SCRIPT_DIR/lib/sha256.sh" || exit 3
sha256_require 'check-skill-drift' || exit 3

# The repository is the one holding the invoked path (never resolved through a symlink), as it always was.
GO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'check-skill-drift: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" check-skill-drift)" || exit 3
INSTALL_TOOLS_LAUNCHER="$SCRIPT_DIR/${0##*/}" exec "$BIN" "$@"
