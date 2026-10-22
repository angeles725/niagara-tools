#!/usr/bin/env bash
# gen-retro-index.sh — generates retros/INDEX.md, the retro promotion registry, from each retro's header.
#
# INDEX.md used to be hand-edited: every kit PR appended or flipped a row, so parallel sessions
# conflicted on it and "keep both sides" resolutions flipped rows. The index is now DERIVED: one row
# per retros/*.md (INDEX.md excluded) built from header comments in the retro itself. To add or flip
# a row, edit the retro header and regenerate; on a merge conflict in INDEX.md, regenerate.
# [ev: retro generated-retro-index]
#
# Header contract (column-0 comments at the top of every retro):
#   <!-- review-status: pending|folded -->   within lines 1-5 (first word after the colon)
#   <!-- module: <Module|kit> -->            within lines 1-10, exactly one line
#   <!-- deltas: <N> -->                     within lines 1-10, N a non-negative integer
#   <!-- index-comment: <text> -->           optional, within lines 1-10; trailing row comment
# Date = the first YYYY-MM-DD in the filename; rows sort by date, then filename (bytewise).
#
# Implementation: the Go module at <repo>/go (go/internal/retroindex, r8 T8j-1). This wrapper is the
# stable entry point: it builds go/bin/gen-retro-index when the binary is missing or older than a Go
# source, then execs it with this script's ../ as the default kit root. Needs `go` on PATH to build.
# Mutation pins (GRI-stale, GRI-order, GRI-comment, GRI-required) live in the Go source and tests.
#
# Usage:  gen-retro-index.sh [--check] [<kit-root>]     (default kit root: this script's ../)
# Exits:  0 written / index fresh · 1 stale or missing index under --check, or retro-header
#         violation · 3 usage/env, including no Go toolchain or a failed build (nothing written)
# VCS-free by design (kit-links L2).
set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
BIN="$GO_DIR/bin/gen-retro-index"

if [ ! -x "$BIN" ] || [ -n "$(find -H "$GO_DIR" -path "$GO_DIR/bin" -prune -o \( -name '*.go' -o -name go.mod \) -newer "$BIN" -print 2>/dev/null | head -1)" ]; then
  [ -f "$GO_DIR/go.mod" ] || { printf 'gen-retro-index: no Go module at %s\n' "$GO_DIR" >&2; exit 3; }
  command -v go >/dev/null 2>&1 || { printf 'gen-retro-index: go is not on PATH; it is needed to build %s\n' "$BIN" >&2; exit 3; }
  # Build to a private name, then rename: concurrent callers never exec a half-written binary.
  tmp="$BIN.tmp.$$"
  if ! (mkdir -p "$GO_DIR/bin" && cd "$GO_DIR" && go build -o "$tmp" ./cmd/gen-retro-index) >&2; then
    rm -f "$tmp"; printf 'gen-retro-index: go build failed in %s\n' "$GO_DIR" >&2; exit 3
  fi
  mv -f "$tmp" "$BIN"
fi
GEN_RETRO_INDEX_DEFAULT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
