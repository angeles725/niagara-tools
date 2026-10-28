#!/usr/bin/env bash
# scaffold-module.sh — emit a buildable N4 module skeleton from a bundled fixture.
# Fixture-driven (D1): copies the bundled fixtures/<Type> tree and substitutes
# the fixture type name → <ModuleName> in paths and content (B790/B793 proven tree).
#
# Usage: scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>] [--type <logic|dashboard|driver>]
#   ModuleName       Java identifier: uppercase-first, alphanumeric only (e.g. ColdRoomPan)
#   out-dir          parent directory; emitted root is <out-dir>/<ModuleName>
#   --vendor <v>     display vendor name (copyright, defaultVendor); default Angeles
#   --target-version <x.y>  baja target version; default 4.14
#   --plugin-version <v>    Niagara Gradle plugin version baked into settings.gradle.kts; default 7.6.17
#   --type <logic|dashboard|driver>  scaffold type (default: logic)
#                    logic     — single-profile -rt skeleton (MinimalPan fixture) + station-test overlay
#                    dashboard — two-profile -rt facade + -ux servlet skeleton (MinimalDash fixture) + station-test overlay
#                    driver    — NDriver -rt (UDP comm config subclass, codec, discovery) + -wb point manager
#                                (MinimalDriver fixture; types/driver-authoring.md § 1.6-§ 1.11, § 5, § 12)
#
# Every type also gets the client version ledger <out-dir>/<ModuleName>/odd/VERSION-LEDGER.md (templates/, its
# sample row dropped; check it with ledger-check.sh).
# The emitted tree is pre-slotomatic. Run build.sh to generate the AUTO region and build the jar.
# The root build.gradle.kts carries the signing block (templates/signing-root.gradle.kts, issue #266):
# set niagaraSigningAlias and niagaraSigningProfile as gradle properties before the first build.
# This script is VCS-free by design. version control is never invoked.
#
# Exit: 0 ok · 2 usage or invalid ModuleName or unknown --type · 3 env (skeleton missing,
#        out-dir not creatable, or <out-dir>/<ModuleName> already exists)
#
# Add a concern to an emitted (or any mapped) module:
#   scaffold-module.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]
# execs toolbelt/module-map.sh add-concern with the same arguments (its header has the contract and exits).
# [ev: retro modularity-structure Δ1]
set -u

################################################################
# Skeleton resolved relative to this script — never from $HOME
# (K8; TC-K8 verifies the script works under HOME=/nonexistent)
################################################################
_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

################################################################
# usage
################################################################
usage() {
  printf 'usage: scaffold-module.sh <ModuleName> <out-dir> [--vendor <v>] [--target-version <x.y>] [--plugin-version <v>] [--type <logic|dashboard|driver>]\n' >&2
  printf '       scaffold-module.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]\n' >&2
}

################################################################
# add-concern: the module-map tool owns it (Go, go/internal/modulemap)
################################################################
if [ "${1:-}" = "add-concern" ]; then
  exec "${_SCRIPT_DIR}/module-map.sh" "$@"
fi

################################################################
# Argument parsing
################################################################
if [ $# -lt 2 ]; then usage; exit 2; fi

MOD="$1"; shift
OUT="$1"; shift

VENDOR="Angeles"
TARGET_VERSION="4.14"
PLUGIN_VERSION="7.6.17"
TYPE="logic"

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
    --type)
      if [ $# -lt 2 ]; then usage; exit 2; fi
      TYPE="$2"; shift 2 ;;
    *)
      printf 'scaffold-module: unknown option: %s\n' "$1" >&2; usage; exit 2 ;;
  esac
done

################################################################
# Validate --type
################################################################
case "$TYPE" in
  logic | dashboard | driver) ;;
  *)
    printf 'scaffold-module: unknown --type: %s (must be logic, dashboard or driver)\n' "$TYPE" >&2
    usage; exit 2 ;;
esac

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
# Select fixture based on --type
################################################################
case "$TYPE" in
  logic)     FIXTURE_TYPE_NAME="MinimalPan" ;;
  dashboard) FIXTURE_TYPE_NAME="MinimalDash" ;;
  driver)    FIXTURE_TYPE_NAME="MinimalDriver" ;;
esac
FIXTURE_ROOT="${_SCRIPT_DIR}/../fixtures/${FIXTURE_TYPE_NAME}"

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
# lowerCamel ModuleName — retained for future var-name sites (0 occurrences in current fixtures)
MOD_LC="$(printf '%s' "${MOD:0:1}" | tr '[:upper:]' '[:lower:]')${MOD:1}"
# Palette namespace prefix — uppercase initials of ModuleName, lowercased (e.g. MinimalPan → mp)
SYM="$(printf '%s' "$MOD" | tr -dc '[:upper:]' | tr '[:upper:]' '[:lower:]')"

# Fixture-type derived values (used in sed patterns instead of hard-coded fixture names)
FIXTURE_TYPE_NAME_LC="$(printf '%s' "${FIXTURE_TYPE_NAME:0:1}" | tr '[:upper:]' '[:lower:]')${FIXTURE_TYPE_NAME:1}"
FIXTURE_TYPE_SYM="$(printf '%s' "$FIXTURE_TYPE_NAME" | tr -dc '[:upper:]' | tr '[:upper:]' '[:lower:]')"

################################################################
# Copy fixture tree — rename paths and substitute content
################################################################
while IFS= read -r -d '' src; do
  # Relative path from fixture root
  rel="${src#"${FIXTURE_ROOT}"/}"

  # Transform path segments (more-specific first: -ux/-wb before -rt before bare name)
  dst_rel="$(printf '%s' "$rel" \
    | sed \
        -e "s|${FIXTURE_TYPE_NAME}-ux|${MOD}-ux|g" \
        -e "s|${FIXTURE_TYPE_NAME}-wb|${MOD}-wb|g" \
        -e "s|${FIXTURE_TYPE_NAME}-rt|${MOD}-rt|g" \
        -e "s|${FIXTURE_TYPE_NAME}|${MOD}|g" \
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
        -e "s|${FIXTURE_TYPE_NAME}-ux|${MOD}-ux|g" \
        -e "s|${FIXTURE_TYPE_NAME}-wb|${MOD}-wb|g" \
        -e "s|${FIXTURE_TYPE_NAME}-rt|${MOD}-rt|g" \
        -e "s|${FIXTURE_TYPE_NAME_LC}|${MOD_LC}|g" \
        -e "s|${FIXTURE_TYPE_NAME}|${MOD}|g" \
        -e "s|Angeles|${VENDOR}|g" \
        -e "s|angeles|${VENDOR_LC}|g" \
        -e "s|${FIXTURE_TYPE_SYM}=|${SYM}=|g" \
        -e "s|${FIXTURE_TYPE_SYM}:|${SYM}:|g" \
        -e "s|7\.6\.17|${PLUGIN_VERSION}|g" \
        -e "s|4\.14|${TARGET_VERSION}|g" \
        "$src" > "$dst" ;;
  esac
done < <(find "$FIXTURE_ROOT" -type f -print0)

################################################################
# Station-test overlay (logic + dashboard only): a Fox-off BTestNgStation fixture base,
# a link-in sample test and moduleTest-include.xml, from templates/station-test
# (types/moduleTest.md). The driver fixture ships its own loopback station test.
# Template tokens: ModuleA → <ModuleName>, com.example → com.<vendor-lc>,
# Example → <vendor>, @LINK_IN_SLOTS@ → the fixture component's link-in slots.
################################################################
case "$TYPE" in
  logic)     LINK_IN_SLOTS='"setpoint", "interval"' ;;
  dashboard) LINK_IN_SLOTS='"setpoint", "status"' ;;
  *)         LINK_IN_SLOTS="" ;;
esac
if [ -n "$LINK_IN_SLOTS" ]; then
  ST_ROOT="${_SCRIPT_DIR}/../templates/station-test"
  if [ ! -f "${ST_ROOT}/moduleTest-include.xml" ]; then
    printf 'scaffold-module: station-test template not found: %s\n' "${ST_ROOT}" >&2
    exit 3
  fi
  ST_DEST="${DEST}/${MOD}/${MOD}-rt"
  while IFS= read -r -d '' src; do
    rel="${src#"${ST_ROOT}"/}"
    dst_rel="$(printf '%s' "$rel" \
      | sed -e "s|com/example/|com/${VENDOR_LC}/|g" -e "s|ModuleA|${MOD}|g")"
    dst="${ST_DEST}/${dst_rel}"
    mkdir -p "$(dirname "$dst")"
    sed \
      -e "s|@LINK_IN_SLOTS@|${LINK_IN_SLOTS}|g" \
      -e "s|com\.example\.|com.${VENDOR_LC}.|g" \
      -e "s|ModuleA|${MOD}|g" \
      -e "s|Example|${VENDOR}|g" \
      "$src" > "$dst"
  done < <(find "$ST_ROOT" -type f -print0)
fi

################################################################
# Client version ledger (every type): templates/VERSION-LEDGER.md as <gradle-root>/odd/VERSION-LEDGER.md, the path
# the template names (BUILD-LOOP.md §0 reads it, §7 gates a close on its rows), with the template's sample row
# dropped so the first real build adds the first row; toolbelt/ledger-check.sh passes it clean.
# (the 2026-09-26 version-defect-ledger retro, Δ1)
################################################################
LEDGER_TPL="${_SCRIPT_DIR}/../templates/VERSION-LEDGER.md"
if [ ! -f "$LEDGER_TPL" ]; then
  printf 'scaffold-module: version-ledger template not found: %s\n' "$LEDGER_TPL" >&2
  exit 3
fi
mkdir -p "${DEST}/odd"
grep -v '^| <Module> | <x\.y\.z> |' "$LEDGER_TPL" > "${DEST}/odd/VERSION-LEDGER.md"

# Preserve executable bit on gradlew (binary copy loses the chmod recorded in git)
chmod +x "${DEST}/gradlew"

printf 'scaffold-module: emitted %s -> %s\n' "$MOD" "$DEST"
printf 'scaffold-module: signing: set niagaraSigningAlias + niagaraSigningProfile (gradle properties) before the first build\n'
