#!/usr/bin/env bash
# lint-wb-external-ord-value.sh — flags a -wb manager/view method that resolves a per-row
# ORD to a target OUTSIDE the view's subscribed subtree, reads that target's live
# value/status, but never leases a refresh for that specific external target.
#
# Shape (Δ3, corpus B1140-G1): subject-subtree `registerForComponentEvents(subject, depth)`
# only ever reaches DESCENDANTS of `subject`. A manager column that resolves a per-row
# `BOrd` property to an ARBITRARY external station component (e.g.
# `BApillmImportMap.targetOrd` -> any point outside the importer subtree, or
# `BApillmExport.ord` -> any export target) and then reads that resolved target's live
# value/status is reading a component the subject-subtree subscription never covers.
# It stays at its resolve-time snapshot (or shows "unresolved" on the first, async,
# resolve) no matter how the subject-subtree depth is tuned. Fix: for each resolved
# external target, call `target.loadSlots()` once (forces the synchronous first read)
# and `registerForComponentEvents(target, 0)` (leases future refreshes for that target
# specifically) -- guard repeat resolves with `isRegisteredForComponentEvents(target)`.
# See types/wb-widgets.md "view refresh rules" (Δ3 bullet) for the full rule and the
# depth rule it pairs with (Δ2).
# [ev: retro apillm-wb-subscription-refresh-and-points-deltas Δ3]
# [ev: corpus B1140-G1; Apillm BApillmImporterManager.java rowFor()]
#
# Implementation: go/internal/lint/rules/extordvalue on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-wb-external-ord-value <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_run did).
#
# Usage:  lint-wb-external-ord-value.sh [--strict] <wb-src-root>
#   Scans *.java under <wb-src-root> (dot and *-worktrees dirs pruned). Comments are stripped and
#   method boundaries are found with the shared PEAK-depth parser (go/internal/javasrc Strip/Methods,
#   the port of toolbelt/lib/method-boundary.sh mb_strip/mb_parse -- never re-implement a
#   parallel parser in a rule). Per METHOD:
#     1. a candidate ORD-resolve line: `<ident>Ord()<opt-space>.get(` (e.g.
#        `getTargetOrd().get(`) or `<ident>Ord<opt-space>.get(` / a bare `ord.get(` (e.g.
#        `folderOrd.get(`; `record.get(` is not an ORD);
#     2. AND, anywhere in the SAME method, a live-value/status read off the resolved
#        object: `.get("out")`, `getOutStatusValue(`, or `BStatusValue`;
#     3. AND the SAME method has NO `registerForComponentEvents(` call anywhere in its
#        body.
#   All three true -> WARN at the resolve line.
#   Row:  WARN  lint-wb-external-ord-value  <file>:<line>  <method>() resolves an external
#         ORD and reads its value/status with no registerForComponentEvents(target,0)
#         lease in scope -- subject-subtree depth does not cover it; add loadSlots() +
#         registerForComponentEvents(target,0) for the resolved target
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-wb-external-ord-value  <root>  no *.java sources found (wrong path or empty source tree)` (go/internal/lint
# Spec.EmptyError; declared difference 2026-10-23: the bash lint exited 0 with no row).
#
# Known limitations (advisory heuristic, documented per kit style):
#   - a resolve line and its value-read split across TWO methods (helper extraction) is
#     a false NEGATIVE -- this lint scans one method's body at a time.
#   - a `registerForComponentEvents(` call ANYWHERE in the same method suppresses the
#     WARN even if it targets an unrelated component -- this lint does not track which
#     BComponent variable is passed to which call (a rare false NEGATIVE, not a
#     false-positive risk).
#   - an ORD resolved purely to navigate a CONTAINER (e.g. `folderOrd.get(...)` used only
#     to find a folder to add a new point into) is exempted naturally: it only WARNs when
#     the SAME method also reads a live value/status off the resolved object.
# VCS-free by design (kit-links L2).
# Mutation: WEO2 -- drop the registerForComponentEvents(-absence check so a compliant method still WARNs
# Mutation: WEO-record -- an unanchored [Oo]rd suffix takes record.get( for an ORD resolve
# Mutation: WEO-awkfail -- an unreadable source file read as empty (internal/lint Run) reports it clean
# Mutation: WEO-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-wb-external-ord-value: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-wb-external-ord-value "$@"
