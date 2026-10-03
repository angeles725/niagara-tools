#!/usr/bin/env bash
# install-client-hooks.sh — install the kit's split-package pre-push gate into a CLIENT repository (issue #142).
#
# Copies build-n4-module-kit/templates/client-pre-push to <client-repo>/.githooks/pre-push, sets that clone's
# `core.hooksPath = .githooks` and records this kit's toolbelt in the clone's LOCAL config
# (`n4kit.toolbelt`, never committed). Commit .githooks/pre-push in the client repo so every clone carries it;
# each new clone runs this installer once to activate it.
#
# Usage:  scripts/install-client-hooks.sh [--force] <client-repo>
#   --force  replace a DIFFERENT existing .githooks/pre-push and a custom core.hooksPath
# Exit: 0 installed (or already installed: idempotent) · 2 usage, or <client-repo> is not a git work tree ·
#       3 refused: a different pre-push or a custom core.hooksPath exists (re-run with --force)
#
# git IS used here on purpose: this lives under scripts/, not toolbelt/ (kit-links L2).
set -euo pipefail

usage() { printf 'usage: install-client-hooks.sh [--force] <client-repo>\n' >&2; exit 2; }
FORCE=0
REPO_ARG=""
for arg in "$@"; do
  case "$arg" in
    --force) FORCE=1 ;;
    -h|--help) printf 'usage: install-client-hooks.sh [--force] <client-repo>\n'; exit 0 ;;
    -*) printf 'install-client-hooks: unknown option: %s\n' "$arg" >&2; usage ;;
    *) [ -z "$REPO_ARG" ] || usage; REPO_ARG="$arg" ;;
  esac
done
[ -n "$REPO_ARG" ] || usage

KIT_TOOLBELT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../build-n4-module-kit/toolbelt" && pwd)"
TEMPLATE="$KIT_TOOLBELT/../templates/client-pre-push"
[ -f "$TEMPLATE" ] || { printf 'install-client-hooks: template missing: %s\n' "$TEMPLATE" >&2; exit 2; }

[ -d "$REPO_ARG" ] || { printf 'install-client-hooks: not a directory: %s\n' "$REPO_ARG" >&2; exit 2; }
if [ "$(git -C "$REPO_ARG" rev-parse --is-inside-work-tree 2>/dev/null || true)" != "true" ]; then
  printf 'install-client-hooks: not a git work tree: %s\n' "$REPO_ARG" >&2
  exit 2
fi
TOP="$(git -C "$REPO_ARG" rev-parse --show-toplevel)"
HOOK="$TOP/.githooks/pre-push"

current="$(git -C "$TOP" config --local --get core.hooksPath 2>/dev/null || true)"
if [ -n "$current" ] && [ "$current" != ".githooks" ] && [ "$FORCE" -eq 0 ]; then
  printf 'install-client-hooks: REFUSING — core.hooksPath is already "%s"; nothing changed. Re-run with --force.\n' "$current" >&2
  exit 3
fi
if [ -e "$HOOK" ] && ! cmp -s "$TEMPLATE" "$HOOK" && [ "$FORCE" -eq 0 ]; then
  printf 'install-client-hooks: REFUSING — %s exists and differs from the kit template; nothing changed. Re-run with --force.\n' "$HOOK" >&2
  exit 3
fi

mkdir -p "$TOP/.githooks"
if ! cmp -s "$TEMPLATE" "$HOOK" 2>/dev/null; then
  cp "$TEMPLATE" "$HOOK"
fi
chmod +x "$HOOK"
git -C "$TOP" config --local core.hooksPath .githooks
git -C "$TOP" config --local n4kit.toolbelt "$KIT_TOOLBELT"
printf 'install-client-hooks: %s/.githooks/pre-push active (split-package-check --strict on every push).\n' "$TOP"
printf '                      Commit .githooks/pre-push; each new clone re-runs this installer once.\n'
