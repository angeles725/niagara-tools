#!/usr/bin/env bash
# install-hooks.sh — opt-in activation of the kit's git hooks.
# Points git at the kit's hooks: `git config --local core.hooksPath .githooks`, which activates every
# hook there: the retro-enforcement pre-push gate, and post-merge / post-checkout / post-rewrite /
# post-commit, which regenerate the kit's generated state (BUILD-STATE.md, retros/INDEX.md,
# toolbelt/INDEX.md — not committed, R10-D1) with toolbelt/kit-state.sh after every pull, checkout,
# rebase/amend and commit. The install runs post-merge once
# so the state is materialized right away. The hooks ship INERT; this is what makes them live in a
# clone. Per-clone and reversible. [ev: retro ledger-generated-not-committed]
#
# Usage:
#   scripts/install-hooks.sh              # activate (set core.hooksPath = .githooks)
#   scripts/install-hooks.sh --force      # activate even over a pre-existing CUSTOM hooksPath
#   scripts/install-hooks.sh --uninstall  # deactivate (unset the core.hooksPath we set)
#
# git IS used here on purpose: this is NOT a toolbelt/ check script, so the kit-links L2 rule
# ("no toolbelt script invokes git") does not apply — the installer lives under scripts/.
#
# Exit: 0 done · 2 unknown argument or not a git repository · 3 refused (a custom core.hooksPath without --force,
#       or one that cannot be read) or no Go toolchain · else the exit status of the git command or of the
#       .githooks/post-merge that failed
#
# Implementation: the Go module at <repo>/go (go/internal/hookinstall). This wrapper is the stable entry point:
# go/go-tool.sh `go_tool_bin` builds go/bin/install-hooks when the binary is missing or older than a Go source,
# then this script execs it in the caller's working directory (the repository being configured).
# Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'install-hooks: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" install-hooks)" || exit 3
exec "$BIN" "$@"
