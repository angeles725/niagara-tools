#!/usr/bin/env bash
# install-skill.sh — install a skill launcher into the Claude skills directory.
#
# Copies the tracked canonical launcher into <home>/.claude/skills/<skill>/SKILL.md
# using a sha256 comparison to detect divergence:
#   --skill build-n4-module (default): build-n4-module-kit/skill/SKILL.md
#   --skill mcp-n4:                    mcp-n4-kit/skill/SKILL.md
# (the SKILLS table below is the single source of these names and paths)
#
# --skill build-n4-module also MATERIALIZES the kit's generated state in the kit checkout the launcher
# reads: BUILD-STATE.md and retros/INDEX.md are generated and not committed (R10-D1), so a fresh install
# runs `<kit>/toolbelt/kit-state.sh --write` (on every run, also when the launcher is already current).
# The kit is $BUILD_N4_KIT when it holds METHODOLOGY.md (the launcher's own first rule), else this
# repository's build-n4-module-kit/. [ev: retro ledger-generated-not-committed]
#
# This script is VCS-free by design — version control is never invoked here.
# The source of truth lives in the repository; this script only installs it.
#
# Usage:
#   install-skill.sh [--skill <build-n4-module|mcp-n4>] [--home <dir>] [--dry-run] [--force]
#
# Options:
#   --skill <name> Skill to install: build-n4-module (default) or mcp-n4.
#   --home <dir>   Base home directory (default: $HOME). Every test passes this flag
#                  so no test ever touches the real $HOME.
#   --dry-run      Print what would be done; write nothing. Exits 0.
#   --force        Overwrite a diverged installed copy without exiting 1.
#
# Exit codes:
#   0   Installed or already current (or --dry-run); for build-n4-module, kit state materialized
#       (a kit without toolbelt/kit-state.sh is a warning, not an error).
#   1   Installed copy exists and diverges from the tracked copy; --force absent.
#   2   Usage error (including an unknown --skill name).
#   3   Environment error (tracked source missing, no sha256 tool, target directory not creatable,
#       kit-state failed: the launcher is installed but the kit state is not).
#
# shellcheck disable=SC2006  # not used; POSIX $() throughout
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via SCRIPT_DIR
. "$SCRIPT_DIR/lib/sha256.sh" || exit 3

# The one place that names the skills: "<name>:<tracked launcher, relative to the repo>".
# The usage text, the error messages and the lookup below are all derived from it.
SKILLS="build-n4-module:build-n4-module-kit/skill/SKILL.md mcp-n4:mcp-n4-kit/skill/SKILL.md"
SKILL_NAMES=""
for entry in $SKILLS; do
  SKILL_NAMES="${SKILL_NAMES:+$SKILL_NAMES|}${entry%%:*}"
done
USAGE="usage: install-skill.sh [--skill <$SKILL_NAMES>] [--home <dir>] [--dry-run] [--force]"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
SKILL="build-n4-module"
HOME_DIR=""
DRY_RUN=0
FORCE=0

while [ $# -gt 0 ]; do
  case "$1" in
    --home)
      shift
      [ $# -ge 1 ] || { printf 'install-skill: --home requires an argument\n' >&2; exit 2; }
      HOME_DIR="$1"
      shift
      ;;
    --skill)
      shift
      [ $# -ge 1 ] || { printf 'install-skill: --skill requires an argument\n' >&2; exit 2; }
      SKILL="$1"
      shift
      ;;
    --dry-run)  DRY_RUN=1; shift ;;
    --force)    FORCE=1;   shift ;;
    --)         shift; break ;;
    -*)
      printf 'install-skill: unknown option: %s\n' "$1" >&2
      printf '%s\n' "$USAGE" >&2
      exit 2
      ;;
    *)
      printf 'install-skill: unexpected argument: %s\n' "$1" >&2
      printf '%s\n' "$USAGE" >&2
      exit 2
      ;;
  esac
done

# Default home: $HOME (but never used as a path resolution strategy in tests)
[ -n "$HOME_DIR" ] || HOME_DIR="$HOME"

# ---------------------------------------------------------------------------
# Resolve the skill to its tracked source
# ---------------------------------------------------------------------------
TRACKED=""
for entry in $SKILLS; do
  if [ "${entry%%:*}" = "$SKILL" ]; then
    TRACKED="$REPO_ROOT/${entry#*:}"
  fi
done
if [ -z "$TRACKED" ]; then
  printf 'install-skill: unknown skill: %s (expected %s)\n' "$SKILL" "${SKILL_NAMES//|/ or }" >&2
  printf '%s\n' "$USAGE" >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# Validate tracked source
# ---------------------------------------------------------------------------
if [ ! -f "$TRACKED" ]; then
  printf 'install-skill: tracked source not found: %s\n' "$TRACKED" >&2
  exit 3
fi

# The kit checkout whose generated state a build-n4-module install materializes.
KIT_DIR=""
if [ "$SKILL" = build-n4-module ]; then
  if [ -n "${BUILD_N4_KIT:-}" ] && [ -f "$BUILD_N4_KIT/METHODOLOGY.md" ]; then KIT_DIR="$BUILD_N4_KIT"
  else KIT_DIR="$REPO_ROOT/build-n4-module-kit"; fi
fi

# materialize: run the kit's kit-state --write (it prints each path written); exit 3 when it fails.
materialize() {
  [ -n "$KIT_DIR" ] || return 0
  if [ ! -f "$KIT_DIR/toolbelt/kit-state.sh" ]; then
    printf 'install-skill: no toolbelt/kit-state.sh in %s; generated kit state not materialized\n' "$KIT_DIR" >&2
    return 0
  fi
  bash "$KIT_DIR/toolbelt/kit-state.sh" --write "$KIT_DIR" || {
    printf 'install-skill: kit-state --write failed in %s; the launcher is installed, the kit state is not\n' "$KIT_DIR" >&2
    exit 3
  }
}

TARGET_DIR="$HOME_DIR/.claude/skills/$SKILL"
TARGET="$TARGET_DIR/SKILL.md"

sha256_require 'install-skill' || exit 3   # up front: an exit inside $( ) cannot stop us
tracked_sha=$(sha256_of "$TRACKED")

# ---------------------------------------------------------------------------
# Dry-run mode: report without writing
# ---------------------------------------------------------------------------
if [ "$DRY_RUN" -eq 1 ]; then
  if [ -f "$TARGET" ]; then
    target_sha=$(sha256_of "$TARGET")
    if [ "$tracked_sha" = "$target_sha" ]; then
      printf 'install-skill: [dry-run] already current: %s\n' "$TARGET"
    else
      printf 'install-skill: [dry-run] would update (diverged): %s\n' "$TARGET"
      printf '  tracked sha256: %s\n' "$tracked_sha"
      printf '  current sha256: %s\n' "$target_sha"
    fi
  else
    printf 'install-skill: [dry-run] would install to: %s\n' "$TARGET"
  fi
  [ -z "$KIT_DIR" ] || printf 'install-skill: [dry-run] would run kit-state --write in: %s\n' "$KIT_DIR"
  exit 0
fi

# ---------------------------------------------------------------------------
# Check if already current
# ---------------------------------------------------------------------------
if [ -f "$TARGET" ]; then
  target_sha=$(sha256_of "$TARGET")
  if [ "$tracked_sha" = "$target_sha" ]; then
    printf 'install-skill: already current: %s\n' "$TARGET"
    materialize
    exit 0
  fi
  # Diverged
  if [ "$FORCE" -eq 0 ]; then
    printf 'install-skill: installed copy diverges from tracked source\n' >&2
    printf '  tracked:  %s\n' "$TRACKED" >&2
    printf '  installed: %s\n' "$TARGET" >&2
    printf '  tracked sha256:   %s\n' "$tracked_sha" >&2
    printf '  installed sha256: %s\n' "$target_sha" >&2
    printf 'Re-run with --force to overwrite.\n' >&2
    exit 1
  fi
  printf 'install-skill: --force: overwriting diverged copy\n'
fi

# ---------------------------------------------------------------------------
# Create target directory
# ---------------------------------------------------------------------------
if ! mkdir -p "$TARGET_DIR" 2>/dev/null; then
  printf 'install-skill: cannot create target directory: %s\n' "$TARGET_DIR" >&2
  exit 3
fi

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------
cp "$TRACKED" "$TARGET"
printf 'install-skill: installed %s\n' "$TARGET"
materialize
exit 0
