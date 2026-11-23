#!/usr/bin/env bash
# lint-instance-type-order.sh — FAILs a Java class annotated @NiagaraType (or holding a Slot-o-Matic
# `TYPE = Sys.loadType(...)` field) that declares a static INSTANCE field AFTER its static Type TYPE field.
#
# A 4.14 lab station refuses such a type with "INSTANCE field is null; insure getType() comes last" while
# pure tests, build, verify and report all pass (issue #847; the station's own source was not read). The row
# says: declare INSTANCE before TYPE. A class with no INSTANCE, or INSTANCE before TYPE, is clean.
# Source is read as the sibling lints read it (go/internal/javasrc, comments stripped, literals one token):
# a comment, a string or text block, a local variable, a nested type's member and an anonymous class are no
# field of the class; each class is judged on its own fields. Fix: move the INSTANCE field above TYPE.
# [ev: issue #847]
#
# Implementation: rule package go/internal/lint/rules/instancetypeorder on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-instance-type-order <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-instance-type-order.sh [--strict] <src-root>...
#   Row:  FAIL  lint-instance-type-order  <file>:<instance-line>  static INSTANCE is declared after TYPE: ... declare INSTANCE before TYPE
#   Exit: 0  clean · 1  any FAIL · 3  usage/env, a root with no *.java or a sub-directory find cannot enter
# Mutation: ITO1 -- reading INSTANCE position-blind flags a class that declares INSTANCE before TYPE
# Mutation: ITO2 -- reading INSTANCE from comment or string text flags a quoted declaration
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-instance-type-order: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-instance-type-order "$@"
