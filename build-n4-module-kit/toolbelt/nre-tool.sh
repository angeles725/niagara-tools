#!/usr/bin/env bash
# nre-tool.sh — run a Niagara install's build-time nre tools by name (issue #851), no station and no Workbench needed.
#
# Usage: nre-tool.sh deps [--raw] <niagara_home> <module>
#   deps   the module parts <module> depends on, one per line, from the install's registry (read-only):
#          nre devkit:com.tridium.devkit.util.DependencyChecker <module>. --raw prints nre's whole output instead.
#   A Windows install is run from WSL through its bin/nre.exe; the runtime boots from the install (about 35 s).
# exit:  0 listed · 1 the install holds no such module · 3 usage/env (no bin/nre.exe or bin/nre, a bad module name)
#
# Implementation: the Go module at <repo>/go (go/internal/nretool); go/go-tool.sh builds go/bin/nre-tool on demand.
# This script is VCS-free by design — version control is never invoked (kit-links L2).
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'nre-tool: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" nre-tool)" || exit 3
exec "$BIN" "$@"
