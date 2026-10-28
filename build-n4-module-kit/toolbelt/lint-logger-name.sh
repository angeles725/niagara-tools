#!/usr/bin/env bash
# lint-logger-name.sh — WARNs a java.util.logging logger whose name is not the module's moduleName.
#
# The logger name is the exact moduleName (or a <moduleName>.<child> logger), so the DebugService and Logger
# Configuration list one row per module and an integrator can raise one module to FINE (types/observability.md § 1).
# A row is each Logger.getLogger("<literal>") whose literal is neither the module name nor <module name>.<child>, and
# each Logger.getLogger(X.class.getName()). A part's module name is the moduleName.set("X") / moduleName = "X" of a
# gradle file in the part directory, else in its group directory, else the part directory name without its
# -rt/-ux/-wb/-se suffix. Scans every *.java under a -rt, -ux, -wb or -se part's src/ of each module root (srcTest/,
# test/ and build/ are not read); Java is read as tokens (go/internal/javasrc), so a comment or a string is never a
# call, and a non-literal argument is not judged. [ev: retro n4-course-parity-observability-horizons Δ3]
#
# Implementation: rule package go/internal/lint/rules/loggername on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-logger-name <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-logger-name.sh [--strict] <module-root>...
#   Row:  WARN  lint-logger-name  <file>:<line>  Logger.getLogger("<x>") is not the module name <M> or a <M>.<child> logger (types/observability.md § 1)
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no part src/ *.java
# Mutation: LN1 -- comparing case-insensitively passes a lower-case "minimalpan" logger in module MinimalPan
# Mutation: LN2 -- dropping the <module>.<child> exemption WARNs a "MinimalPan.defrost" child logger
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-logger-name: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-logger-name "$@"
