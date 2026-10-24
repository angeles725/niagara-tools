#!/usr/bin/env bash
# lint-slotomatic-markers.sh — FAILs a *.java whose Slot-o-Matic generated region (BEGIN … END BAJA AUTO
# GENERATED CODE) holds a number of `/*@ $…$ @*/` stamp lines other than exactly one.
#
# Slot-o-Matic counts the region lines whose trim() starts with "/*@"; on any count other than 1 it throws
# "Found multiple metadata blocks" (devkit-wb vineflower com/tridium/slottool/JavaUnit.java:270-272), logs
# `Could not process <file>` as a WARNING and its task SUCCEEDS (Compiler.java:240-246): the file is skipped
# and the build stays green until compileJava fails with `Slot with name <x> not found`. The usual cause is
# a hand-written or copied region, e.g. empty BEGIN/END markers pre-written in a new class (METHODOLOGY.md
# K15). Real case: Cliente/panccadia-leon DashboardPan-rt BHmiStatus.java:29-75, zero stamps (de36722).
# Markers are read in comments only (go/internal/javasrc File.Marker: a marker quoted in a string literal or
# a text block is code); a stamp line is counted on the raw line, as Slot-o-Matic does. A region runs from
# the last BEGIN to the next END (from line 1 when no BEGIN came first); every closed region is checked
# (Slot-o-Matic reads the last one). A BEGIN with no END after it WARNs (unclosed-region).
# Fix: delete the BEGIN/END markers and rerun `:<part>:slotomatic --info`, read it for `Could not process`.
# [ev: retro n4-course-handson-slotomatic-authoring Δ1] [ev: retro r8-t9i-k15 Δ1]
#
# Implementation: rule package go/internal/lint/rules/slotomaticmarkers on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-slotomatic-markers <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-slotomatic-markers.sh [--strict] <src-root>...
#   Row:  FAIL  lint-slotomatic-markers  <file>:<begin-line>  generated region holds <n> /*@ $...$ @*/ stamp lines, ...
#   Row:  WARN  unclosed-region  <file>:<begin-line>  BEGIN BAJA AUTO GENERATED CODE with no END marker after it ...
#   Exit: 0  clean (or WARN only) · 1  any FAIL, or a WARN under --strict · 3  usage/env, a root with no *.java or a sub-directory find cannot enter
# Mutation: SM1 -- dropping the stamp count check passes an empty BEGIN/END region clean
# Mutation: SM2 -- checking only for a missing stamp (n == 0) passes a region with two stamps
# Mutation: SM5 -- reading markers on the raw line (not File.Marker) FAILs marker text quoted in a string literal
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-slotomatic-markers: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-slotomatic-markers "$@"
