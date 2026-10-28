#!/usr/bin/env bash
# ledger-check.sh — checks a client's VERSION-LEDGER.md against the kit's templates/VERSION-LEDGER.md.
#
# The per-client version ledger (`odd/VERSION-LEDGER.md` in the client repository; BUILD-LOOP.md §0 reads it before
# the first write, §7 gates a feature close on its rows) is the one place a repeated root-cause class shows up; a row
# with a free-text status, an undeclared class or a defect with no class silently drops out of that count. The
# contract is read from the template, never restated: its title, its `##` sections, the Versions header and the
# Status vocabulary. Checks:
#   LC1  the first non-empty line is not a `# VERSION-LEDGER` title (FAIL)
#   LC2  a `##` section of the template is missing (FAIL)
#   LC3  the Versions table header is not the template's, cell for cell (FAIL)
#   LC4  a Versions row with another cell count, or an empty Module or Version cell (FAIL)
#   LC5  a Status outside the template's vocabulary (FAIL)
#   LC6  a Date that is not a real YYYY-MM-DD date (FAIL)
#   LC7  a Class that is neither — nor a class of the ledger's own Root-cause classes table (FAIL)
#   LC8  a Defect found with Root cause or Class left — (FAIL)
#   LC9  a template placeholder (<...>) left in a Versions row (WARN)
# An empty Versions table (a freshly scaffolded ledger) is clean.
# (the 2026-09-26 version-defect-ledger retro, Δ1)
#
# Usage:  ledger-check.sh [--template <VERSION-LEDGER template>] <VERSION-LEDGER.md>
#   Row:  FAIL|WARN  ledger-check  <file>[:<line>]  LC<n>: <reason>
#   Exit: 0  clean or WARN-only · 1  any FAIL · 3  usage, an unreadable ledger, or a template that cannot be read or
#         holds no Versions table or Status sentence
# Implementation: go/cmd/ledger-check + go/internal/ledgercheck. This wrapper is the stable entry point:
# go/go-tool.sh `go_tool_bin` builds go/bin/ledger-check when it is missing or older than a Go source, then this
# script execs it with LEDGER_CHECK_TEMPLATE = this kit's templates/VERSION-LEDGER.md. Needs `go` on PATH to build
# (exit 3 otherwise). VCS-free by design.
# Mutation: LCB1 -- dropping the status check passes a row whose status is free text
set -u
# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'ledger-check: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" ledger-check)" || exit 3
LEDGER_CHECK_TEMPLATE="$here/../templates/VERSION-LEDGER.md" exec "$BIN" "$@"
