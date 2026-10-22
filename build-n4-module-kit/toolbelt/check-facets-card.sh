#!/usr/bin/env bash
# check-facets-card.sh — diffs the BFacets key reference card (types/logic-authoring.md) against the BFacets
# key constants, in declaration order.
#
# The card was hand-copied once and drifted (makeDouble, 17 keys that are not constants). Pass the 4.15.x
# source (BFacets.java: every `public static final String NAME = "key";`) or a keys file (`NAME=key` lines,
# `#` comments allowed; the kit pins tests/fixtures/facets-card/BFacets-4.15.6.20.keys).
# Card rows are `| \`NAME\` | \`"key"\` | ...` lines from the `## BFacets key reference card` heading to the
# first `**Not` line or the next `## ` heading.
# [ev: retro devguide415-runtime-api-corrections Δ8]
#
# Implementation: the Go module at <repo>/go (go/internal/facetscard). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/check-facets-card when the binary is missing or older
# than a Go source, then this script execs it. Needs `go` on PATH to build. [ev: retro go-check-facets-card]
# Mutation pins (FC-order, FC-empty) live in the Go source and tests.
#
# Usage:  check-facets-card.sh <card.md> <BFacets.java | keys-file>
# Output: one line per difference — `MISSING NAME=key` (in the source, not the card), `EXTRA NAME=key`
#         (in the card, not the source), `ORDER card #n NAME=key != source NAME=key` — then a summary line.
# Exit:   0 match · 1 any difference · 3 usage/env (unreadable file, no keys parsed on either side,
#         or no Go toolchain / a failed build)
# VCS-free by design (kit-links L2).
set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'check-facets-card: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" check-facets-card)" || exit 3
exec "$BIN" "$@"
