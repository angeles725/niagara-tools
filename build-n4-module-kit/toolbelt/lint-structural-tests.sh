#!/usr/bin/env bash
# lint-structural-tests.sh — WARNs a srcTest read of a .java source as text in a part whose srcTest has no
# BTestNg/BTestNgStation class.
#
# A test that reads the module's own .java source pins its shape, not the component's behaviour: component behaviour
# is a BTestNgStation test that drives real slots, and a source-structural test is the last resort (types/moduleTest.md
# § Client parity doctrine). A row is each string literal ending in ".java" inside the arguments of
# Files.readAllBytes, Files.readAllLines, Files.readString, Files.lines, Paths.get or Path.of in a srcTest file, when
# that part's srcTest holds no class extending BTestNg or BTestNgStation (simple or qualified superclass name).
# Scans every *.java under a srcTest/ of each module root (build/ is not read); Java is read as tokens
# (go/internal/javasrc), so a comment is never a read. Under --strict the rows print FAIL.
# [ev: retro testing-parity-btestng Δ5]
#
# Implementation: rule package go/internal/lint/rules/structuraltests on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-structural-tests <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-structural-tests.sh [--strict] <module-root>...
#   Row:  WARN  lint-structural-tests  <file>:<line>  reads "<path>.java" as text and <part>/srcTest has no BTestNg or BTestNgStation class: ...
#   Exit: 0  clean or WARN-only · 1  any row under --strict (printed FAIL) · 3  usage/env, a root with no srcTest/ *.java
# Mutation: ST1 -- accepting only BTestNgStation WARNs a part whose test class extends com.tridium.testng.BTestNg
# Mutation: ST2 -- matching any .java literal in the file WARNs a test that reads a .json fixture and only names a .java file
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-structural-tests: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-structural-tests "$@"
