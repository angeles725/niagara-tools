#!/usr/bin/env bash
# check-installed-line.sh <repo> <branch> <installed-ref> — refuse a client branch that would silently downgrade a
# module group already installed: every GROUP build.gradle.kts carrying defaultModuleVersion("X") at <installed-ref>
# must exist on <branch> with a version >= X. A fix branch must start from the installed line; run this with the
# installed ref before building (BUILD-LOOP.md). It lives in scripts/ because toolbelt scripts never call version
# control (kit-links L2); the read and the compare are the shared Go package go/internal/vendorversion
# (fragment rule: one comparator for build, deploy and this guard).
# Local refs only — never fetches. Paths are read NUL-separated, so a group under a path git would quote (a
# non-ASCII directory) is compared like any other. Each `git show <ref>:<path>` is closed with `--`, so a
# working-tree entry of that name never hides a group (git otherwise answers "ambiguous argument"), and a row
# prints a path holding a control character, a double quote or a backslash quoted with C escapes (one row per
# group, one line per row). [ev: retro r10-installed-line-guard Δ1]
# [ev: retro go-check-scripts-ports Δ1]
#
# Rows: ok|DOWNGRADE|INCOMPARABLE <file> <installed> -> <branch>  ·  MISSING <file> <installed>
# Exit: 0 every group >= installed · 1 any DOWNGRADE, MISSING or INCOMPARABLE
#       2 usage, unknown ref, or no defaultModuleVersion at <installed-ref> (nothing to guard is never a pass)
#
# Implementation: the Go module at <repo>/go (go/internal/checkinstalledline). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/check-installed-line when the binary is missing or older than a
# Go source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'check-installed-line: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" check-installed-line)" || exit 3
exec "$BIN" "$@"
