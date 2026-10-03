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
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-bql-string-concat.sh <src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-bql-string-concat: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
had_err=0
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.java'; then
    printf 'lint-bql-string-concat: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
    had_err=1
fi

while IFS= read -r f; do
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=${line%%//*}
        # a bql: string OR a BqlQuery.make, AND a string-concatenation operator next to a quote
        if printf '%s' "$code" | grep -Eq 'bql:|BqlQuery\.make\(' \
           && printf '%s' "$code" | grep -Eq '"[[:space:]]*\+|\+[[:space:]]*"'; then
            trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
            printf 'WARN  lint-bql-string-concat  %s:%s  bql concat: %s\n' "$f" "$ln" "$trimmed"
        fi
    done < "$f"
done < "$_TMP/files"

[ "$had_err" -eq 0 ] || exit 3
exit 0
