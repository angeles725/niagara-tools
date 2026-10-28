#!/usr/bin/env bash
# lint-slot-default-type.sh — FAILs an @NiagaraProperty whose defaultValue literal has another Java type than its slot.
#
# Slot-o-Matic copies defaultValue verbatim into newProperty(flags, <default>, facets); Java overload resolution then
# picks newProperty(int, boolean|int|long|float|double|String|BValue, ...) by the literal's own type, so the property
# holds a value of that type. Type introspection widens a BInteger default for a long/float/double slot and refuses
# every other mismatch ("Property default value is of wrong type", baja ComplexIntrospector): the module compiles and
# passes every build gate, and the type fails at runtime. Judged slot types: int, long, float, double, boolean,
# String (and BString); judged defaults: true/false, integer literals (long with L), floating literals (float with f,
# double with d or a decimal point/exponent), string literals, and
# B<Boolean|Integer|Long|Float|Double|String>.<make(...)|CONSTANT>. An int literal on a long/float/double slot is
# clean. Any other expression is not judged. Comments are out; annotations may span lines. [ev: corpus B1417 §1417.3 l]
# [ev: retro course-coverage Δ21]
#
# Implementation: rule package go/internal/lint/rules/slotdefaulttype on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-slot-default-type <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-slot-default-type.sh <src-root>
#   Row:  FAIL  lint-slot-default-type  <file>:<line>  @NiagaraProperty <name> type="<t>" defaultValue="<v>": the default is ... the type fails to load ... (types/structure.md § Slot defaults)
#   Exit: 0  clean · 1  any FAIL · 3  usage/env, a root with no *.java
# Mutation: SDT1 -- widening every numeric literal (not only an int on long/float/double) passes 2d on an int slot
# Mutation: SDT2 -- dropping the B-value kind passes BBoolean.TRUE on a double slot
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-slot-default-type: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-slot-default-type "$@"
