#!/usr/bin/env bash
# lint-wb-threading.sh — Workbench Swing-thread and agent-breadth checks (Campaign 8 PR11).
#
# Two checks over a -wb src tree (B809 §809.7):
#   ui-thread-traversal  WARN  a doInvoke body (or any same-class private/protected
#                               method reachable within 3 call levels) calls
#                               getNavChildren|getNavNodes|BqlQuery without an
#                               invokeLater|BJobService|JobThread anywhere on the
#                               expanded chain.  Bodies are extracted by brace-counting;
#                               the expansion is cycle-safe (visited set).
#                               Heuristic only: WARN for human review (B809).
#   agent-breadth        WARN  @AgentOn(types="baja:Component") with no comment
#                               containing 'justif', 'why', or 'broad' within 3 lines above.
#
# --strict turns any WARN into exit 1 (useful for CI gates).
#
# Usage: lint-wb-threading.sh [--strict] <wb-src-dir>
#
#   Row format: <check>  WARN  <file>:<line>  <detail>
#   Exits: 0 clean or WARN-only (no --strict) · 1 any WARN under --strict · 3 usage/env (K20)
#
# Dot-directories (e.g. .deploy-baseline/) are excluded per D9b.
# Empty root: a root with no *.java is an env error (exit 3) with
# `ERROR  lint-wb-threading  <root>  no files to scan (K20: wrong path or empty scaffold)` on stdout.
# Implementation: rule package go/internal/lint/rules/wbthreading on the shared engine go/internal/lint
# (walker, Mode Anywhere parser, check-first rows via Spec.CheckFirst; go/README.md). The call chain a row
# names follows gawk's `for (k in a)` order of the callee set (rules/awkorder), as the awk printed it. This
# wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or
# older than a Go source, then this script execs `n4lint lint-wb-threading <args>`. Needs `go` on PATH.
# VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# [ev: retro campaign8-wb-audit]
# Truth fix 2026-10-24: both checks read code, not raw text. A nav call or a guard named only in a comment or
# a string no longer counts, a brace in a string no longer cuts a body short, a commented-out @AgentOn is not
# a row, and the justification must be in a comment (a `whyNot` variable does not justify).
# Truth fix 2026-10-24: getNavChildren|getNavNodes|BqlQuery and invokeLater|BJobService|JobThread match whole
# words (BBqlQueryBuilder, a UI dialog, is not a BqlQuery; invokeLaterLike() is not a guard), and the row names the
# nav call found (the bash always printed getNavChildren). sources/decompiled 1 -> 0 rows, Cliente unchanged.
# Mutation: WBT1c -- changes the invokeLater exemption, making an invokeLater-wrapped traversal false-WARN
# Mutation: WBT-words -- matching substrings flags new BBqlQueryBuilder(...) in a doInvoke as a BqlQuery traversal
# Mutation: WBT-code -- reading raw text passes a doInvoke whose only guard is a comment and flags a commented-out @AgentOn
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-wb-threading: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-wb-threading "$@"
