#!/usr/bin/env bash
# kit-resolve.sh — fail-closed resolution of the build-n4-module kit a session may run.
#
# The /build-n4-module launcher runs this first and uses its KIT line. A stale or dirty kit runs old
# gates silently (2026-10-24: a default checkout 821 commits behind origin/main passed a lexicon
# violation that origin/main fails). It lives in scripts/ because toolbelt/ never calls version control
# (kit-links L2). [ev: retro r10-stale-kit-resolution Δ1]
#
# Usage:
#   kit-resolve.sh [--refresh] [--offline-ok]
#     --refresh     accepted for compatibility, no longer needed: a clean kit-main strictly behind a
#                   freshly fetched origin/main is fast-forwarded automatically (see below).
#     --offline-ok  when `git fetch origin` fails, accept the local origin/main (named WARN, line says
#                   "unfetched"); without it an unreachable origin is a refusal.
#
# Candidates, first one holding METHODOLOGY.md wins (a stale winner is refused, never skipped):
#   1. $BUILD_N4_KIT                    — when set it must hold METHODOLOGY.md (else exit 3)
#   2. <kit-main>/build-n4-module-kit   — <kit-main> = $BUILD_N4_KIT_MAIN, default
#                                         $HOME/modulos_niagara_n4/niagara-tools-worktrees/kit-main
#   3. $HOME/modulos_niagara_n4/niagara-tools/build-n4-module-kit   (the default checkout)
#
# Success: stdout line 1 is exactly `KIT=<abs path> SHA=<short sha> (origin/main, clean)`; every other
# message goes to stderr. Refused when HEAD is behind origin/main or the work tree is dirty.
# Auto-refresh: when the winner is the kit-main worktree, its tree is clean, origin was fetched and HEAD
# is a strict ancestor of origin/main, it is checked out (detached) at origin/main and stderr says
# `kit-resolve: refreshed <old>→<new>`. $BUILD_N4_KIT and the default checkout are never touched.
# Any candidate whose HEAD is not an ancestor of origin/main is refused: `ahead` (behind = 0) or
# `diverged` (behind > 0) — the kit is origin/main, never unmerged commits.
# Exit: 0 resolved · 2 refused (stale, dirty, ahead, diverged, not a git work tree, origin unreachable) ·
#       3 usage or no candidate
#
# Implementation: the Go module at <repo>/go (go/internal/kitresolve; -h prints the header above, read from this
# file). This wrapper is the stable entry point: it is EXECUTED for its stdout line, never sourced.
# go/go-tool.sh `go_tool_bin` builds go/bin/kit-resolve (git-ignored, so the build never dirties the kit) when
# the binary is missing or older than a Go source, then this script execs it. Needs `go` on PATH to build; no Go
# toolchain or a failed build is exit 3 with a `kit-resolve: ...` line on stderr and nothing on stdout.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'kit-resolve: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" kit-resolve)" || exit 3
INSTALL_TOOLS_LAUNCHER="$here/kit-resolve.sh" exec "$BIN" "$@"
