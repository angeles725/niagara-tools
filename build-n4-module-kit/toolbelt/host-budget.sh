#!/usr/bin/env bash
# host-budget.sh — toolbelt entry point of go/cmd/host-budget, the shared-host gate (pids cgroups, the
# thread budget, the Go build cache, disk use) to read before heavy work (BUILD-LOOP.md § 7 Shared host).
# scripts/ci-local.sh runs the same binary before any CI step. [ev: retro r8-t47-host-budget Δ1]
#
# Implementation: the Go module at <repo>/go (go/internal/hostbudget). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/host-budget when the binary is missing or older than a
# Go source, then this script execs it. Needs `go` on PATH to build.
#
# Usage:  host-budget.sh [--cgroup-root DIR] [--proc-root DIR] [--pids-pct N] [--gocache DIR]
#                        [--gocache-gb N] [--disk-pct N] [--mount PATH]... [--strict]
# Exits:  0 every row PASS, or WARN rows without --strict · 1 a FAIL row, or a WARN row under --strict ·
#         3 usage error, an unreadable cgroup root, no Go toolchain or a failed build
# VCS-free by design (kit-links L2).
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'host-budget: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" host-budget)" || exit 3
exec "$BIN" "$@"
