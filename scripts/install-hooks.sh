#!/usr/bin/env bash
# install-hooks.sh — opt-in activation of the kit's git hooks.
# Points git at the kit's hooks: `git config --local core.hooksPath .githooks`, which activates every
# hook there: the retro-enforcement pre-push gate, and post-merge / post-checkout / post-rewrite /
# post-commit, which regenerate the kit's generated state (BUILD-STATE.md, retros/INDEX.md,
# toolbelt/INDEX.md — not committed, R10-D1) with toolbelt/kit-state.sh after every pull, checkout,
# rebase/amend and commit. The install runs post-merge once
# so the state is materialized right away. The hooks ship INERT; this is what makes them live in a
# clone. Per-clone and reversible. [ev: retro ledger-generated-not-committed]
#
# Usage:
#   scripts/install-hooks.sh              # activate (set core.hooksPath = .githooks)
#   scripts/install-hooks.sh --force      # activate even over a pre-existing CUSTOM hooksPath
#   scripts/install-hooks.sh --uninstall  # deactivate (unset the core.hooksPath we set)
#
# git IS used here on purpose: this is NOT a toolbelt/ check script, so the kit-links L2 rule
# ("no toolbelt script invokes git") does not apply — the installer lives under scripts/.
set -euo pipefail

HOOKS_DIR=".githooks"
MODE="install"
FORCE=0
for arg in "$@"; do
  case "$arg" in
    --uninstall) MODE="uninstall" ;;
    --force)     FORCE=1 ;;
    -h|--help)   printf 'Usage: install-hooks.sh [--force] [--uninstall]\n'; exit 0 ;;
    *) printf 'install-hooks: unknown arg: %s\n' "$arg" >&2; exit 2 ;;
  esac
done

# must be inside a git work tree
git rev-parse --git-dir > /dev/null 2>&1 || { printf 'install-hooks: not a git repository\n' >&2; exit 2; }

current="$(git config --local --get core.hooksPath 2>/dev/null || true)"

if [ "$MODE" = "uninstall" ]; then
  if [ "$current" = "$HOOKS_DIR" ]; then
    git config --local --unset core.hooksPath
    printf 'install-hooks: deactivated — core.hooksPath unset (default hooks restored).\n'
  else
    printf 'install-hooks: core.hooksPath is not %s (it is "%s"); left untouched.\n' "$HOOKS_DIR" "$current"
  fi
  exit 0
fi

# install — never silently clobber a user's own hooks configuration
if [ -n "$current" ] && [ "$current" != "$HOOKS_DIR" ] && [ "$FORCE" -eq 0 ]; then
  printf 'install-hooks: REFUSING — core.hooksPath is already set to a custom value "%s".\n' "$current" >&2
  printf '              Your hooks are left untouched. Re-run with --force to override.\n' >&2
  exit 3
fi

git config --local core.hooksPath "$HOOKS_DIR"
printf 'install-hooks: activated — core.hooksPath = %s; live: the retro-enforcement pre-push gate and the\n' "$HOOKS_DIR"
printf '              kit-state regenerate hooks (post-merge, post-checkout, post-rewrite, post-commit).\n'
# materialize the generated kit state now (the hook never fails; it warns and removes stale files)
top="$(git rev-parse --show-toplevel)"
if [ -x "$top/$HOOKS_DIR/post-merge" ]; then (cd "$top" && "./$HOOKS_DIR/post-merge"); fi
printf '              Undo with: scripts/install-hooks.sh --uninstall\n'
exit 0
