#!/usr/bin/env bash
# scaffold-module.sh — emit a buildable N4 single-profile -rt module skeleton.
# Fixture-driven (D1): copies the bundled fixtures/MinimalPan tree and substitutes
# MinimalPan → <ModuleName> in paths and content (B790/B793 proven tree).
#
# Usage: scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>]
#   ModuleName       Java identifier: uppercase-first, alphanumeric only (e.g. ColdRoomPan)
#   out-dir          parent directory; emitted root is <out-dir>/<ModuleName>
#   --vendor <v>     display vendor name (copyright, defaultVendor); default Angeles
#   --target-version <x.y>  baja target version; default 4.14
#   --plugin-version <v>    Niagara Gradle plugin version baked into settings.gradle.kts; default 7.6.17
#
# The emitted tree is pre-slotomatic. Run build.sh to generate the AUTO region and build the jar.
# This script is VCS-free by design. version control is never invoked.
#
# Exit: 0 ok · 2 usage or invalid ModuleName · 3 env (skeleton missing, out-dir not creatable,
#        or <out-dir>/<ModuleName> already exists)
set -u

################################################################
# Skeleton resolved relative to this script — never from $HOME
# (K8; TC-K8 verifies the script works under HOME=/nonexistent)
################################################################
_SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
FIXTURE_ROOT="${_SCRIPT_DIR}/../fixtures/MinimalPan"

################################################################
# usage
################################################################
usage() {
  printf 'usage: scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>]\n' >&2
}

################################################################
# Argument parsing
################################################################
if [ $# -lt 2 ]; then usage; exit 2; fi

MOD="$1"; shift
OUT="$1"; shift

VENDOR="Angeles"
TARGET_VERSION="4.14"
PLUGIN_VERSION="7.6.17"

while [ $# -gt 0 ]; do
  case "$1" in
    --vendor)
      if [ $# -lt 2 ]; then usage; exit 2; fi
      VENDOR="$2"; shift 2 ;;
    --target-version)
      if [ $# -lt 2 ]; then usage; exit 2; fi
      TARGET_VERSION="$2"; shift 2 ;;
    --plugin-version)
      if [ $# -lt 2 ]; then usage; exit 2; fi
      PLUGIN_VERSION="$2"; shift 2 ;;
    *)
      printf 'scaffold-module: unknown option: %s\n' "$1" >&2; usage; exit 2 ;;
  esac
done

################################################################
# Validate ModuleName: uppercase-first, alphanumeric only
# (Java identifier convention, required for Baja type registration)
################################################################
case "$MOD" in
  [A-Z]*) ;;
  *)
    printf 'scaffold-module: ModuleName must start with an uppercase letter: %s\n' "$MOD" >&2
    exit 2 ;;
esac
case "$MOD" in
  *[^A-Za-z0-9]*)
    printf 'scaffold-module: ModuleName must contain only alphanumeric characters: %s\n' "$MOD" >&2
    exit 2 ;;
esac

################################################################
# Environment checks
################################################################
if [ ! -d "${FIXTURE_ROOT}/gradle/wrapper" ]; then
  printf 'scaffold-module: fixture skeleton not found: %s\n' "${FIXTURE_ROOT}" >&2
  exit 3
fi

if ! mkdir -p "$OUT" 2>/dev/null; then
  if [ ! -d "$OUT" ]; then
    printf 'scaffold-module: cannot create out-dir: %s\n' "$OUT" >&2
    exit 3
  fi
fi

DEST="${OUT}/${MOD}"
if [ -e "$DEST" ]; then
  printf 'scaffold-module: destination already exists: %s\n' "$DEST" >&2
  exit 3
fi

################################################################
# Derived values
################################################################
VENDOR_LC="$(printf '%s' "$VENDOR" | tr '[:upper:]' '[:lower:]')"
# lowerCamel ModuleName — 0 occurrences in the current fixture; retained for future var-name sites
MOD_LC="$(printf '%s' "${MOD:0:1}" | tr '[:upper:]' '[:lower:]')${MOD:1}"
# Palette namespace prefix — uppercase initials of ModuleName, lowercased (e.g. MinimalPan → mp)
SYM="$(printf '%s' "$MOD" | tr -dc '[:upper:]' | tr '[:upper:]' '[:lower:]')"

################################################################
# Copy fixture tree — rename paths and substitute content
################################################################
while IFS= read -r -d '' src; do
  # Relative path from fixture root
  rel="${src#"${FIXTURE_ROOT}"/}"

  # Transform path segments: MinimalPan-rt before MinimalPan (more-specific first)
  dst_rel="$(printf '%s' "$rel" \
    | sed \
        -e "s|MinimalPan-rt|${MOD}-rt|g" \
        -e "s|MinimalPan|${MOD}|g" \
        -e "s|angeles|${VENDOR_LC}|g")"

  dst="${DEST}/${dst_rel}"
  mkdir -p "$(dirname "$dst")"

  case "$src" in
    # Binary: copy verbatim — never run sed on jars or wrapper scripts
    *.jar | */gradlew | */gradlew.bat)
      cp "$src" "$dst" ;;
    *)
      # Text files: all substitutions in one pass (ordered most-specific → less-specific)
      sed \
        -e "s|MinimalPan-rt|${MOD}-rt|g" \
        -e "s|minimalPan|${MOD_LC}|g" \
        -e "s|MinimalPan|${MOD}|g" \
        -e "s|Angeles|${VENDOR}|g" \
        -e "s|angeles|${VENDOR_LC}|g" \
        -e "s|mp=|${SYM}=|g" \
        -e "s|mp:|${SYM}:|g" \
        -e "s|7\.6\.17|${PLUGIN_VERSION}|g" \
        -e "s|4\.14|${TARGET_VERSION}|g" \
        "$src" > "$dst" ;;
  esac
done < <(find "$FIXTURE_ROOT" -type f -print0)

# Preserve executable bit on gradlew (binary copy loses the chmod recorded in git)
chmod +x "${DEST}/gradlew"

printf 'scaffold-module: emitted %s -> %s\n' "$MOD" "$DEST"
