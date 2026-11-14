#!/usr/bin/env bash
# install-client-hooks.sh — install the kit's split-package pre-push gate into a CLIENT repository (issue #142).
#
# Copies build-n4-module-kit/templates/client-pre-push to <client-repo>/.githooks/pre-push, sets that clone's
# `core.hooksPath = .githooks` and records this kit's toolbelt in the clone's LOCAL config
# (`n4kit.toolbelt`, never committed). Commit .githooks/pre-push in the client repo so every clone carries it;
# each new clone runs this installer once to activate it.
#
# Usage:  scripts/install-client-hooks.sh [--force] <client-repo>
#   --force  replace a DIFFERENT existing .githooks/pre-push and a custom core.hooksPath
# Exit: 0 installed (or already installed: idempotent) · 2 usage, <client-repo> is not a git work tree, or the kit
#       template is missing · 3 refused, nothing changed (re-run with --force): a different pre-push exists, a
#       custom core.hooksPath is set (local, global or system), or the default hooks directory holds an active
#       hook that setting .githooks would silently stop
#
# git IS used here on purpose: this lives under scripts/, not toolbelt/ (kit-links L2).
#
# Implementation: the Go module at <repo>/go (go/internal/hookinstall). This wrapper is the stable entry point:
# go/go-tool.sh `go_tool_bin` builds go/bin/install-client-hooks when the binary is missing or older than a Go
# source, then this script execs it. The kit recorded in the client clone is the checkout holding the invoked
# path (never resolved through a symlink), as it always was. Needs `go` on PATH to build; no Go toolchain or a
# failed build is exit 3. Anything that cannot be checked (the hooks path, the default hooks directory, the
# existing hook) is exit 3 with nothing changed; a file operation that fails is exit 1 and a git command that
# fails ends with git's own status.
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'install-client-hooks: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" install-client-hooks)" || exit 3
INSTALL_TOOLS_LAUNCHER="$SCRIPT_DIR/install-client-hooks.sh" exec "$BIN" "$@"
