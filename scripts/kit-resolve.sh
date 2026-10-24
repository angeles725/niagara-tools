#!/usr/bin/env bash
# kit-resolve.sh — fail-closed resolution of the build-n4-module kit a session may run.
#
# The /build-n4-module launcher runs this first and uses its KIT line. A stale or dirty kit runs old
# gates silently (2026-10-24: a default checkout 821 commits behind origin/main passed a lexicon
# violation that origin/main fails). It lives in scripts/ because toolbelt/ never calls version control
# (kit-links L2). [ev: retro r10-stale-kit-resolution Δ1]
#
# Usage:
#   kit-resolve.sh [--refresh] [--offline-ok]
#     --refresh     accepted for compatibility, no longer needed: a clean kit-main strictly behind a
#                   freshly fetched origin/main is fast-forwarded automatically (see below).
#     --offline-ok  when `git fetch origin` fails, accept the local origin/main (named WARN, line says
#                   "unfetched"); without it an unreachable origin is a refusal.
#
# Candidates, first one holding METHODOLOGY.md wins (a stale winner is refused, never skipped):
#   1. $BUILD_N4_KIT                    — when set it must hold METHODOLOGY.md (else exit 3)
#   2. <kit-main>/build-n4-module-kit   — <kit-main> = $BUILD_N4_KIT_MAIN, default
#                                         $HOME/modulos_niagara_n4/niagara-tools-worktrees/kit-main
#   3. $HOME/modulos_niagara_n4/niagara-tools/build-n4-module-kit   (the default checkout)
#
# Success: stdout line 1 is exactly `KIT=<abs path> SHA=<short sha> (origin/main, clean)`; every other
# message goes to stderr. Refused when HEAD is behind origin/main or the work tree is dirty.
# Auto-refresh: when the winner is the kit-main worktree, its tree is clean, origin was fetched and HEAD
# is a strict ancestor of origin/main, it is checked out (detached) at origin/main and stderr says
# `kit-resolve: refreshed <old>→<new>`. $BUILD_N4_KIT and the default checkout are never touched.
# Any candidate whose HEAD is not an ancestor of origin/main is refused: `ahead` (behind = 0) or
# `diverged` (behind > 0) — the kit is origin/main, never unmerged commits.
# Exit: 0 resolved · 2 refused (stale, dirty, ahead, diverged, not a git work tree, origin unreachable) ·
#       3 usage or no candidate
set -u

REFRESH=0
OFFLINE_OK=0
usage_exit() { printf 'usage: kit-resolve.sh [--refresh] [--offline-ok]\n' >&2; exit 3; }
while [ $# -gt 0 ]; do
  case "$1" in
    --refresh)    REFRESH=1; shift ;;
    --offline-ok) OFFLINE_OK=1; shift ;;
    -h|--help) awk 'NR == 1 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "$0"; exit 0 ;;
    *) usage_exit ;;
  esac
done

say() { printf 'kit-resolve: %s\n' "$*" >&2; }
refuse() { say "REFUSED $*"; exit 2; }

KIT_MAIN="${BUILD_N4_KIT_MAIN:-${HOME:-}/modulos_niagara_n4/niagara-tools-worktrees/kit-main}"
DEFAULT_KIT="${HOME:-}/modulos_niagara_n4/niagara-tools/build-n4-module-kit"

# choose the candidate
KIT="" ; SOURCE=""
if [ -n "${BUILD_N4_KIT:-}" ]; then
  [ -f "$BUILD_N4_KIT/METHODOLOGY.md" ] || { say "BUILD_N4_KIT ($BUILD_N4_KIT) has no METHODOLOGY.md; fix or unset it"; exit 3; }
  KIT="$BUILD_N4_KIT"; SOURCE="env"
elif [ -f "$KIT_MAIN/build-n4-module-kit/METHODOLOGY.md" ]; then
  KIT="$KIT_MAIN/build-n4-module-kit"; SOURCE="kit-main"
elif [ -f "$DEFAULT_KIT/METHODOLOGY.md" ]; then
  KIT="$DEFAULT_KIT"; SOURCE="default"
else
  say "no kit found: tried \$BUILD_N4_KIT (unset), $KIT_MAIN/build-n4-module-kit, $DEFAULT_KIT"
  exit 3
fi
KIT="$(cd "$KIT" && pwd -P)"

TOP="$(git -C "$KIT" rev-parse --show-toplevel 2>/dev/null)" \
  || refuse "untracked: $KIT is not in a git work tree; its freshness cannot be checked"

# the remedy printed with every refusal, by candidate
case "$SOURCE" in
  kit-main) REMEDY="refresh: git -C $TOP checkout --detach origin/main" ;;
  env)      REMEDY="update $TOP to origin/main, or unset BUILD_N4_KIT to use the kit-main worktree" ;;
  *)        REMEDY="use the kit-main worktree: git -C $TOP worktree add --detach $KIT_MAIN origin/main" ;;
esac

FRESH="origin/main"
if ! GIT_TERMINAL_PROMPT=0 git -C "$TOP" fetch -q origin main >/dev/null 2>&1; then
  say "WARN offline: could not fetch origin; freshness unchecked"
  [ "$OFFLINE_OK" -eq 1 ] || refuse "offline: rerun with --offline-ok to accept the local origin/main"
  FRESH="origin/main unfetched"
fi
git -C "$TOP" rev-parse -q --verify origin/main >/dev/null || refuse "no origin/main ref in $TOP"

dirty() { [ -n "$(git -C "$TOP" status --porcelain)" ]; }
behind() { git -C "$TOP" rev-list --count HEAD..origin/main; }

short() { git -C "$TOP" rev-parse --short=7 "$1"; }
ancestor() { git -C "$TOP" merge-base --is-ancestor HEAD origin/main; }

if [ "$REFRESH" -eq 1 ] && [ "$SOURCE" != kit-main ]; then
  say "--refresh applies only to the kit-main worktree; not refreshing $TOP"
fi
# auto-refresh: kit-main only, only after a successful fetch, only clean, only a strict fast-forward
if [ "$SOURCE" = kit-main ] && [ "$FRESH" = origin/main ] && ! dirty && [ "$(behind)" -gt 0 ] && ancestor; then
  old="$(short HEAD)"
  git -C "$TOP" checkout -q --detach origin/main || refuse "refresh failed in $TOP"
  say "refreshed $old→$(short HEAD)"
fi

if dirty; then
  refuse "dirty: $TOP has uncommitted changes ($(git -C "$TOP" status --porcelain | wc -l | tr -d ' ') path(s)); clean it first — $REMEDY"
fi
n="$(behind)"
# any candidate: HEAD must be origin/main or behind it — unmerged commits are never served as the kit
if ! ancestor; then
  [ "$n" -eq 0 ] && refuse "ahead: $TOP HEAD $(short HEAD) has commits not on origin/main ($(short origin/main)) — $REMEDY"
  refuse "diverged: $TOP HEAD $(short HEAD) has commits not on origin/main ($(short origin/main)) and is $n commit(s) behind — $REMEDY"
fi
[ "$n" -eq 0 ] || refuse "stale: $TOP is $n commit(s) behind origin/main — $REMEDY"

printf 'KIT=%s SHA=%s (%s, clean)\n' "$KIT" "$(git -C "$TOP" rev-parse --short HEAD)" "$FRESH"
