#!/usr/bin/env bash
# count-retro-deltas.sh — counts each retro's delta rows and compares them with its stated totals (METHODOLOGY K34).
#
# A delta total is counted by script, never summed by hand: for each retro the rows `| Δ<n> |` / `| <n> |` of its delta
# section ("## Proposed kit deltas", "## Deltas" or "## Proposed delta table") are counted and compared with its
# `**Delta count**` line, its `<!-- deltas: N -->` header and its row of the generated retros/INDEX.md beside it (the
# Deltas column). A value the retro or the index does not state prints `-` and is a mismatch. Quote the TOTAL line in
# a commit message or summary that states a delta total. [ev: retro review2-signing-build-route Δ6]
#
# Implementation: go/cmd/count-retro-deltas + go/internal/retrodeltas. This wrapper is the stable entry point:
# go/go-tool.sh `go_tool_bin` builds go/bin/count-retro-deltas when it is missing or older than a Go source, then this
# script execs it. Needs `go` on PATH to build (exit 3 otherwise). VCS-free by design.
#
# Usage:  count-retro-deltas.sh <retro.md>...      (generate retros/INDEX.md first: toolbelt/kit-state.sh --write)
#   Row:  <retro>  rows=<n>  count=<n|->  header=<n|->  index=<n|->  OK|MISMATCH
#   Row:  TOTAL  rows=<sum>  files=<n>  mismatches=<n>
#   Exit: 0  every retro agrees · 1  any mismatch · 3  usage, an unreadable retro, no INDEX.md beside a retro
set -u
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'count-retro-deltas: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" count-retro-deltas)" || exit 3
exec "$BIN" "$@"
