#!/usr/bin/env bash
# sweep-build-state.sh — CONTENT-ONLY validator for BUILD-STATE.md + retros/INDEX.md.
#
# This script is VCS-free by design. kit-links.bats L2 forbids a toolbelt/*.sh from invoking
# version control (these scripts run inside worktrees and on stations); the diff half of the
# retro gate lives in .githooks/pre-push, which calls this script for the content half.
#
# usage (content check):
#   sweep-build-state.sh <BUILD-STATE.md> <retros-dir> <INDEX.md>
#   exit:  0 clean · 1 named integrity violation · 3 usage/env
#
# usage (retro-debt aging — E5 contract):
#   sweep-build-state.sh --age --today <YYYY-MM-DD> [--max-age <N>] [--strict] <retros-dir> <INDEX.md>
#   exit:  0 (non-strict, always); with --strict, 1 when escalated_count > 0 (the same report on stdout, one
#          `--strict: <n> pending retro(s) older than <N> days` line on stderr) · 3 usage/env
#   stdout (three lines):
#     total_pending=<n>
#     escalated_count=<n>
#     oldest_age=<n|N/A>
#   Only pending rows age; folded rows and 0-delta pending rows whose INDEX line carries a
#   `<!-- disposition: tracked-by-issue #<n> … -->` or `<!-- disposition: worklist-complete … -->`
#   comment (a parked non-retro doc) are excluded; any other disposition still ages.  Age is derived from the retro's FILENAME
#   date prefix (YYYY-MM-DD-…), never from the file body.  Today is INJECTED via --today.
#
# content checks:
#   - build-state.v1 markers are balanced, anchored at column 0 (a marker quoted in prose,
#     inside backticks or indented, is NOT an envelope);
#   - each envelope carries module + retro_required + retro_pending, both booleans;
#   - the open_issues: multi-line list is tolerated (list items are not fields);
#   - WARN (stderr, exit unchanged): a module envelope other than `module: kit` with no or an empty
#     module_root / source_of_truth (`unknown` is a value) — both are DECLARED for every module (audit A4);
#   - ledger evidence tags: in a kit with ledger.d/, an `[ev:` or `[deferred:` tag of a fragment left open (no
#     closing `]` before the next tag or the end of the text) FAILs as `ledger.d/<file>:<line>` — it would
#     credit a fold it does not name (audit A4; read with go/internal/citeparse, so backticked code is no tag);
#   - INDEX integrity: every retro file has a row, every row points at a real retro, and every
#     row's review-status is pending or folded;
#   - freshness: BUILD-STATE.md and retros/INDEX.md are generated and not committed, so callers
#     (CI, .githooks/pre-push) pass fresh `toolbelt/kit-state.sh build-state|retro-index` output from
#     a temp dir. A GENERATED file (line 1 = the kit-state GENERATED comment) must equal what the kit
#     at <retros-dir>/.. generates now (BUILD-STATE.src.md + ledger.d/, the retro headers); a
#     hand-written file (test fixtures) skips its check, but a kit with a ledger.d/ takes only the
#     generated BUILD-STATE.md. Both run in-process (go/internal/retroindex, go/internal/buildstate)
#     with the generators' own messages on stderr; a failing check is exit 1.
#
# Implementation: the Go module at <repo>/go (go/internal/buildstatefields). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/sweep-build-state when the binary is missing or
# older than a Go source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a
# failed build is exit 3. [ev: retro go-sweep-build-state]
# Mutation pins (GRI-sweep, GBS-sweep, D8, D9, BSF-pending, LT1-unterminated, LT2-nested, SOT1-warn,
# AGE-strict) live in the Go source and tests.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'sweep-build-state: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" sweep-build-state)" || exit 3
exec "$BIN" "$@"
