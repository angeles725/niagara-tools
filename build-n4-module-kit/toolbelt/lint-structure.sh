#!/usr/bin/env bash
# lint-structure.sh — module-structure lint for Niagara N4 source trees (Campaign 8 PR18, B817).
#
# Iterates every profile under <module-root> by finding each module-include.xml.
# Source-tree-only; L8 (signed-jar check) stays in verify-module.sh (D15). Dot-directories
# are pruned (D9b). Reports per-check rows.
#
# Checks:
#   L1  FAIL  package naming — every .java package is com.<vendor>.*, never javax.baja.*
#   L2  FAIL  one @NiagaraType per .java file (max one public type per file)
#   L3  WARN  pure-model package (src/.../model/) has tests and no baja imports (advisory)
#   L4  FAIL  module.lexicon non-empty (>=1 key=value) when >=1 type is declared
#   L5  FAIL  module.palette non-empty for -rt profile
#   L6  FAIL  module-include.xml present; no hand-authored META-INF/module.xml in source
#   L7  FAIL  dependency version is a 3-part floor (4.14.0), not 2-part (4.14)
#   L9  FAIL  no empty skeleton artifact: -wb/-ux with 0 Java classes AND empty palette
#   L10 FAIL  no absolute host paths in tracked gradle.properties
#   L11 FAIL  mixed srcTest (BTest+JUnit) without both :test-wb AND junit gradle declarations
#   (L8 signed-jar check is in verify-module.sh)
#
# Usage:  lint-structure.sh <module-root>
#
#   Row format:  FAIL|WARN  lint-structure  <path>  L<n>: <reason>
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20)
#
# This script is VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign8-structure]
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-structure.sh <module-root>\n' >&2
    exit 3
fi

MODULE_ROOT="$1"

if [ ! -d "$MODULE_ROOT" ]; then
    printf 'lint-structure: not a directory: %s\n' "$MODULE_ROOT" >&2
    exit 3
fi

_TMP=$(mktemp -d)
trap 'rm -rf "$_TMP"' EXIT
_ROWS="$_TMP/rows.txt"
touch "$_ROWS"

_row() {
    local sev="$1" path="$2" reason="$3"
    printf '%s  lint-structure  %s  %s\n' "$sev" "$path" "$reason" >> "$_ROWS"
}

# Find helper: files under $1 matching name $2, dot-directories pruned (D9b)
_find_files() {
    find "$1" \( -type d -name '.*' -prune \) -o \( -type f -name "$2" -print \)
}

# Find helper: *.java files under $1, dot-directories pruned (D9b)
_find_java() {
    find "$1" \( -type d -name '.*' -prune \) -o \( -type f -name '*.java' -print \)
}

# ---------------------------------------------------------------------------
# L10: no absolute host paths in tracked gradle.properties (module-root level)
# Dot-directories pruned, so .deploy-baseline/ is excluded (D9b)
# Patterns: niagara_home=C:\..., niagara_user_home=C:\..., nodeHome=C:\...
# ---------------------------------------------------------------------------
while IFS= read -r _gp; do
    if LC_ALL=C grep -qE '^(niagara_home|niagara_user_home|user_home|nodeHome)=[A-Za-z]:[\\/]' \
            "$_gp" 2>/dev/null; then
        _row FAIL "$_gp" "L10: absolute host path in tracked gradle.properties"
    fi
done < <(_find_files "$MODULE_ROOT" "gradle.properties" | LC_ALL=C sort)

# ---------------------------------------------------------------------------
# Per-profile checks: each module-include.xml defines one profile
# ---------------------------------------------------------------------------
while IFS= read -r _inc; do
    _PDIR="$(dirname "$_inc")"
    _PNAME="$(basename "$_PDIR")"
    _LEX="$_PDIR/module.lexicon"
    _PAL="$_PDIR/module.palette"
    _META="$_PDIR/META-INF/module.xml"
    _SRC="$_PDIR/src"
    _SRCTEST="$_PDIR/srcTest"

    # -----------------------------------------------------------------------
    # L6: hand-authored META-INF/module.xml in source — the gradle plugin generates it
    # -----------------------------------------------------------------------
    if [ -f "$_META" ]; then
        _rel="${_META#"$MODULE_ROOT/"}"
        _row FAIL "$_rel" "L6: hand-authored META-INF/module.xml found; author module-include.xml, not the generated manifest"
    fi

    # -----------------------------------------------------------------------
    # Count declared types in module-include.xml (used by L4)
    # -----------------------------------------------------------------------
    _TYPE_COUNT=0
    _TYPE_COUNT=$(LC_ALL=C grep -c '<type ' "$_inc" 2>/dev/null || true)
    [ -z "$_TYPE_COUNT" ] && _TYPE_COUNT=0

    # -----------------------------------------------------------------------
    # L4: module.lexicon non-empty when >=1 type is declared
    # -----------------------------------------------------------------------
    if [ "${_TYPE_COUNT:-0}" -gt 0 ]; then
        if [ ! -f "$_LEX" ]; then
            _prel="${_PDIR#"$MODULE_ROOT/"}"
            _row FAIL "$_prel" "L4: module.lexicon missing ($_TYPE_COUNT type(s) declared in module-include.xml)"
        elif ! LC_ALL=C grep -qE '^[^#[:space:]][^=]*=' "$_LEX" 2>/dev/null; then
            _lrel="${_LEX#"$MODULE_ROOT/"}"
            _row FAIL "$_lrel" "L4: lexicon empty ($_TYPE_COUNT type(s) declared, zero key=value entries)"
        fi
    fi

    # -----------------------------------------------------------------------
    # L5: module.palette non-empty for -rt profiles
    # -----------------------------------------------------------------------
    case "$_PNAME" in
        *-rt)
            if [ ! -f "$_PAL" ]; then
                _prel="${_PDIR#"$MODULE_ROOT/"}"
                _row FAIL "$_prel" "L5: module.palette missing for -rt profile"
            elif ! LC_ALL=C grep -q '<p n=' "$_PAL" 2>/dev/null; then
                _palrel="${_PAL#"$MODULE_ROOT/"}"
                _row FAIL "$_palrel" "L5: module.palette has no named component entries (empty palette — see B788)"
            fi
            ;;
    esac

    # -----------------------------------------------------------------------
    # L7: dependency version must be a 3-part floor (X.Y.Z), not 2-part (X.Y)
    # Scans *.gradle.kts in the profile directory for patterns like ":baja:4.14"
    # (the closing " prevents matching ":baja:4.14.0")
    # -----------------------------------------------------------------------
    while IFS= read -r _gkts; do
        _grel="${_gkts#"$MODULE_ROOT/"}"
        if LC_ALL=C grep -qE '":[^:]+:[0-9]+\.[0-9]+"' "$_gkts" 2>/dev/null; then
            _row FAIL "$_grel" "L7: dependency version is not a 3-part floor (use X.Y.Z e.g. 4.14.0, not X.Y e.g. 4.14)"
        fi
    done < <(_find_files "$_PDIR" "*.gradle.kts" | LC_ALL=C sort)

    # -----------------------------------------------------------------------
    # Count Java source files under src/ (used by L9)
    # -----------------------------------------------------------------------
    _JAVA_COUNT=0
    if [ -d "$_SRC" ]; then
        _JAVA_COUNT=$(_find_java "$_SRC" | wc -l | tr -d ' ')
    fi

    # -----------------------------------------------------------------------
    # L9: empty skeleton artifact — -wb/-ux profile with 0 Java files AND empty palette
    # -----------------------------------------------------------------------
    case "$_PNAME" in
        *-wb | *-ux)
            _PAL_EMPTY=1
            if [ -f "$_PAL" ] && LC_ALL=C grep -q '<p n=' "$_PAL" 2>/dev/null; then
                _PAL_EMPTY=0
            fi
            if [ "$_JAVA_COUNT" -eq 0 ] && [ "$_PAL_EMPTY" -eq 1 ]; then
                _prel="${_PDIR#"$MODULE_ROOT/"}"
                _row FAIL "$_prel" "L9: empty skeleton artifact (0 Java classes, empty palette — delete or populate the profile)"
            fi
            ;;
    esac

    # -----------------------------------------------------------------------
    # L1, L2: Java source checks (src/ only, dot-dirs pruned)
    # L1: package must never declare javax.baja.* (that is the framework namespace)
    # L2: at most one @NiagaraType annotation per file
    # -----------------------------------------------------------------------
    if [ -d "$_SRC" ]; then
        while IFS= read -r _jf; do
            _jrel="${_jf#"$MODULE_ROOT/"}"

            # L1
            if LC_ALL=C grep -qE '^[[:space:]]*package[[:space:]]+javax\.baja\.' "$_jf" 2>/dev/null; then
                _row FAIL "$_jrel" "L1: package declares javax.baja.* (framework namespace; OEM modules use com.<vendor>.*)"
            fi

            # L2: count only annotation-position lines (^[[:space:]]*@NiagaraType),
            # not occurrences in Javadoc comments like " * @NiagaraType / ..."
            _NT=0
            _NT=$(LC_ALL=C grep -cE '^[[:space:]]*@NiagaraType' "$_jf" 2>/dev/null || true)
            [ -z "$_NT" ] && _NT=0
            if [ "${_NT:-0}" -gt 1 ]; then
                _row FAIL "$_jrel" "L2: ${_NT} @NiagaraType annotations in one file (one public type per file)"
            fi
        done < <(_find_java "$_SRC" | LC_ALL=C sort)
    fi

    # -----------------------------------------------------------------------
    # L3: pure-model package advisory — src/.../model/ with javax.baja.* imports (WARN)
    # Advisory only: the only WARN in the checks; --strict is not implemented for this tool.
    # -----------------------------------------------------------------------
    if [ -d "$_SRC" ]; then
        while IFS= read -r _mdir; do
            _mrel="${_mdir#"$MODULE_ROOT/"}"
            _found_baja=0
            while IFS= read -r _mf; do
                if LC_ALL=C grep -qE '^[[:space:]]*import[[:space:]]+javax\.baja\.' "$_mf" 2>/dev/null; then
                    _found_baja=1
                    break
                fi
            done < <(_find_java "$_mdir" | LC_ALL=C sort)
            if [ "$_found_baja" -eq 1 ]; then
                _row WARN "$_mrel" "L3: pure-model package contains javax.baja.* imports (advisory; isolate control logic from Baja for testability)"
            fi
        done < <(find "$_SRC" \( -type d -name '.*' -prune \) -o \( -type d -name 'model' -print \))
    fi

    # -----------------------------------------------------------------------
    # L11: mixed srcTest (BTest + JUnit) must declare BOTH :test-wb AND junit
    # -----------------------------------------------------------------------
    if [ -d "$_SRCTEST" ]; then
        _HAS_BTEST=0
        _HAS_JUNIT=0
        while IFS= read -r _tf; do
            if LC_ALL=C grep -qE \
                    '^[[:space:]]*import[[:space:]]+(javax\.baja\.test\.|com\.tridium\.btest\.)' \
                    "$_tf" 2>/dev/null; then
                _HAS_BTEST=1
            fi
            if LC_ALL=C grep -qE '^[[:space:]]*import[[:space:]]+org\.junit\.' \
                    "$_tf" 2>/dev/null; then
                _HAS_JUNIT=1
            fi
        done < <(_find_java "$_SRCTEST" | LC_ALL=C sort)

        if [ "$_HAS_BTEST" -eq 1 ] && [ "$_HAS_JUNIT" -eq 1 ]; then
            # Mixed test sources: both :test-wb and junit must appear in gradle.kts
            _HAS_TESTWB=0
            _HAS_JUNITDEP=0
            while IFS= read -r _gkts; do
                if LC_ALL=C grep -qE '":test-wb"' "$_gkts" 2>/dev/null; then
                    _HAS_TESTWB=1
                fi
                if LC_ALL=C grep -qE '"junit[:/][^"]*"' "$_gkts" 2>/dev/null; then
                    _HAS_JUNITDEP=1
                fi
            done < <(_find_files "$_PDIR" "*.gradle.kts" | LC_ALL=C sort)

            if [ "$_HAS_TESTWB" -eq 0 ] || [ "$_HAS_JUNITDEP" -eq 0 ]; then
                _prel="${_PDIR#"$MODULE_ROOT/"}"
                _row FAIL "$_prel" "L11: mixed BTest+JUnit srcTest without both :test-wb and junit gradle declarations"
            fi
        fi
    fi

done < <(find "$MODULE_ROOT" \( -type d -name '.*' -prune \) \
             -o \( -type f -name 'module-include.xml' -print \) | LC_ALL=C sort)

# ---------------------------------------------------------------------------
# Output rows and exit based on whether any FAIL was emitted
# ---------------------------------------------------------------------------
cat "$_ROWS"
LC_ALL=C grep -q '^FAIL' "$_ROWS" && exit 1
exit 0
