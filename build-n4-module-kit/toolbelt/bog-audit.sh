#!/usr/bin/env bash
# bog-audit.sh — Niagara station config.bog auditor (Campaign 8 PR10, B810/B816)
# [ev: retro campaign8-bog-audit]
#
# Audits a station config.bog against the deployed module source.  The bog
# is a zip containing file.xml; the script also accepts a bare .xml path.
#
# Usage:
#   bog-audit.sh <config.bog|file.xml> --module <MOD> [--module <MOD>]...
#                [--source-dir <DIR>] [--strict]
#
# Row format: <CHECK_ID>  PASS|FAIL|WARN|SKIP  <component-path>  <detail>
# Exits:      0 clean · 1 any FAIL · 3 usage, not a file, no file.xml in a zip, a malformed bog
#             (xmlsrc path:line:col) or no component of the named modules
#
# Checks that run from the bog alone (no --source-dir needed):
#   CHECK1  inventory              INFO  component count per own module
#   CHECK8  hoa-leftover           WARN  writable with active manual override
#   CHECK9  orphan-handle          FAIL  link sourceOrd not in any component
#   CHECK10 duplicate-handle       FAIL  same handle on multiple components
#   CHECK11 proxy-link-safety      FAIL  own-module output -> writable, no fallback  [ev: corpus B810]
#   CHECK12 dashboard-write-link   WARN  servlet-written slot is also a link target (dashboard-write-to-LINK_TARGET; needs --source-dir)  [ev: corpus B816]
#   CHECK13-19 station logic        relay-double-source FAIL · own-output-unlinked WARN · sensor-crossed-by-name WARN ·
#                                   hasDefrost<->DefrostController FAIL · roomN-index FAIL · tile-number FAIL · link-direction WARN
#   CHECK20 numeric-to-bool-direct FAIL  Mode/Command/Cmd slot -> BOrLogic/BAndLogic/BNotLogic, no Equal() intermediate
#           Limitation: name heuristic only — type reflection not available from bog XML.
#                       False negative when the numeric source slot does not carry Mode/Command/Cmd in its name.
#
# Checks that need --source-dir (emits SKIP rows without it):
#   CHECK2  action-flag-drift      WARN  (--strict -> FAIL)
#   CHECK3  out-of-facet           WARN  value below source MIN facet
#   CHECK4  transient-persisted    WARN  TRANSIENT slot has value in bog
#   CHECK5  schema-drift-bog-extra FAIL  slot in bog not in source class
#   CHECK6  src-missing            WARN  slot in source not in bog
#   CHECK7  link-dangling          FAIL  link targetSlotName not in source class
#
# Dot-directories are excluded when walking --source-dir (D9b).
# The last stdout line is `# bog-audit: parse time <s>s`.
# This script is VCS-free by design (kit-links.bats L2).
#
# Implementation: go/internal/bogaudit (command go/cmd/bog-audit; go/README.md). The bog is read once by the shared
# XML scanner go/internal/xmlsrc (a zipped .bog through its file.xml member; R8-D12) and replayed through the state
# machine of the python line scanner it replaced; python3 is no longer needed. This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/bog-audit when it is missing or older than a Go source, then
# this script execs it. Needs `go` on PATH to build (exit 3 otherwise). [ev: retro go-bog-audit]
#
# [ev: corpus B810] [ev: corpus B816]
set -u
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'bog-audit: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" bog-audit)" || exit 3
exec "$BIN" "$@"
