#!/usr/bin/env bash
# lint-demand-scope.sh — demand-in-scope lint (Campaign 9 PR2, B820). [ev: retro campaign9-demand-scope]
# Usage: lint-demand-scope.sh [--strict] <java-src-dir>
#
# Within a control-decision method (one that writes an output/cmd/target/setBool driven by a
# numeric input), checks whether the method's PARAMETER list OR the enclosing class's FIELDS
# contain a demand-shaped input {demand*, *call*, enable, loopEnable, *count, BStatusBoolean in*}.
# If the method reads a PROCESS VARIABLE (suction/pressure/discharge/temp/cv/coil/head) in a
# comparison but has ZERO demand-shaped input in scope -> WARN ("pressure without demand").
#
# Row:  WARN  demand-in-scope  <file>:<line>  <method> reads <pv> with no demand-shaped input in scope
# Exits: 0 no WARN (or WARN without --strict) · 1 any WARN under --strict · 3 usage/env
# Env errors (3) cover a sub-directory find cannot enter (no rows), an unreadable .java file (stderr
# "lint-demand-scope: cannot scan <file>: ...", the other files still scanned) and a root with no .java
# file, which prints the stdout row  ERROR  demand-in-scope  <dir>  no *.java sources found
# Only a first --strict is a flag; the first other argument is the root, later ones are ignored.
#
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2).
# §820.3: WARN only (statically-decidable absence), never a hard FAIL — a pure-modulator block
# driven by an upstream demand gate would false-positive on a hard FAIL.
# Advisory: bare *count (e.g. onCount) is the weakest demand signal; kept per B820 but the
# matched token is named in the row so a reviewer sees which term cleared it.
# Mutation: DS2 -- removes demand-shaped input check, allowing pressure-without-demand methods to pass silently
#
# Implementation: rule package go/internal/lint/rules/demandscope (a per-file lint.Rule) on the shared
# engine go/internal/lint (walker, argument parser, row formatter; go/README.md). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source,
# then this script execs `n4lint lint-demand-scope <args>`. Needs `go` on PATH to build (exit 3 otherwise).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-demand-scope: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-demand-scope "$@"
