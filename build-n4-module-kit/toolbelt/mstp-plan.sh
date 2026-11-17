#!/usr/bin/env bash
# mstp-plan.sh — plans BACnet MS/TP addresses set by DIP switches: per device its MAC, the switches to
# turn ON (switch n has weight 2^(n-1)) and its device ID (id-base + MAC), refusing a MAC that is
# reserved (0-3 by default: the JACE and spare masters), above maxMaster, beyond what the switches can
# set, or already used on the trunk (--existing CSV with a `mac,device_id` header). [ev: retro mstp-plan Δ1]
#
# Implementation: the Go module at <repo>/go (go/internal/mstpplan); go/go-tool.sh builds go/bin/mstp-plan
# on demand, then this script execs it.
#
# Usage:  mstp-plan.sh --first-mac N --count N [--dip 7] [--id-base 2000] [--max-master 127]
#           [--reserved-max 3] [--label Device] [--label-start 1] [--existing file.csv] [--format md|csv]
# Output: a markdown (or CSV) table Label | MAC | Switches ON | Device ID.
# Exit:   0 plan OK · 1 conflict (every one listed on stderr, no table) · 2 usage · 3 no Go toolchain
set -u
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'mstp-plan: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" mstp-plan)" || exit 3
exec "$BIN" "$@"
