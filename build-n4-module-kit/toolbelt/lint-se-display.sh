#!/usr/bin/env bash
# lint-se-display.sh — a -se part must not use display-dependent AWT/Swing classes.
#
# The -se profile loads on the headless station daemon (JACE launches -rp:rt,se). A
# display class (JFrame/JDialog/JWindow, java.awt.Frame/Window) instantiates a GUI and
# fails at runtime on a headless controller — with NO build-time signal. Headless-safe
# AWT (java.awt.print, java.awt.Font, java.awt.image) is fine. See types/structure.md §-se.
# [ev: retro 2026-09-19-se-profile]
#
# Implementation: go/internal/lint/rules/sedisplay on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-se-display <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_each did).
#
# Usage:  lint-se-display.sh <se-src-root>     (a -se profile source tree)
#   Row:  FAIL  lint-se-display  <file>:<line>  display class: <source>
#   Exit: 0  clean · 1  any FAIL · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: SED2 -- drop the JFrame import case and SED2 stops flagging a display import
# Mutation: SED-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: SED-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-se-display: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-se-display "$@"
