#!/usr/bin/env bash
# lint-lifecycle-guard.sh — WARNs component lifecycle callbacks that work when the component is not running.
#
# types/logic.md § Tridium rt idioms (callback contract): changed/added/removed fire whenever the slot
# map is initialized, while the BOG decodes too, so their first line after super is `if (!isRunning()) return;`; and
# stationStarted() runs once at station boot, never for a component dropped into a running station.
# Rows: LG1 a changed(Property, Context) / added(Property, Context) / removed(Property, BValue, Context) body that does
# anything besides super calls and whose first other statement is not an if whose condition calls isRunning() (or
# Sys.atSteadyState()); LG2 a no-argument stationStarted() that does work besides super.stationStarted() in a class
# with no started(), serviceStarted() or descendantsStarted(). Java is read as tokens (go/internal/javasrc): a comment
# or a string never matches. Advisory: Tridium's own source guards differently in many callbacks (a smoke on the
# organized tree gives 2273 LG1 rows), so WARN, never FAIL. [ev: corpus B1242 §1242.7] [ev: corpus B1416 §1416.3]
# [ev: retro course-coverage Δ23]
#
# Implementation: rule package go/internal/lint/rules/lifecycleguard on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-lifecycle-guard <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-lifecycle-guard.sh [--strict] <src-root>
#   Row:  WARN  lint-lifecycle-guard  <file>:<line>  LG<n> <what> ... (types/logic.md § Tridium rt idioms)
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.java
# Mutation: LG1 -- accepting any leading if as the guard clears changed() whose first statement is if (p == in)
# Mutation: LG2 -- dropping the started() lookup WARNs a stationStarted() whose class also has started()
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-lifecycle-guard: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-lifecycle-guard "$@"
