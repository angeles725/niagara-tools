#!/usr/bin/env bash
# merge-when-green.sh — wait for the CI checks of a pull request's exact head commit, then merge that commit.
#
# Usage: scripts/merge-when-green.sh <pr> [--rebase|--squash|--merge] [--repo <owner/name>]
#                                    [--interval <duration>] [--timeout <duration>] [--log-lines <n>]
#
# The rules (go/internal/mergewait): merge only on green checks of the exact head SHA, through
# `gh pr merge --match-head-commit <sha>`; the branch must contain the current base branch (rebase or update
# it first); a draft is never green; a check that failed on a transient download error is re-run once; a
# failed check is reported with the last lines of its log. Exit: 0 merged; 1 a check failed; 2 usage;
# 3 gh missing, a gh call failed, or the Go tool could not be built; 4 draft; 5 the branch lacks the current
# base; 6 not open or the head moved; 7 timed out; 8 gh pr merge refused. (retro 2026-11-15 coordination-stretch C5)
#
# A launcher only (CONTRIBUTING.md § 2.2): go/go-tool.sh builds go/bin/merge-when-green when it is missing or
# older than a Go source, then this execs it. It runs under bash and under zsh (`zsh scripts/merge-when-green.sh`):
# the session shell of the agents that merge is zsh.
self="${BASH_SOURCE[0]:-$0}"
ROOT="$(cd "$(dirname "$self")/.." && pwd)" || exit 3
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via ROOT
. "$ROOT/go/go-tool.sh" || exit 3
bin="$(go_tool_bin "$ROOT/go" merge-when-green)" || exit 3
exec "$bin" "$@"
