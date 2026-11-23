#!/usr/bin/env bash
# retro-issues.sh — retro deltas are GitHub issues (CONTRIBUTING.md § 6.3): file the mcp-n4 / build-n4-module deltas of
# retros as issues, scrubbed against the private client denylist, deduplicated by marker and by meaning.
#
# Usage: retro-issues.sh [--apply [--write-back]] [--area mcp-n4|build-n4-module] [--alias <stem>=<ALIAS>]...
#                        [--only <id>[,<id>...]]... [--new <id>[,<id>...]]... [--dup-threshold <0..1>] <retro.md>...
#        retro-issues.sh --check            list the retro-delta issues still open (exit 1 when any)
#   Default is a dry run: it prints the planned issues and calls only read-only gh commands.
#   With no denylist the dry run sends no search term to GitHub (dedup uses the markers only); --apply needs one.
#   A retro outside the kit needs --alias <stem>=<ALIAS> so its stem never reaches GitHub.
# exit:  0 planned / done · 1 an issue refused by the scrub, a gh failure under --apply, or open issues under --check · 3 usage/env
#
# Implementation: the Go module at <repo>/go (go/internal/retroissues); go/go-tool.sh builds go/bin/retro-issues on demand.
# This script is VCS-free by design — version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'retro-issues: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" retro-issues)" || exit 3
RETRO_ISSUES_KIT_ROOT="$(cd "$here/.." && pwd)" exec "$BIN" "$@"
