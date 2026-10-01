#!/usr/bin/env bash
# install-skill.sh — install a skill launcher into the Claude skills directory.
#
# Copies the tracked canonical launcher into <home>/.claude/skills/<skill>/SKILL.md
# using a sha256 comparison to detect divergence:
#   --skill build-n4-module (default): build-n4-module-kit/skill/SKILL.md
#   --skill mcp-n4:                    mcp-n4-kit/skill/SKILL.md
# (the SKILLS table below is the single source of these names and paths)
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
#   0   Installed or already current (or --dry-run).
#   1   Installed copy exists and diverges from the tracked copy; --force absent.
#   2   Usage error (including an unknown --skill name).
#   3   Environment error (target directory not creatable).
#
# shellcheck disable=SC2006  # not used; POSIX $() throughout
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

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

TARGET_DIR="$HOME_DIR/.claude/skills/$SKILL"
TARGET="$TARGET_DIR/SKILL.md"

# ---------------------------------------------------------------------------
# Sha256 helper — works on both Linux (sha256sum) and macOS (shasum -a 256)
# ---------------------------------------------------------------------------
sha256_of() {
  local f="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$f" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$f" | cut -d' ' -f1
  else
    printf 'install-skill: no sha256 tool found (sha256sum or shasum)\n' >&2
    exit 3
  fi
}

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
  exit 0
fi

# ---------------------------------------------------------------------------
# Check if already current
# ---------------------------------------------------------------------------
if [ -f "$TARGET" ]; then
  target_sha=$(sha256_of "$TARGET")
  if [ "$tracked_sha" = "$target_sha" ]; then
    printf 'install-skill: already current: %s\n' "$TARGET"
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
exit 0
