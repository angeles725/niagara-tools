#!/usr/bin/env bash
# lint-wb-file-chooser.sh — flags a -wb BWbFieldEditor.dialog(...BOrd.NULL) that opens the
# file-system chooser (C:\) instead of a station-component chooser.
#
# A BWbFieldEditor.dialog(parent, target, BOrd.NULL) falls back to the generic file-system
# chooser (opens C:\). In a station context this is useless — the user cannot navigate to a
# station ORD via the file dialog. The correct pattern is either a BComponentChooser (explicit
# station-component picker) or the targetType facet that narrows the dialog automatically.
# See types/wb-widgets.md "station-space picker" rule.
# [ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8]
#
# Implementation: go/internal/lint/rules/wbfilechooser on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-wb-file-chooser <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_strip_run did).
#
# Usage: lint-wb-file-chooser.sh [--strict] <wb-src-root>
#   Scans *.java under <wb-src-root>, dot-dirs pruned. Per FILE: a BWbFieldEditor.dialog( call and
#   BOrd.NULL anywhere in the file, when the same file has neither BComponentChooser nor targetType,
#   is the file-chooser shape; the row is the first dialog line that also holds BOrd.NULL, else the
#   first dialog line (an argument list split across lines). The call, BOrd.NULL and
#   BComponentChooser count in code only (never in a comment, Javadoc or string); targetType also
#   counts in a string (the facet key "targetType"), never in a comment. A `//` inside a string
#   never hides the rest of a line.
#   Row:   WARN  lint-wb-file-chooser  <file>:<line>  <trimmed source>
#   Exit:  0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env
#
# Heuristic (advisory, hence WARN not FAIL): a file that uses BComponentChooser or targetType
# is assumed to be using the correct station-space picker; a file that calls the dialog with only
# BOrd.NULL is the file-chooser candidate a reviewer must confirm.
# VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# Mutation: WFC5 -- matching the raw line (no comment/literal blanking) lets a commented BComponentChooser exempt a file
# Mutation: WFC2 -- remove the BComponentChooser/targetType-absent guard so a file WITH BComponentChooser still WARNs
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-wb-file-chooser: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-wb-file-chooser "$@"
