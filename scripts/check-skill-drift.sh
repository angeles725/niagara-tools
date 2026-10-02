#!/usr/bin/env bash
# check-skill-drift.sh — read-only close-gate check: does the INSTALLED skill launcher match the tracked copy?
#
# An agent follows the launcher installed under <home>/.claude/skills/<skill>/SKILL.md, not the kit's
# tracked skill/SKILL.md; an installed copy rots silently (a kit change to the tracked launcher never
# reaches it). BUILD-LOOP.md §7 runs this at close; a FAIL is fixed with
# `scripts/install-skill.sh --force` (that script installs; this one never writes).
# The skill names and tracked paths are read from install-skill.sh's SKILLS table (single source).
# [ev: retro client-source-of-truth Δ3]
#
# Usage:
#   check-skill-drift.sh [--skill <name>] [--home <dir>]
#     --skill <name>  a skill from install-skill.sh's table (default build-n4-module)
#     --home <dir>    base home directory (default $HOME); tests always pass it
#
# Row format:  PASS|FAIL  skill-drift  <skill>  <detail>
# Exit: 0 installed copy is current · 1 diverged or not installed · 2 usage (unknown skill) ·
#       3 environment (install-skill.sh, its table or the tracked launcher missing; no sha256 tool)
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALLER="$SCRIPT_DIR/install-skill.sh"

SKILL="build-n4-module"
HOME_DIR="${HOME:-}"

usage_exit() {
  printf 'usage: check-skill-drift.sh [--skill <name>] [--home <dir>]\n' >&2
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --skill) [ $# -ge 2 ] || usage_exit; SKILL="$2"; shift 2 ;;
    --home)  [ $# -ge 2 ] || usage_exit; HOME_DIR="$2"; shift 2 ;;
    -h|--help) awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "$0"; exit 0 ;;
    *) usage_exit ;;
  esac
done

[ -f "$INSTALLER" ] || { printf 'check-skill-drift: installer not found: %s\n' "$INSTALLER" >&2; exit 3; }
SKILLS="$(sed -n 's/^SKILLS="\(.*\)"$/\1/p' "$INSTALLER" | head -1)"
[ -n "$SKILLS" ] || { printf 'check-skill-drift: no SKILLS table in %s\n' "$INSTALLER" >&2; exit 3; }

TRACKED=""
for entry in $SKILLS; do
  if [ "${entry%%:*}" = "$SKILL" ]; then TRACKED="$REPO_ROOT/${entry#*:}"; fi
done
if [ -z "$TRACKED" ]; then
  printf 'check-skill-drift: unknown skill: %s\n' "$SKILL" >&2
  usage_exit
fi
[ -f "$TRACKED" ] || { printf 'check-skill-drift: tracked launcher not found: %s\n' "$TRACKED" >&2; exit 3; }
[ -n "$HOME_DIR" ] || { printf 'check-skill-drift: no home directory (pass --home)\n' >&2; exit 3; }

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    printf 'check-skill-drift: no sha256 tool found (sha256sum or shasum)\n' >&2
    exit 3
  fi
}

INSTALLED="$HOME_DIR/.claude/skills/$SKILL/SKILL.md"
REMEDY="scripts/install-skill.sh --skill $SKILL"
if [ ! -f "$INSTALLED" ]; then
  printf 'FAIL  skill-drift  %s  not installed at %s — run %s\n' "$SKILL" "$INSTALLED" "$REMEDY"
  exit 1
fi
# sha256_of runs in a subshell, so its own exit 3 cannot stop this script: check the tool here.
if ! command -v sha256sum >/dev/null 2>&1; then
  if ! command -v shasum >/dev/null 2>&1; then
    printf 'check-skill-drift: no sha256 tool found (sha256sum or shasum)\n' >&2
    exit 3
  fi
fi
tracked_sha="$(sha256_of "$TRACKED")"
installed_sha="$(sha256_of "$INSTALLED")"
if [ -z "$tracked_sha" ] || [ -z "$installed_sha" ]; then
  printf 'check-skill-drift: could not compute a digest\n' >&2
  exit 3
fi
if [ "$tracked_sha" = "$installed_sha" ]; then
  printf 'PASS  skill-drift  %s  installed copy matches %s\n' "$SKILL" "${TRACKED#"$REPO_ROOT/"}"
  exit 0
fi
printf 'FAIL  skill-drift  %s  installed copy %s differs from %s — run scripts/install-skill.sh --force --skill %s\n' \
  "$SKILL" "$INSTALLED" "${TRACKED#"$REPO_ROOT/"}" "$SKILL"
exit 1
