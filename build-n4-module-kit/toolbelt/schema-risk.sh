#!/usr/bin/env bash
# schema-risk.sh <before-dir> <after-dir>
# B795 §795.4 slot-diff survival classifier for Niagara N4 modules.
# Classifies each slot change between two module snapshots and returns the worst-cell verdict,
# so a saved-data-breaking change is caught BEFORE deploy (a .bog binds by name, ungated).
#
# Snapshot dir layout: module-include.xml + <pkg-path>/*.java  (B799 fixture shape).
# Annotation blocks are joined until parens balance (handles single-line fixtures AND
# multi-line real-module annotations identically — D4 requirement).
#
# Documented limits (D4 L1-L4):
#   L1  Multiple simultaneous renames (>=2 removed + >=2 added) are not paired; they surface
#       as separate remove_slot_unknown + add_slot rows. Verdict unchanged (LOSSY-or-worse).
#   L2  remove_slot_complex (SAFE) and retype_complex (LOSSY) are unreachable by design:
#       every unresolved removal -> remove_slot_unknown (LOSSY), every unresolved retype ->
#       retype_unknown (OUTAGE). Never downgrade on uncertainty (B795 §795.2).
#   L3  An annotation whose parentheses never balance in its file, or with no name, and a slot-kind
#       swap (property<->action at same name) emit UNKNOWN -> OUTAGE (`<Type>.<unparsed>@<file>:<line>`
#       for the annotation; the lines after an unbalanced one are still read). No silent skips.
#   L4  @Range enum-tag changes (add/remove/rename tag on a BFrozenEnum @Range) are NOT auto-detected from snapshots; only slot annotations are harvested, never @Range. Manually diff @Range declarations before deploy.
#
# Exit: 0=SAFE  1=LOSSY  2=OUTAGE  3=usage  4=env (unreadable snapshot or *.java / missing tool / no Go toolchain or a failed build)
# Exit-code justification: usage (3) and env (4) sit ABOVE the verdict domain (0-2) so that a
# bad invocation never masquerades as OUTAGE in a deployment script.  This deviates from the
# kit's usual 2=usage/3=env convention by design (D4a).
#
# Rows: `<verdict>  <change_kind>  <Type>.<slot>: <evidence> (<note>)`, then `verdict=<V>` (the worst cell).
# Several rows of one category list in declaration order: the after snapshot's for add_slot, the before
# snapshot's for removals, kind swaps and retypes (the bash classifier printed gawk for-in hash order).
#
# Implementation: go/internal/schemarisk (the B795 §795.4 table is go/internal/schemarisk/b795-795.4.csv,
# diff-guarded by SR-CSV in schema-risk.bats). go/go-tool.sh `go_tool_bin` builds go/bin/schema-risk when it
# is missing or older than a Go source, then this script execs it. VCS-free by design (kit-links L2).
# An empty before snapshot (no slot) adds every after slot: SAFE (the bash classifier read them as removed).
# An unreadable *.java is env (exit 4, `error: cannot read <file>: <reason>`, no row): the bash classifier
# stopped at it and computed a verdict from the slots read before it (retro go-schema-risk Δ2).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'schema-risk: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 4; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" schema-risk)" || exit 4
exec "$BIN" "$@"
