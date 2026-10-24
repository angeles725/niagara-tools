#!/usr/bin/env bash
# lint-agent-on-shape.sh — every module-include.xml <on type="..."/> must be module:Type shaped.
#
# An agent-on binding whose target omits the module prefix (e.g. <on type="ReflowService"/>
# instead of <on type="nmodsreflow:ReflowService"/>), or is otherwise malformed, makes the
# agent silently fail to appear in Workbench — no build error. The value must be exactly
# `module:Type` (one colon, non-empty parts). See types/module-wiring.md §3.
# [ev: retro 2026-09-19-module-wiring]
#
# Usage:  lint-agent-on-shape.sh <src-root>     (finds module-include.xml under it)
#   Row:  FAIL  lint-agent-on-shape  <file>:<line>  malformed agent-on target (need module:Type): <value>
#   Exit: 0  clean · 1  any FAIL · 3  usage/env or a sub-directory find cannot enter
# Read as XML (go/internal/xmlsrc, R8-D12; truth fix 2026-10-24 over the line-based awk): every <on> element's
# type attribute, entity-decoded, single or double quoted, at the element's own line; an <on> inside a comment or
# CDATA is not one; an empty type="" is malformed too; a malformed module-include.xml is
# `lint-agent-on-shape: <file>:<line>:<col>: <reason>` on stderr (exit 3; the other files are
# still scanned). The awk took only the last <on per line and read a malformed file as text. Real trees unchanged.
# Empty root: a root with no module-include.xml is an env error (exit 3) with
# `ERROR  lint-agent-on-shape  <root>  no files to scan (K20: wrong path or empty scaffold)` on stdout.
# Implementation: rule package go/internal/lint/rules/agentonshape on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs `n4lint lint-agent-on-shape
# <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design.
# Mutation: AOS2 -- relax the module:Type regex and AOS2 stops failing on a missing module prefix
# Mutation: AOS-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
# Mutation: AOS-xml -- reading lines instead of XML counts a commented-out <on> and misses a second <on> on its line
# Mutation: AOS-unreadable -- ignoring the per-file scan failure reports an unreadable file clean (exit 0)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-agent-on-shape: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-agent-on-shape "$@"
