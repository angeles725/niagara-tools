#!/usr/bin/env bash
# kit-ticket.sh — Opens a kit defect issue (or writes the offline fallback when gh is absent).
#
# Usage: kit-ticket.sh "<one-line title>" [--from <retro-file>] [--repo <owner/repo>]
#
# Resolves the kit root from $KIT (env, when a directory) → this script's ../ (unresolved) → cwd.
#
# Behaviour:
#   With gh available: gh issue create [--repo <repo>] --title "[kit] <title>"
#                      --label kit,from-run,campaign-9 --body-file <tmp>
#   Without gh:        print SKIP row; write retros/tickets/<date>-<slug>.md; exit 0
#                      (never fails a run — gh absence is a known offline situation)
#
# Output row (SKIP path):
#   ticket  SKIP  <retro-file-or-title>  gh absent; wrote retros/tickets/<date>-<slug>.md
#
# gh present but failing: the ticket is ALWAYS kept at retros/tickets/<date>-<slug>.md and its path printed.
#   no known host / no auth / outside a repository / no base repo -> `ticket  LOCAL  …` row, exit 4;
#   any other gh failure -> `ticket  FAIL  …` row, exit 3 (retro deltas 1928).
#
# Exit codes: 0 ok (including gh absent) · 3 usage/env/other gh failure (K20), including no Go toolchain or a
#   failed build · 4 gh no host/auth/repo, kept locally
#
# Implementation: go/internal/kitticket (goldens recorded from the bash script at 5179bf23 by
# go/internal/lint/rules/golden/record/record.sh). go/go-tool.sh `go_tool_bin` builds go/bin/kit-ticket when it
# is missing or older than a Go source, then this script execs it with KIT_TICKET_DEFAULT_ROOT = its ../ and
# KIT_TICKET_SCRIPT = its own path (the prefix of the shell-style error messages the bash printed).
# The wrapper needs dirname, readlink and find on PATH (go too when the binary must be rebuilt).
# VCS-free by design; kit-links.bats L2 enforces.
# [ev: retro campaign8-retro-loop] [ev: retro go-kit-ticket]
set -u
# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'kit-ticket: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" kit-ticket)" || exit 3
# The default kit root stays the UNRESOLVED ../ of this file, as the bash script's BASH_SOURCE parent was.
KIT_TICKET_DEFAULT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)" KIT_TICKET_SCRIPT="${BASH_SOURCE[0]}" exec "$BIN" "$@"
