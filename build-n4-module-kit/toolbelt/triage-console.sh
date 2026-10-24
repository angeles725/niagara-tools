#!/usr/bin/env bash
# triage-console.sh — own-module exception triage over station console logs (Campaign 8 PR2).
# Three attribution channels:
#   C1  own frame   — stack line at <PKG>. under an exception
#   C2  own logger  — WARNING/SEVERE with a tag in the own-tag WHITELIST (stack-less ok);
#                     also: message contains the own-package path (e.g. [loader] cert-chain)
#   C3  bog-drift   — [sys] fatal exception, [sys.xml] slot-drift, [sys.registry] Missing class
# --site: a WARNING/SEVERE block attributed to NO own channel is classified into a separate
#   site-issues list — device-offline, duplicate-device-id, comm-timeout, history-flood — one row
#   per class + digit-normalized message, with count, first/last seen and distinct raw messages:
#     SITE  triage-console  <subject>  <class>  <N>x <first> -> <last>  <LEVEL> [tag] <msg> (distinct K)
#   SITE rows print after the own rows, count descending, and never change the exit code.
#   An unclassified foreign warning is in neither list (unknown stays unknown).
# Usage: triage-console.sh [--package PKG] [--tag TAG_LIST] [--module-prefix PFX_LIST]
#                          [--site] [--console-dir DIR] <console.txt|dir>...
# Exit: 0 no own rows · 1 any own row · 3 usage/env (an option without its value, an unreadable console,
#   no Go toolchain or a failed build)
#
# Implementation: go/internal/triageconsole (goldens recorded from this script at 79ca5a2c by
# go/internal/lint/rules/golden/record/record.sh). go/go-tool.sh `go_tool_bin` builds go/bin/triage-console when
# it is missing or older than a Go source, then this script execs it with the default lists below.
# [ev: corpus B800]  [ev: retro campaign8-triage-console]
# [ev: retro site-fault-triage-and-incident-journal Δ1]
set -u
# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'triage-console: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" triage-console)" || exit 3

# The defaults of --package, --tag and --module-prefix (the Go test reads them from these three lines).
PKG="com.angeles"
OWN_TAGS="coldRoomPan,dashboardPan,dashboardpan,compPan,chihuahua"
MOD_PREFIX="ColdRoomPan,DashboardPan,CompPan,chihuahua"

TRIAGE_CONSOLE_DEFAULT_PACKAGE="$PKG" TRIAGE_CONSOLE_DEFAULT_TAGS="$OWN_TAGS" TRIAGE_CONSOLE_DEFAULT_MODULE_PREFIX="$MOD_PREFIX" \
  exec "$BIN" "$@"
