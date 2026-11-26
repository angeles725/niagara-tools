#!/usr/bin/env bash
# lint-property-accessors.sh — FAILs a B* class whose hand-declared slot lacks its public getter or setter.
#
# A `public static final Property x = newProperty(...)` in a top-level B<Upper>* class needs a public get<X>() or
# is<X>() and a public set<X>(...) in the same class (X is x with its first letter upper-cased). Without them the
# module compiles, signs and passes every build gate, and the station cannot load the type: it logs
# `GRAVE [sys.schema] ... No setter for property (x)` / `Cannot load type for ...` and every later use throws
# NoClassDefFoundError (measured on Niagara 4.14.0.162: a BPointExtension with three newProperty slots and getters
# only was unloadable; adding public setters fixed it). Slot-o-Matic generated code always has both accessors, so
# only hand-written slots are caught. One FAIL row per missing or non-public accessor, at the Property declaration.
# Judged: static `Property` fields initialised by `newProperty(` directly in the class body; the accessors must be
# declared directly in the same class (a nested class's do not count, and its own slots are not judged). A class
# whose name is not B plus an upper-case letter is not judged. Comments and string literals are not code.
# [ev: issue #1159]
#
# Implementation: rule package go/internal/lint/rules/propertyaccessors on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-property-accessors <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-property-accessors.sh <src-root>
#   Row:  FAIL  lint-property-accessors  <file>:<line>  property <name> of <Class>: no public set<X>(...): the station cannot load the type ... (types/structure.md § Property accessors)
#   Exit: 0  clean · 1  any FAIL · 3  usage/env, a root with no *.java
# Mutation: PA1 -- not requiring a setter passes a slot with a getter and no set<X> (the issue #1159 type)
# Mutation: PA2 -- accepting a non-public setter (or no getter) passes a slot the station still cannot load
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-property-accessors: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-property-accessors "$@"
