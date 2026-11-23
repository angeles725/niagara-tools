#!/usr/bin/env bash
# lab-station.sh — start, stop and restart a local lab station from a script, no interactive console (issue #848).
#
# Usage: lab-station.sh start|stop|restart|status [--home DIR] [--state DIR] [--timeout SECONDS] [--dry-run] <station>
#        lab-station.sh console <station> gc|heap|help|threads|version
#   start    runs <home>/bin/station.exe (or bin/station) <station> with its console on a named pipe; waits for
#            "*** Station Started" (default timeout 180 s). The log is <state>/<station>/out.log.
#   stop     writes quit to the console pipe and waits for "*** Station shutdown ***" (an orderly shutdown, no kill).
#   restart  stop when running, then start. status: running or stopped. console: one read-only console verb.
#   --dry-run prints the plan and touches nothing. home: --home or NIAGARA_HOME; state: --state, LAB_STATION_STATE
#   or $TMPDIR/lab-station. Only a station started by lab-station can be stopped by it.
# exit:  0 done · 1 failed (did not start or stop, not running) · 3 usage/env
#
# Implementation: the Go module at <repo>/go (go/internal/labstation); go/go-tool.sh builds go/bin/lab-station.
# This script is VCS-free by design — version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lab-station: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" lab-station)" || exit 3
exec "$BIN" "$@"
