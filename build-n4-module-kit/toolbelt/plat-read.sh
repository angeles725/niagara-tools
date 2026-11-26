#!/usr/bin/env bash
# plat-read.sh — run one read-only plat verb against a Niagara platform daemon (issue #850).
#
# Usage: plat-read.sh [--env FILE] [--home DIR] [--dry-run] details|liststations|ipconfig
#   Runs <home>/bin/plat.exe (or bin/plat) <verb> -h:ip:<host> -p:<port> -secure -noinput with the daemon user and
#   password of a mode-0600 env file (NIAGARA_PLAT_HOST, NIAGARA_PLAT_PORT (default 5011), NIAGARA_PLAT_USER,
#   NIAGARA_PLAT_PASSWORD; --env, NIAGARA_PLAT_ENV or ~/.config/niagara-private/lab-platform.env). The password is
#   masked in everything printed; the NRE boot log and module permission blocks are dropped. Verbs that change the
#   host or a station (startstation, stopstation, reboothost, fput, settime, tellstation...) are refused.
#   --dry-run prints the plat call (password masked) and runs nothing. home: --home or NIAGARA_HOME.
# exit:  plat's exit code · 1 when plat printed a Java exception (it exits 0 then, e.g. a wrong password) · 3 usage/env
#
# Implementation: the Go module at <repo>/go (go/internal/platread); go/go-tool.sh builds go/bin/plat-read.
# This script is VCS-free by design — version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'plat-read: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" plat-read)" || exit 3
exec "$BIN" "$@"
