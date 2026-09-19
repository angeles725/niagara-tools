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
#   Exit: 0  always (advisory) · 3  usage/env
# VCS-free by design.
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
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.gradle.kts' -print 2>/dev/null)

exit 0
