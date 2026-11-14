#!/usr/bin/env bash
# check-bats-skips.sh — the global bats SKIP allowlist assertion (audit A6; CI bats job).
#
# A bats test that skips reads green. A suite that starts skipping for a NEW reason — a tool install that
# broke, a fixture that went missing, a guard that now always trips — passes CI unseen. This check reads the
# TAP of a bats run and fails when any `# skip <reason>` matches no line of the allowlist.
#
# Usage:  check-bats-skips.sh <bats.tap> <allowlist>
#   allowlist: one POSIX extended regular expression per line, matched against the WHOLE reason;
#   `#` lines and blank lines are ignored; a skip with no reason is the empty string. The committed list is
#   tests/bats-skip-allowlist.txt — add a line only for a skip that is legitimate on a CI runner, with a
#   comment saying why.
# Rows:   UNLISTED skip: <ok N test name> — <reason>   (one per unlisted skip, stdout)
# Exit:   0 every skip is allowlisted · 1 an unlisted skip · 3 usage, an unreadable TAP or allowlist, a TAP
#         with no `1..N` plan (bats never ran), or an allowlist line that is not a valid ERE
#
# Implementation: the Go module at <repo>/go (go/internal/checkbatsskips). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/check-bats-skips when the binary is missing or older than a
# Go source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3.
# The patterns are Go POSIX EREs: a GNU grep extension (a back-reference, \w, \<) is an invalid line, exit 3.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'check-bats-skips: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" check-bats-skips)" || exit 3
exec "$BIN" "$@"
