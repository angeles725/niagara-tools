#!/usr/bin/env bash
# check-skill-drift.sh — read-only close-gate check: does the INSTALLED skill launcher match the tracked copy?
#
# An agent follows the launcher installed under <harness-dir>/skills/<skill>/SKILL.md, not the kit's
# tracked skill/SKILL.md; an installed copy rots silently (a kit change to the tracked launcher never
# reaches it). BUILD-LOOP.md §7 runs this at close; a FAIL is fixed with
# `scripts/install-skill.sh --force` (that script installs; this one never writes).
# The skill names and tracked paths are read from install-skill.sh's SKILLS table (single source).
# [ev: retro client-source-of-truth Δ3]
#
# Usage:
#   check-skill-drift.sh [--skill <name>] [--harness <claude|pi|gentle-shell|all> | --all] [--home <dir>]
#     --skill <name>    a skill from install-skill.sh's table (default build-n4-module)
#     --harness <h>     the harness whose copy is checked (default claude; harnesses as in install-skill.sh)
#     --all             every harness (the same as --harness all)
#     --home <dir>      base home directory (default $HOME; an empty value is a usage error); tests always pass it.
#                       It wins over $PI_CODING_AGENT_DIR for pi, as in install-skill.sh
#
# Row format:  PASS|FAIL|SKIP  skill-drift  <skill>  <detail>
# Without --harness the rows are the Claude copy's, as before. With --harness or --all the skill column is
# <skill>[<harness>]; pi and gentle-shell add a row for the prompt file <agent-dir>/prompts/<skill>.md
# (compared with the kit's rendered skill/prompt.md); an agent directory that does not exist is a SKIP row
# ("absent"), never a failure.
# Exit: 0 every checked copy is current · 1 any FAIL row (diverged or not installed) · 2 usage (unknown skill
#       or harness, empty --home) ·
#       3 environment (install-skill.sh, its table or the tracked launcher missing; no Go toolchain to build
#         the command)
#
# Implementation: the Go module at <repo>/go (go/internal/skillinstall; -h prints the header above, read from
# this file). This wrapper is the stable entry point (the digests are computed in Go, so no sha256 tool is
# needed). go/go-tool.sh `go_tool_bin` builds
# go/bin/check-skill-drift when the binary is missing or older than a Go source, then this script execs it.
# Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# The repository is the one holding the invoked path (never resolved through a symlink), as it always was.
GO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'check-skill-drift: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" check-skill-drift)" || exit 3
INSTALL_TOOLS_LAUNCHER="$SCRIPT_DIR/${0##*/}" exec "$BIN" "$@"
