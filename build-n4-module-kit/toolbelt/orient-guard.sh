#!/usr/bin/env bash
# orient-guard.sh — BUILD-LOOP §0.a location guard for Niagara N4 module builds.
#
# Asserts that <module-root> is under the sole legal build prefix
# /home/cristian/modulos_niagara_n4/Cliente/ before any build or file-editing action.
# Paths are canonicalized (symlinks + relative components resolved) before compare.
#
# Usage:  orient-guard.sh <module-root>
#
# Environment:
#   BUILD_N4_LEGAL_ROOT_PREFIX   Override the legal prefix (test seam — ALWAYS emits WARN).
#   BUILD_N4_CLIENTE_OVERRIDE    Set to exactly '1' to warn-and-pass on a non-legal path.
#                                Any other non-empty value is treated as unset (fail-closed).
#
# Row format:  PASS|FAIL|WARN  orient-guard  <detail>
# Exit:        0 (PASS or WARN) · 1 (FAIL) · 3 (usage error)
#
# This script is VCS-free by design. Version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
#
# Mutation: OG1 -- moves prefix compare to allow any path, making outside-prefix return PASS
# Mutation: OG2 -- removes the trailing-slash append before compare (ClienteX/ false-positive)
# Mutation: OG3 -- treats BUILD_N4_CLIENTE_OVERRIDE=0 as =1 (non-1 override bypass)
# Mutation: OG4 -- skips canonicalization, comparing raw input instead of realpath
set -euo pipefail

# ---------------------------------------------------------------------------
# Row emitter (matches preflight.sh convention)
# ---------------------------------------------------------------------------
row() {
  printf '%-4s  %-12s  %s\n' "$1" "orient-guard" "$2"
}

usage_exit() {
  printf 'usage: orient-guard.sh <module-root>\n' >&2
  exit 3
}

# ---------------------------------------------------------------------------
# Argument check
# ---------------------------------------------------------------------------
[ $# -eq 1 ] || usage_exit

ARG="$1"

# ---------------------------------------------------------------------------
# Legal prefix — hardcoded, overridable by test seam only (always WARN-loud)
# ---------------------------------------------------------------------------
readonly DEFAULT_LEGAL_ROOT_PREFIX="/home/cristian/modulos_niagara_n4/Cliente/"

if [ -n "${BUILD_N4_LEGAL_ROOT_PREFIX:-}" ]; then
  # Test seam is always loud — no silent bypass
  LEGAL_ROOT_PREFIX="${BUILD_N4_LEGAL_ROOT_PREFIX}"
  # Append trailing slash if missing
  case "$LEGAL_ROOT_PREFIX" in
    */) ;;
    *)  LEGAL_ROOT_PREFIX="${LEGAL_ROOT_PREFIX}/" ;;
  esac
  row "WARN" "non-default legal root ${LEGAL_ROOT_PREFIX} (test seam)"
else
  LEGAL_ROOT_PREFIX="${DEFAULT_LEGAL_ROOT_PREFIX}"
fi

# ---------------------------------------------------------------------------
# Canonicalize the argument (resolve symlinks, relative components)
# ---------------------------------------------------------------------------
if command -v realpath > /dev/null 2>&1; then
  CANON="$(realpath "$ARG" 2>/dev/null)" || {
    # realpath fails if path does not exist; fall back to cd -P for existing dirs
    CANON="$(cd -P "$ARG" 2>/dev/null && pwd -P)" || CANON="$ARG"
  }
else
  CANON="$(cd -P "$ARG" 2>/dev/null && pwd -P)" || CANON="$ARG"
fi

# Append trailing slash for prefix compare (prevents ClienteX/ false-positive)
CANON_SLASH="${CANON%/}/"

# ---------------------------------------------------------------------------
# Prefix check
# ---------------------------------------------------------------------------
if [[ "$CANON_SLASH" == "${LEGAL_ROOT_PREFIX}"* ]]; then
  row "PASS" "$CANON"
  exit 0
fi

# Outside legal prefix — check override
if [ "${BUILD_N4_CLIENTE_OVERRIDE:-}" = "1" ]; then
  row "WARN" "non-standard path: $CANON"
  exit 0
fi

# Fail-closed: unset, empty, or any non-'1' value → FAIL
row "FAIL" "$CANON — must be under ${DEFAULT_LEGAL_ROOT_PREFIX}; set BUILD_N4_CLIENTE_OVERRIDE=1 to override"
exit 1
