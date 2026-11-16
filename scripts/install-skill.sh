#!/usr/bin/env bash
# install-skill.sh — install a skill launcher into an agent harness (Claude Code, pi, gentle-shell).
#
# Copies the tracked canonical launcher into <harness-dir>/skills/<skill>/SKILL.md
# using a sha256 comparison (computed by the Go command, no sha256 tool needed) to detect divergence:
#   --skill build-n4-module (default): build-n4-module-kit/skill/SKILL.md
#   --skill mcp-n4:                    mcp-n4-kit/skill/SKILL.md
# (the SKILLS table below is the single source of these names and paths)
#
# Harnesses (--harness; the table is go/internal/skillinstall/harness.go):
#   claude (default)  <home>/.claude/skills/<skill>/SKILL.md (the directory is created)
#   pi                <agent-dir>/skills/<skill>/SKILL.md and <agent-dir>/prompts/<skill>.md, where
#                     <agent-dir> is <home>/.pi/agent when --home is given, else $PI_CODING_AGENT_DIR
#                     when set, else $HOME/.pi/agent (an explicit --home always wins)
#   gentle-shell      the same two files under <home>/.gentle-shell/agent
#   all               every harness above, in that order
# A pi or gentle-shell agent directory that does not exist is never created: the run prints an
# "absent: <harness>" row and goes on (exit 0). The prompt file gives the harness a bare /<skill>; it is
# rendered from the kit's skill/prompt.md beside the tracked launcher (@SKILL_FILE@ becomes the absolute
# path of the installed SKILL.md), or from the Go package's default template when the kit has none, and follows the same
# current / diverged / --force rules as the launcher. Every file of every selected harness is checked
# before the first write: a diverged copy without --force (exit 1), a symbolic link on the way (exit 3), or
# a file that cannot be placed (a non-directory where its directory must be, or a target that is not a
# regular file; exit 3) stops the run with nothing written.
#
# --skill build-n4-module also MATERIALIZES the kit's generated state in the kit checkout the launcher
# reads: BUILD-STATE.md, retros/INDEX.md and toolbelt/INDEX.md are generated and not committed (R10-D1), so a fresh install
# runs `<kit>/toolbelt/kit-state.sh --write` (on every run, also when the launcher is already current).
# The kit is $BUILD_N4_KIT when it holds METHODOLOGY.md (the launcher's own first rule), else this
# repository's build-n4-module-kit/. [ev: retro ledger-generated-not-committed]
#
# This script is VCS-free by design — version control is never invoked here.
# The source of truth lives in the repository; this script only installs it.
#
# Usage:
#   install-skill.sh [--skill <build-n4-module|mcp-n4>] [--harness <claude|pi|gentle-shell|all>] [--home <dir>]
#                    [--dry-run] [--force]
#
# Options:
#   --skill <name> Skill to install: build-n4-module (default) or mcp-n4.
#   --harness <h>  Harness to install into: claude (default), pi, gentle-shell or all.
#   --home <dir>   Base home directory (default: $HOME); an empty value is a usage error. Every test
#                  passes this flag so no test ever touches the real $HOME.
#   --dry-run      Print what would be done; write nothing. Exits 0.
#   --force        Overwrite a diverged installed copy without exiting 1. A target SKILL.md or prompt
#                  file, the skill directory, or the skills/ or prompts/ directory that is a symbolic
#                  link is refused (exit 3), never written through: remove the link, then re-run.
#
# Exit codes:
#   0   Installed or already current (or --dry-run); for build-n4-module, kit state materialized
#       (a kit without toolbelt/kit-state.sh is a warning, not an error).
#   1   Installed copy exists and diverges from the tracked copy; --force absent.
#   2   Usage error (including an unknown --skill or --harness name and an empty --home).
#   3   Environment error (tracked source missing, a prompt template that cannot be read, a symlinked target, a
#       non-directory on the target's path or a target that is not a regular file, no Go toolchain to build the command, target directory not
#       creatable, kit-state failed: the launcher is installed but the kit state is not).
#
# Implementation: the Go module at <repo>/go (go/internal/skillinstall). This wrapper is the stable entry point
# and keeps one thing: the SKILLS table below (data: Go reads that line from this file, for this script and for
# check-skill-drift.sh). go/go-tool.sh `go_tool_bin` builds go/bin/install-skill when the binary is missing or older
# than a Go source, then this script execs it.
# Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# The one place that names the skills: "<name>:<tracked launcher, relative to the repo>".
# The usage text, the error messages and the lookup are all derived from it (go/internal/skillinstall/table.go).
# shellcheck disable=SC2034  # data, read from this file by the Go command
SKILLS="build-n4-module:build-n4-module-kit/skill/SKILL.md mcp-n4:mcp-n4-kit/skill/SKILL.md"

# The repository is the one holding the invoked path (never resolved through a symlink), as it always was.
GO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'install-skill: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" install-skill)" || exit 3
INSTALL_TOOLS_LAUNCHER="$SCRIPT_DIR/${0##*/}" exec "$BIN" "$@"
