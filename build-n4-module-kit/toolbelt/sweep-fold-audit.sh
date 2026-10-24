#!/usr/bin/env bash
# sweep-fold-audit.sh — Audit folded retros for [ev: retro] citation coverage.
#
# Every folded row in INDEX.md must have at least one [ev: retro <token>]
# citation in the core kit corpus. A missing citation indicates that the
# retro's lesson was never folded into a kit file — this audit makes that
# visible so it becomes a PR7 follow-up item, not a silent gap.
#
# Usage: sweep-fold-audit.sh [--strict] [--exact-stems] [--deltas-since <YYYY-MM-DD>] <INDEX.md> <kit-root>
# exit:  0 clean or WARN-only · 1 uncited rows (or uncited Δ, or --exact-stems alias FAIL) AND --strict · 3 usage/env
#
# --deltas-since <date>: Δ-granular reconcile check. Every folded row whose filename date is on or
# after <date> must account for EACH of its Δ1..Δn (n = the deltas cell after review-status): cited
# as `[ev: retro <stem> Δk]`, or explicitly parked out of repo as `[deferred: retro <stem> Δk]`,
# somewhere in the corpus. Exact stem, exact Δ number — no segment or prefix credit. A missing one
# is `fold-audit: WARN <file> Δk not cited`; a non-numeric or oversize (>999) deltas cell is a WARN
# too (fail closed), counted on its own `fold-audit: deltas: N unreadable cells` line (shown when N>0).
# The cell parser is go/internal/retrocells ParseDeltasCell, shared with the other INDEX sweeps.
# Older rows cite by stem only (the pre-Δ convention), hence the cutoff.
#
# Corpus: every *.md under <kit-root> excluding */retros/* and INDEX.md (ledger.d/ fragments count).
# Stem: filename minus leading YYYY-MM-DD- and trailing .md. A stem token T (6-char floor) credits
# a row when "-T-" is a hyphen-segment-aligned part of "-<stem>-".
#
# Retro aliases (#419): a tag must name the exact stem (CONTRIBUTING.md:261). Every [ev:|deferred: retro T]
# whose T is not the stem of a dated INDEX row (any status) but is a segment-aligned part of one is a row
#   fold-audit: WARN <file>:<line> retro alias "<T>" — use the exact stem "<stem>" (CONTRIBUTING.md:261)
# or, matching several stems, `... is ambiguous — use one exact stem of "<a>" "<b>" ...`, then a summary
# `fold-audit: aliases: N tokens are not an exact stem (M ambiguous)`. By default the rows are advisory
# (they never change the exit code) and aliases still credit. --exact-stems makes them FAIL rows that fail
# --strict and lets only an exact stem credit a fold. A T matching no stem is not reported.
#
# A deltas cell is decimal (a leading zero is not octal); an unterminated last INDEX row is audited;
# an unreadable INDEX exits 3. (The former bash tool passed all three silently.)
#
# Implementation: the Go module at <repo>/go (go/internal/foldaudit, r8 T17), byte-identical to the
# former bash tool under C.UTF-8 except those fail-open cases. This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/sweep-fold-audit when the binary is missing or
# older than a Go source, then this script execs it. Needs `go` on PATH to build (a missing toolchain or failed build exits 3).
# Mutation pins (F3, F4, F6, F9, F11, F12, F13, F19, F20, F21, R2-unreadable) are exercised by tests/sweep-fold-audit.bats and the Go tests.
# This script is VCS-free by design — version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'sweep-fold-audit: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" sweep-fold-audit)" || exit 3
exec "$BIN" "$@"
