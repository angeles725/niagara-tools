#!/usr/bin/env bash
# module-map.sh — add a concern to a module, find a file's concern, run one concern's pure tests.
#
# Usage:
#   module-map.sh add-concern <Name> --root <module-root> [--part rt|ux|wb|se] [--adapter]
#       <Name> is a Java identifier starting uppercase (Battery); the concern is its lowercase (battery). Creates
#       <part>/src/<pkg>/<role>/<concern>/<Name>Logic.java (pure, no framework import) and its <Name>LogicTest under
#       srcTest (srcTest/test/ when the part has one), under the part's role folder (rt, wb, se model/; ux route/),
#       an rt <Name>Adapter stub under adapter/<concern>/ with --adapter, and inserts the concern's MODULE-MAP.md
#       row after the table's last row. An existing row, concern folder or test file is refused (exit 1) and
#       nothing is written. Also reachable as `scaffold-module.sh add-concern ...`.
#   module-map.sh where <file|concern> --root <module-root>
#       Prints the concern (by name, or the row whose Files or Tests list the file, a glob included), its part,
#       files, tests and "change here when".
#   module-map.sh tests <concern> --root <module-root>
#       Runs each pure JUnit test of the concern's Tests cell through run-pure-test.sh <part-dir> <fqcn>; a test
#       importing javax.baja (a station test) and a JS spec are printed as skip rows with the reason.
#   Rows: created <path> · mapped MODULE-MAP.md:<line> <concern> · run <path> <fqcn> · skip <path> <reason>
#   Exit: 0 ok · 1 refused (add-concern), not listed (where, tests) or a test failed · 2 usage · 3 env (no or
#         invalid MODULE-MAP.md, no such part, a write error, a runner env error)
# [ev: retro modularity-structure Δ1] [ev: retro modularity-structure Δ4]
#
# Implementation: go/internal/modulemap (map parser shared with lint-module-map). go/go-tool.sh `go_tool_bin`
# builds go/bin/module-map when it is missing or older than a Go source, then this script execs it with
# MODULE_MAP_TOOLBELT = this directory. Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design; kit-links.bats L2 enforces.
set -u
# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'module-map: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" module-map)" || exit 3
MODULE_MAP_TOOLBELT="$here" exec "$BIN" "$@"
