#!/usr/bin/env bash
# lint-bacnet-status-flags.sh — WARNs BACnet Present_Value proxy points whose facets lack statusFlags=true, per device.
#
# BBacnetProxyExt.useStatusFlags() is true only for a Present_Value proxy whose point facets carry statusFlags=true
# (vineflower bacnet-rt 4.14.0.162 BBacnetProxyExt.java:808-822): without the facet a point in alarm, fault or
# override in the controller reads {ok} in Niagara, polled or COV alike (types/driver-authoring.md § 2). Reads every
# *.bog under each station directory (a zipped config.bog through its file.xml member, go/internal/xmlsrc). A proxy
# is a Bacnet…ProxyExt of the bacnet module (its module prefix resolved through the bog's m= declarations); it reads
# Present_Value when its propertyId slot is absent (the default) or holds 85; a useStatusFlags=true slot on the ext
# also counts as set. One row per device (the parent of the point's …PointDeviceExt, a vendor device type too) with
# the count and the first point. Fix: add statusFlags=b:true to each point's facets (one extra Status_Flags read
# per poll). [ev: retro devguide415-r2-drivers-protocols Δ4] [ev: retro devguide415-r3-drivers-integration Δ3]
#
# Implementation: rule package go/internal/lint/rules/bacnetstatusflags on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-bacnet-status-flags <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-bacnet-status-flags.sh [--strict] <station-dir>...
#   Row:  WARN  lint-bacnet-status-flags  <bog>: <device>: <n> Present_Value proxy point(s) without facets statusFlags=true (first: <point>) -- a field alarm, fault or override reads {ok}
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.bog, a malformed or unreadable bog
# Mutation: BSF1 -- ignoring the statusFlags facet WARNs the fixture's FanStatus point too (count 2, not 1)
# Mutation: BSF2 -- matching the literal bacnet: prefix (no m= alias resolution) passes the fixture station
# Mutation: BSF3 -- treating every propertyId as Present_Value counts the fixture's Units proxy (property 117)
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-bacnet-status-flags: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-bacnet-status-flags "$@"
