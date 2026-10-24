#!/usr/bin/env bash
# lint-arbitrary-ord.sh — flags BOrd.make() resolved from a non-literal (client-supplied) value.
#
# BOrd.make(someVariable) that comes from client input lets a caller resolve ANY station ORD
# (e.g. a foreign service, a credential store). Validate/allowlist a client-supplied ORD prefix
# before resolving. See types/security.md §4. Advisory (WARN), FP-prone: many BOrd.make(var)
# calls take a trusted internal value. [ev: retro 2026-09-19-security-model]
#
# Implementation: go/internal/lint/rules/arbitraryord on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-arbitrary-ord <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise).
# The marker comment is found by javasrc.LineCommentAt (string-, char- and inline-block-comment-aware). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_each did).
#
# Usage:  lint-arbitrary-ord.sh <src-root>
#   Flags BOrd.make(<lowercase-identifier>) — skips string literals and UPPER_CASE constants.
#   Row:  WARN  lint-arbitrary-ord  <file>:<line>  BOrd.make from a variable: <source>
#   Reviewed call site: `// lint-arbitrary-ord: reviewed <reason>` on the same line or the line
#   directly above suppresses that one WARN; the reason is mandatory — a bare marker still WARNs
#   ("reviewed marker without a reason"). [ev: retro panccadia-restart-seq-comp-lockout-hours Δ4]
#   The marker counts only in a real // comment: a "//" inside a string or char literal is code.
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: AO2 -- require a leading quote and AO2 stops warning on BOrd.make(variable)
# Mutation: AO4 -- ignore the reviewed marker and AO4 WARNs again on the reviewed call site
# Mutation: AO6 -- accept a bare marker without a reason and AO6 stops warning
# Mutation: AO8 -- split code/comment at the first // regardless of quotes and AO8/AO9 stop warning
# Mutation: AO12 -- no block-comment skip in code_part and the apostrophe in /* it's */ hides the marker
# Mutation: AO-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: AO-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
# Mutation: AO13 -- matching the raw line (no comment/literal blanking) WARNs on a Javadoc or string mention
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-arbitrary-ord: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-arbitrary-ord "$@"
