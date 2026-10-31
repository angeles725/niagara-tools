#!/usr/bin/env bash
# lint-port-parity.sh — a port of a reference frontend lacks no reference file, DOM id or named function
#   without a waiver that records the user's decision (a port once dropped a whole equipment view unseen).
#
# A module that ports a reference frontend (a viewer, a prototype, an earlier dashboard) declares it in
# <module-root>/.port-parity, the module's parity file:
#   ref: <reference dir>             required; relative to the file's directory, or absolute
#   port: <port dir>                 optional; default the module root
#   <file|id|function> <name> -- <reason> [decision: <ref, e.g. W5 / user 2026-10-13>]
#   rename: <reference file> -> <port file> [-- <reason>]
#   rename: <function|id> <ref name> -> <port name> -- <reason>   (behaviour kept under another name; the port
#           must have <port name> as the same kind; several renames may share one target; no decision needed)
# Items per tree (.js .mjs .html .css; vendor/, node_modules/, data/, dot-dirs and *.min.js skipped): file
# basenames, DOM ids (id="x", also inside JS strings) and named functions (function f, const f = ... =>).
#
# Rows:
#   FAIL  lint-port-parity  <ref-file>:<line>  missing <kind> <name> (in the reference, not in the port, no waiver)
#   FAIL  lint-port-parity  .port-parity:<n>  waiver <kind> <name> has no [decision: <ref>] (...)
#   FAIL  lint-port-parity  .port-parity:<n>  malformed waiver (...) · rename target <f> is not a file of the port
#   INFO  lint-port-parity  <ref-file>:<line>  <kind> <name> waived via file <f> (defined only in that reference file):
#         a `file` waiver with a decision also waives every id and function only that reference file defines; a
#         name another, non-waived reference file also defines is still a FAIL row. INFO rows never fail.
#   INFO  lint-port-parity  <ref-file>:<line>  <kind> <name> renamed to <port name> (<reason>)
#   FAIL  lint-port-parity  .port-parity:<n>  rename target <kind> <name> is not in the port · rename ... has no -- <reason>
#   WARN  lint-port-parity  .port-parity:<n>  stale rename: <kind> <name> is still in the port (remove the line)
#   WARN  lint-port-parity  .port-parity:<n>  stale waiver: <kind> <name> matches no missing item (remove the line)
# then one `port-parity: <n> files, <n> ids, <n> functions in the reference; ...` summary line (not a row).
# Mutation: PP-drop-waiver -- a gate that ignores missing items stays exit 0 when one waiver line is removed
# Mutation: PP-no-decision -- accepting a reason alone lets a reference feature drop with no recorded decision
# Mutation: PP-cascade-shared -- cascading a name another reference file also defines hides that file's drop
# Mutation: PP-rename-target -- accepting a rename whose target is absent hides a dropped function or id
# Mutation: PP-drop-file-waiver -- a cascade that does not need the file waiver stays exit 0 when the file line is removed
#
# Usage: lint-port-parity.sh [--format text|json] <module-root>
# Usage: lint-port-parity.sh --ref <dir> --port <dir> [--waivers <file>] [--format text|json]
# Exit:  0 clean (every missing item waived with a decision; WARN rows allowed) · 1 any FAIL · 3 usage/IO, an
#        empty tree, or a module with no .port-parity (report-module SKIPs that one)
# [ev: retro reference-port-parity Δ1]
#
# Implementation: the Go module at <repo>/go (go/internal/portparity, go/cmd/port-parity). This wrapper is the
# stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/port-parity when the binary is missing or older
# than a Go source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a failed build is
# exit 3.
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-port-parity: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" port-parity)" || exit 3
exec "$BIN" "$@"
