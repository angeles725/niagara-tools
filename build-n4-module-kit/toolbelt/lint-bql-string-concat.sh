#!/usr/bin/env bash
# lint-bql-string-concat.sh — flags a BQL query built by string concatenation.
#
# BQL has no parameterized-query API. Embedding a value into a `bql:` ORD or a
# BqlQuery.make(...) by string concatenation (`"...where x = '" + v + "'"`) is an injection
# risk. Embed typed values with BqlQuery.toBqlLiteral(BSimple) + SlotPath.escape(...) instead.
# See types/security.md §4. Advisory (WARN): a concat with a constant is usually fine.
# [ev: retro 2026-09-19-security-model]
#
# Usage:  lint-bql-string-concat.sh <src-root>
#   Row:  WARN  lint-bql-string-concat  <file>:<line>  bql concat: <source>
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: BSC2 -- drop the concat-operator test and BSC2 stops warning on a concatenated bql: string
# Mutation: BSC-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: BSC-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-bql-string-concat raw 'usage: lint-bql-string-concat.sh <src-root>' "$@"
awk_lint_list -name '*.java'

# One awk per file (was two printf|grep pipelines per source line). Line // comments are stripped first.
# A bql: string OR a BqlQuery.make, AND a string-concatenation operator next to a quote.
# shellcheck disable=SC2016  # a literal awk program: $0, $n are awk fields
awk_lint_each '
    { code = $0; i = index(code, "//"); if (i) code = substr(code, 1, i - 1) }
    code ~ /bql:|BqlQuery\.make\(/ && code ~ /"[[:space:]]*\+|\+[[:space:]]*"/ {
        t = $0; sub(/^[[:space:]]+/, "", t)
        printf "WARN  lint-bql-string-concat  %s:%d  bql concat: %s\n", ENVIRON["AL_FILE"], FNR, t
    }'

awk_lint_exit 0
