#!/usr/bin/env bash
# lint-agent-on-shape.sh — every module-include.xml <on type="..."/> must be module:Type shaped.
#
# An agent-on binding whose target omits the module prefix (e.g. <on type="ReflowService"/>
# instead of <on type="nmodsreflow:ReflowService"/>), or is otherwise malformed, makes the
# agent silently fail to appear in Workbench — no build error. The value must be exactly
# `module:Type` (one colon, non-empty parts). See types/module-wiring.md §3.
# [ev: retro 2026-09-19-module-wiring]
#
# Usage:  lint-agent-on-shape.sh <src-root>     (finds module-include.xml under it)
#   Row:  FAIL  lint-agent-on-shape  <file>:<line>  malformed agent-on target (need module:Type): <value>
#   Exit: 0  clean · 1  any FAIL · 3  usage/env
# VCS-free by design.
# Mutation: AOS2 -- relax the module:Type regex and AOS2 stops failing on a missing module prefix
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-agent-on-shape.sh <src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-agent-on-shape: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

fail=0
while IFS= read -r f; do
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        # only lines that declare an <on ... type="...">
        printf '%s' "$line" | grep -Eq '<on[[:space:]][^>]*type=' || continue
        val=$(printf '%s' "$line" | sed -nE 's/.*<on[[:space:]][^>]*type="([^"]*)".*/\1/p')
        [ -n "$val" ] || continue
        # well-formed = exactly one colon, non-empty module and type parts
        if ! printf '%s' "$val" | grep -Eq '^[A-Za-z0-9_.-]+:[A-Za-z0-9_$]+$'; then
            printf 'FAIL  lint-agent-on-shape  %s:%s  malformed agent-on target (need module:Type): %s\n' "$f" "$ln" "$val"
            fail=1
        fi
    done < "$f"
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name 'module-include.xml' -print 2>/dev/null)

exit "$fail"
