#!/usr/bin/env bash
# orient-guard.sh — BUILD-LOOP §0.a location guard for Niagara N4 module builds.
#
# Asserts that <module-root> is under the sole legal build prefix
# $HOME/modulos_niagara_n4/Cliente/ before any build or file-editing action.
# Paths are canonicalized (symlinks + relative components resolved) before compare.
#
# Usage:  orient-guard.sh <module-root>
#
# Environment:
#   BUILD_N4_LEGAL_ROOT_PREFIX   Override the legal prefix (test seam — ALWAYS emits WARN).
#   BUILD_N4_CLIENTE_OVERRIDE    Set to exactly '1' to warn-and-pass on a non-legal path.
#                                Any other non-empty value is treated as unset (fail-closed).
#
# Row format:  PASS|FAIL|WARN  orient-guard  <detail>
# Exit:        0 (PASS or WARN) · 1 (FAIL) · 3 (usage error, or no HOME to derive the default prefix)
#
# This script is VCS-free by design. Version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
#
# Mutation: OG1 -- moves prefix compare to allow any path, making outside-prefix return PASS
# Mutation: OG2 -- removes the trailing-slash append before compare (ClienteX/ false-positive)
# Mutation: OG3 -- treats BUILD_N4_CLIENTE_OVERRIDE=0 as =1 (non-1 override bypass)
# Mutation: OG4 -- skips canonicalization, comparing raw input instead of realpath
# Mutation: OG10 -- a hardcoded home directory as the default prefix (fails under a temp HOME)
#
# Truth fix of the Go port: a path that cannot be resolved (a missing directory before the last component,
# a path through a file, a link loop) is still compared as given, but it no longer PASSes on its text alone:
# what does resolve of it must stay under the prefix too. The bash tool PASSed <prefix>/New/../../elsewhere.
# The path is always resolved by the tool itself (realpath on PATH is not needed).
#
# Implementation: the Go module at <repo>/go (go/cmd/orient-guard, go/internal/orientguard). This script is
# only the launcher: go/go-tool.sh `go_tool_bin` builds go/bin/orient-guard when the binary is missing or older
# than a Go source, then this script execs it. No Go toolchain or a failed build is exit 3.
set -u

# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'orient-guard: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" orient-guard)" || exit 3
exec "$BIN" "$@"
