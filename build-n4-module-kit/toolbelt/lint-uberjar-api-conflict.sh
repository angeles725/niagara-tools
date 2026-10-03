#!/usr/bin/env bash
# lint-uberjar-api-conflict.sh — a library must not be BOTH uberjar()'d and api()'d.
#
# Embedding a library with uberjar("group:artifact:ver") AND also declaring it as a compile
# dep with api("group:artifact:ver") puts two copies of the same classes in different
# classloader tiers — a conflict. Pick one: bundle it (uberjar) OR depend on a module that
# provides it (api). See types/module-wiring.md §1. Advisory (WARN).
# [ev: retro 2026-09-19-module-wiring]
#
# Usage:  lint-uberjar-api-conflict.sh <src-root>     (finds *.gradle.kts under it)
#   Row:  WARN  lint-uberjar-api-conflict  <file>  <group:artifact> is both uberjar()'d and api()'d
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: UAC2 -- skip the intersection and UAC2 stops warning on a lib both uberjar'd and api'd
# Mutation: UAC-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-uberjar-api-conflict.sh <src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-uberjar-api-conflict: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/scan-files.sh"
_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
# A sub-directory find cannot enter would be skipped silently: env error, never a clean pass.
had_err=0
if ! scan_files "$_TMP/files" "$_TMP/find.err" "$ROOT" -name '*.gradle.kts'; then
    printf 'lint-uberjar-api-conflict: cannot list every file under %s: %s\n' "$ROOT" "$(head -n 1 "$_TMP/find.err")" >&2
    had_err=1
fi

# extract group:artifact (drop :version) from a config line
coords() {
    # $1 = config keyword (uberjar|api|implementation)
    grep -Eo "$1\\([\"']([A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+)(:[^\"']*)?[\"']\\)" "$f" 2>/dev/null \
        | sed -nE "s/.*[\"']([A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+)(:[^\"']*)?[\"'].*/\\1/p" | sort -u
}

while IFS= read -r f; do
    ub=$(coords uberjar)
    [ -n "$ub" ] || continue
    apis=$( { coords api; coords implementation; } | sort -u )
    [ -n "$apis" ] || continue
    dup=$(comm -12 <(printf '%s\n' "$ub") <(printf '%s\n' "$apis") 2>/dev/null)
    if [ -n "$dup" ]; then
        while IFS= read -r ga; do
            [ -n "$ga" ] || continue
            printf 'WARN  lint-uberjar-api-conflict  %s  %s is both uberjar()d and api()d\n' "$f" "$ga"
        done <<< "$dup"
    fi
done < "$_TMP/files"

[ "$had_err" -eq 0 ] || exit 3
exit 0
