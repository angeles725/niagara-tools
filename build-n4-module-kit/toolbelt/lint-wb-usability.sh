#!/usr/bin/env bash
# lint-wb-usability.sh — WARNs a -wb view that holds a station reference as a BOrd in a file that never opens a station-
# component picker.
#
# types/wb-widgets.md (UX rule) asks every -wb config view to offer a chooser for the components it refers to, never a
# blank-entry Add. Its lint candidate was too semantic to lint whole; this is the narrow part that reads from source.
# One WARN per class, at the class name, when all hold:
#   1. the class `extends BWbComponentView` (a qualified name counts by its last part);
#   2. the class holds a BOrd member: a non-static field of type BOrd or BOrd[], or a class-level
#      `@NiagaraProperty(... type = "BOrd")` (also "baja:Ord");
#   3. the FILE has neither a `BComponentChooser` in code (not in a comment, a string or an import line) nor a
#      `targetType` facet (an identifier, or text inside a string: the facet key).
# Waiver: a comment `lint-wb-usability: waived <reason>` on the class line or in the five lines above it. The reason is
# required, and a string literal holding the same text is not a waiver.
# [ev: retro apillm-headless-servlet-rt-4.14-deltas Δ7] [ev: types/wb-widgets.md UX rule]
#
# Left out on purpose (needs the semantic reading the 2026-09 triage deferred; a noisy lint costs the kit its
# precision): whether the user edits the BOrd or it is only held, a chooser opened by a controller in another class or
# file, drag-drop from the nav tree, the status table, a view that extends a subclass of BWbComponentView or a manager
# base, and a BOrd carried in a collection or a getter. A WARN is a question for the reviewer, not a verdict; the
# false negatives are wide on purpose. Not a report-module member: a WARN-only advisory run by hand.
#
# Implementation: rule package go/internal/lint/rules/wbusability on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-wb-usability <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-wb-usability.sh [--strict] <wb-src-root>
#   Scans *.java under <wb-src-root>, dot-directories pruned.
#   Row:  WARN  lint-wb-usability  <file>:<line>  <Class> extends BWbComponentView and holds a BOrd (<member>), but ...
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.java, an unreadable
#         sub-directory
# Mutation: WBU-import -- counting an import line as a BComponentChooser invocation clears a view that only imports it
# Mutation: WBU-static -- counting a static BOrd constant as a member WARNs a view that only names a fixed ORD
# Mutation: WBU-waiver-string -- reading the waiver from the code view too lets a string literal waive a class
# Mutation: WBU-base -- matching any base class WARNs a BWbView or a BComponent that holds a BOrd
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-wb-usability: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-wb-usability "$@"
