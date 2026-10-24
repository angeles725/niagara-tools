#!/usr/bin/env bash
# obix-link-audit.sh — declared wiring map vs the links actually present on a live component.
#
# A link from the WRONG-but-valid source (e.g. a status slot fed by a raw relay command instead
# of the current-proven running flag) is structurally fine, so bog-audit.sh CHECK7/9/11 cannot
# see it. This tool diffs the source column of docs/wiring-map.md (generate-wiring-map.sh)
# against the baja:Link children of ONE component, read over oBIX (read-only GET) or from a
# saved oBIX response.
#
# Usage:
#   obix-link-audit.sh --map <wiring-map.md> --xml <component.xml> [--table 1|2]
#   obix-link-audit.sh --map <wiring-map.md> --obix <base-url> --component <path>
#                      [--table 1|2] [--insecure]
#
#   --table 2 (default)  "SUMMARY display slots (control → facade)": the audited component is
#                        the FACADE; each facade slot must be linked from its declared RT slot.
#   --table 1            "OPERATOR config slots (facade → control)": the audited component is
#                        the CONTROL component; each RT slot must be linked from its facade slot.
#   --obix               base like https://<station>/obix; GETs <base>/config/<path>/ once (no
#                        redirect followed, HTTP >= 400 fails, 20 s bound). Credentials come from
#                        OBIX_USER / OBIX_PASS only (basic auth header, never argv, never printed).
#                        --insecure accepts a self-signed station certificate. An http:// base warns
#                        on stderr (clear-text credentials).
#
# Declared sources are split into slot (last segment) and component (the path before it;
# Rack/comp1Running, Rack.comp1Running -> slot comp1Running in Rack). MATCH needs the same slot AND,
# when the map declares a component, a link source whose component path ends in that component on
# a whole path segment (slot:/Plant/Rack2 matches Rack2 and Plant/Rack2, not Rack12): a link from
# the right slot of the wrong instance is a MISMATCH. A bare declared slot compares the slot only.
# The component check applies to Table 2 only: a Table 1 row (facade -> control) declares the facade slot and is
# compared by slot alone.
# A row whose declared column is empty or _(fill)_ is SKIP. [polish-2026-10-02 P3, #199 WU9]
#
# Rows:    STATUS  obix-link-audit  <slot>  <detail>   (MATCH | MISMATCH | MISSING | SKIP)
# Summary: obix-link-audit: M MATCH · X MISMATCH · Y MISSING · S SKIP  ->  CLEAN|ISSUES
# Links are read from the XML (go/internal/xmlsrc): every element whose display attribute is
# "Indirect: <src> -> <tgt>" or "Direct: ..." (arrow as →, &#x2192; or ->); a malformed or empty
# response is exit 3 with path:line:col, never a partial link set.
# Exit:    0 no MISMATCH/MISSING · 1 any MISMATCH/MISSING · 3 usage/env (incl. a malformed response,
#          a failed GET, or no Go toolchain / a failed build)
# Read-only: no station write; VCS-free (kit-links.bats L2).
# [ev: retro panccadia-commissioning-lessons Δ4]
#
# Implementation: the Go module at <repo>/go (go/internal/obixlinkaudit). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/obix-link-audit when the binary is missing or
# older than a Go source, then this script execs it. Needs `go` on PATH to build.
# Mutation pins (OLA-comp, OLA-http, OLA-insecure, OLA-malformed) live in
# the Go source and tests.

set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'obix-link-audit: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" obix-link-audit)" || exit 3
exec "$BIN" "$@"
