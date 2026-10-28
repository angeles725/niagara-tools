#!/usr/bin/env bash
# lint-adapter-cfg-default.sh — WARNs a pure-model Cfg field whose default differs from the default of the
# @NiagaraProperty slot the Baja adapter fills it from.
#
# The station starts with the slot default; a pure JUnit test that builds `new Cfg()` runs with the field's default,
# so the module is tested with a configuration the station never starts with (types/logic.md § stateful controller;
# audit 2026-10-03 A7-D1). A Cfg is any class whose simple name ends in "Cfg"; the adapter's link is a statement
# `<x>.<field> = get<Slot>();`, or `get<Slot>().getMillis();` / `.getSeconds();` for a BRelTime slot. The field's
# default is its literal initializer (a number or true/false) or Java's 0/false with none; the slot's is a literal
# defaultValue or BRelTime.make/makeSeconds/makeMinutes/makeHours(<n>). Anything else (a computed initializer, a field
# with several declarators, any other right-hand side, a default it cannot evaluate) is not judged. Fields and slots
# are matched by (class, name): the field of the Cfg type <x> is declared with, the slot of the class holding the
# statement; two Cfg classes or two components sharing a name never cross, and a link that cannot be resolved to one
# field and one slot is not judged (no row rather than a wrong one). Scans every *.java under a -rt part's src/ of
# each module root (srcTest/ and a part's own test/ and build/ are not read; a package named test or build is); Java
# is read as tokens (go/internal/javasrc), so a commented-out annotation or assignment is never one.
#
# Implementation: rule package go/internal/lint/rules/adaptercfgdefault on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-adapter-cfg-default <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-adapter-cfg-default.sh [--strict] <module-root>...
#   Row:  WARN  lint-adapter-cfg-default  <cfg-file>:<field-line>  Cfg.<f> defaults to <x> but slot <s> (<adapter>:<line>) defaults to <y>: ...
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no -rt src/ *.java
# Mutation: ACD3 -- keying Cfg fields by bare name lets the last-read Cfg hide the other's differing default
# Mutation: ACD1 -- comparing a BRelTime slot in raw milliseconds (no getSeconds conversion) WARNs a 30 s field against BRelTime.makeSeconds(30)
# Mutation: ACD2 -- reading a field with no initializer as unjudged passes an int field the adapter fills from a slot defaulting to 6
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-adapter-cfg-default: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-adapter-cfg-default "$@"
