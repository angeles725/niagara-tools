#!/usr/bin/env bash
# lint-lexicon-ascii.sh — ASCII-purity lint for *.lexicon files under a module root.
#
# Niagara reads lexicon files as Latin-1 (ISO-8859-1), then serves the display
# strings to browsers as UTF-8.  Any non-ASCII byte — e.g. the UTF-8 encoding of
# "Presión" — arrives on-station as mojibake ("PresiÃ³n"), silently breaking labels.
# All lexicon content must use only ASCII (U+0000–U+007F); accented characters
# must be represented as Unicode escapes (\uNNNN) per the Java .properties spec.
#
# Real defect trigger: UmbrellaDashboard-rt/module.lexicon values for "Presion"
# were previously authored as "Presión" (UTF-8); this lint would have caught it.
#
# Usage:  lint-lexicon-ascii.sh <module-root>
#
#   Finds every *.lexicon under <module-root>, dot-dirs pruned (D9b).
#   Prints one row per non-ASCII occurrence:
#     FAIL  lexicon-ascii  <file>:<line>  non-ASCII: <hex bytes>
#   Exits: 0  no FAIL (clean) · 1  any FAIL · 3  usage/env
#
# Row format: FAIL  lint-lexicon-ascii  <path>:<line>  non-ASCII: 0x<HH> ...
# VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro 2026-09-17-umbrelladashboard-module-creation]
# Mutation: LA2 -- remove the non-ASCII byte scan and LA2 stops failing on a UTF-8 accented lexicon
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-lexicon-ascii.sh <module-root>\n' >&2
    exit 3
fi

MODULE_ROOT="$1"

if [ ! -d "$MODULE_ROOT" ]; then
    printf 'lint-lexicon-ascii: not a directory: %s\n' "$MODULE_ROOT" >&2
    exit 3
fi

# Require od (POSIX; present on all target systems)
command -v od >/dev/null 2>&1 || {
    printf 'lint-lexicon-ascii: missing tool: od\n' >&2
    exit 3
}

_FAILED=0

# Find every *.lexicon under module root, dot-dirs pruned (D9b)
while IFS= read -r _lex; do
    _lrel="${_lex#"$MODULE_ROOT/"}"
    _lineno=0
    while IFS= read -r _raw_line; do
        _lineno=$((_lineno + 1))
        # Check for non-ASCII bytes using od -An -tx1 and filter for bytes > 7f.
        # We pass the raw line through od and look for any hex value > 7f.
        _bad=$(printf '%s\n' "$_raw_line" | od -An -tx1 | tr ' ' '\n' | grep -E '^[89a-fA-F][0-9a-fA-F]$' || true)
        if [ -n "$_bad" ]; then
            _hex=$(printf '%s\n' "$_bad" | while read -r _b; do printf '0x%s ' "$_b"; done | sed 's/ $//')
            printf 'FAIL  lint-lexicon-ascii  %s:%d  non-ASCII: %s\n' "$_lrel" "$_lineno" "$_hex"
            _FAILED=1
        fi
    done < "$_lex"
done < <(find "$MODULE_ROOT" \( -type d -name '.*' -prune \) -o \( -type f -name '*.lexicon' -print \) | LC_ALL=C sort)

[ "$_FAILED" -eq 0 ] && exit 0
exit 1
