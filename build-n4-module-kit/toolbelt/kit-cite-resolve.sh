#!/usr/bin/env bash
# kit-cite-resolve.sh — resolves every evidence pointer of kit cards (`[ev: …]` / `[deferred: …]` tags and
# prose citations: retro deltas, corpus blocks and sections, code, devguide/guide lines, memories, kit
# cross-references) and prints its status with an excerpt of the cited line, so a reviewer only judges
# whether the source says it. A pre-filter: RESOLVED means the cited place exists, never that it is true.
#
# Implementation: the Go module at <repo>/go (go/internal/citeparse, go/internal/citeresolve). This wrapper
# is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/kit-cite-resolve when the binary is
# missing or older than a Go source, then this script execs it. Needs `go` on PATH to build.
#
# Usage:  kit-cite-resolve.sh [--kit DIR] [--research DIR] [--modules DIR] [--memory DIR] [--version VER]
#                             [--format tsv|json] [--strict] <card.md>...
# Output: card line kind target locator status detail (TSV, or a JSON array that adds the scope), then on
#         stderr the per-status counts of all rows, of the LIVE rows and of the HISTORY rows, and the strict
#         findings of each scope. Findings: MISSING_FILE, MISSING_SECTION, MISSING_DELTA, LINE_OUT_OF_RANGE,
#         BARE_ID, UNPARSED, TOOL_ERROR. Not findings: RESOLVED; HISTORICAL_DRIFT (a missing or out-of-range
#         target on a history card: retros/, ledger.d/, BUILD-STATE.md, BUILD-STATE.src.md, CHANGELOG.md;
#         the original status stays in the detail); UNVERIFIABLE_KIND (evidence no tree holds: memory-tool,
#         live, VCS/tracker, command run, decision ID, self-marked); SKIPPED_TEMPLATE (a `<stem>` placeholder).
# Exits:  0 no finding, or findings without --strict · 1 any finding under --strict · 3 usage, a missing
#         or unreadable root or card, a tool error row, no Go toolchain or a failed build
# VCS-free by design (kit-links L2).
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'kit-cite-resolve: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" kit-cite-resolve)" || exit 3
exec "$BIN" "$@"
