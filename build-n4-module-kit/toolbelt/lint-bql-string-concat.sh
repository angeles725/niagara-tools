#!/usr/bin/env bash
# lint-bql-string-concat.sh — flags a BQL query built by string concatenation.
#
# BQL has no parameterized-query API. Embedding a value into a `bql:` ORD or a
# BqlQuery.make(...) by string concatenation (`"...where x = '" + v + "'"`) is an injection
# risk. Embed typed values with BqlQuery.toBqlLiteral(BSimple) + SlotPath.escape(...) instead.
# See types/security.md §4. Advisory (WARN): a concat with a constant is usually fine.
# [ev: retro 2026-09-19-security-model]
#
# Implementation: go/internal/lint/rules/bqlconcat on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-bql-string-concat <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_each did).
#
# Usage:  lint-bql-string-concat.sh <src-root>
#   Per line: a `bql:` text (in a literal) or a BqlQuery.make( call, and a `+` next to a quote, in the
#   code only: comments and Javadoc never trigger, a quoted "BqlQuery.make(" is not a call, and a `//`
#   inside a string (an ORD such as "foxs://host") never hides the rest of the line.
#   A line whose every `+` joins only literals, numbers, true/false and simple names declared with a
#   numeric type (byte/short/int/long/float/double/boolean or the boxed type, never char) in the same
#   method or its parameters is clean: such a value cannot carry a quote. A field, a call, a field access,
#   an array element, a cast, a parenthesized expression or a name also declared with another type stays a
#   row: the rule cannot prove its type. [ev: issue #459]
#   Row:  WARN  lint-bql-string-concat  <file>:<line>  bql concat: <source>
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: BSC2 -- drop the concat-operator test and BSC2 stops warning on a concatenated bql: string
# Mutation: BSC-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: BSC4 -- matching the raw line (no comment/literal blanking) WARNs a commented or quoted BqlQuery.make concat
# Mutation: BSC5 -- dropping the numeric-operand check WARNs a long/int-only bql: concat again
# Mutation: BSC-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-bql-string-concat: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-bql-string-concat "$@"
