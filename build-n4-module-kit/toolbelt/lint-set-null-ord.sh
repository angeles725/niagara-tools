#!/usr/bin/env bash
# lint-set-null-ord.sh — flags a getSlotPathOrd() result passed to a setter / set() with no null guard: the ORD is
# null on an unmounted component and BComplex.set NPEs on a null value.
#
# BComponent.getSlotPathOrd() returns null when getSlotPath() is null, i.e. the component is not mounted in a
# component space yet (a point a -wb manager just created). Passed to a BOrd property setter it reaches
# BComplex.set(Property, BValue, Context), which dereferences value.getSlotMap() and throws a NullPointerException
# from inside the framework. Fix: guard `if (ord == null || ord.isNull())`, build the ORD from the resolved parent
# folder + slot name, mount the component first. See types/issues-and-gotchas.md §H3.
# [ev: retro wb-mapping-ord-npe-and-wsl-windows-jdk Δ1]
# [ev: code javax/baja/sys/BComponent.java:321-324] [ev: code com/tridium/sys/schema/ComponentSlotMap.java:171-173]
# [ev: code javax/baja/sys/BComplex.java:386-390]
#
# Structural rules (comments blanked by go/internal/javasrc Strip; scope = one method from javasrc Methods, the
# ports of lib/method-boundary.sh mb_strip/mb_parse):
#   SNO1 direct  — `set<Name>(… x.getSlotPathOrd())` / `set(<prop>, x.getSlotPathOrd(), …)`: the getSlotPathOrd()
#                  call is itself an argument (followed by `,` or `)`). Guarded when the call line, or an earlier
#                  line of the same method, compares the SAME receiver's getSlotPathOrd() with null (`!= null` /
#                  `== null`); a check on `a.` never guards `b.getSlotPathOrd()`, and a call-chain or indexed
#                  receiver (`getA().`, `arr[i].`) is never guarded (its row always shows).
#   SNO2 local   — `[BOrd] v = <expr>getSlotPathOrd();` then a later `set…(…v…)` in the same method, with no
#                  `v == null`, `v != null`, `null == v`, `null != v`, `v.isNull()` or `requireNonNull(v` on any
#                  line from the assignment to the call (a ternary on the call line counts).
#   Scope limits (false negatives, never false WARNs): a value carried through a field, a return value or another
#   method; a signature wrapped over several lines before `{` (Methods does not see that method).
#   Not overlapping lint-null-context-write: that lint flags a null CONTEXT argument, this one a null VALUE.
#
# Implementation: go/internal/lint/rules/setnullord on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-set-null-ord <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_run did).
#
# Usage:  lint-set-null-ord.sh [--strict] <src-root>
#   Row:  WARN  lint-set-null-ord  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design (kit-links L2).
# Mutation: SNO1 -- dropping the direct-argument rule passes setTargetOrd(point.getSlotPathOrd()) clean
# Mutation: SNO1-recv -- a receiver-agnostic guard silences b.getSlotPathOrd() after a check on `a.`
# Mutation: SNO1-chain -- keying a call-chain receiver as "." lets getA()'s null check silence getB()'s pass
# Mutation: SNO2-guard -- ignoring the null guard WARNs a local that is checked before the set
# Mutation: SNO2-scope -- matching the variable outside its own method WARNs a same-named local in a sibling method
# Mutation: SNO-awkfail -- an unreadable source file read as empty (internal/lint Run) reports it clean
# Mutation: SNO-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-set-null-ord: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-set-null-ord "$@"
