#!/usr/bin/env bash
# lint-action-handler.sh — FAILs a Java class whose @NiagaraAction(name = "x") has no usable do<X> method.
#
# The station's type introspector looks up do<X>() / do<X>(P), then do<X>(Context) / do<X>(P, Context) among the
# class's PUBLIC methods and refuses the type at load with "Missing action do method" when neither exists, with
# "Do action method must be not be static" for a static one and with "Action method and do method have mismatched
# signature" when its return type differs from the action's (com/tridium/sys/schema/ComplexIntrospector.java:
# 316-343, 4.14 vineflower; corpus B1231 §1231.4, B1417 §1417.3 j). Slot-o-Matic and the jar task still succeed,
# so a misspelled `doReeset()` only shows on a station: this is the gate. `parameterType` and `returnType` are the
# action's own; a missing `returnType` means void, a missing `parameterType` means no parameter. A `parameterType`
# that is no plain class name (a constant, a baja:Type spec, a concatenation) is UNKNOWN: any of the four shapes
# (doX(), doX(P), doX(Context), doX(P, Context)) then satisfies the action, and a `returnType` of that kind is not
# compared.
# WARN rows (exit 0, 1 under --strict): a public doX(..., Context) beside a public no-Context doX(...) of the same
# parameters, for an action X declared on the class (the no-Context one is found first, the Context one never
# runs; a non-public overload is invisible to the station; ledger G1, types/actions.md § 1); an action with no do
# method in a class whose superclass chain ends at a class declared in ANOTHER file that is no javax.baja framework
# base (BComponent, BObject, BAbstractService, ...): "handler may be inherited from <Super>: not judged"; and `this::doX` or
# `new Thread(() -> doX(` of a declared action, which runs the handler outside the action pipeline (course F3;
# a post(Action...) override is lint-async-blocking's). Source is read as the sibling lints read it
# (go/internal/javasrc: comments stripped, a literal one token). A superclass in the same file is searched for the
# handler; one in another file is not (the WARN above); an abstract do method counts. [ev: corpus B1231 §1231.4]
#
# Implementation: rule package go/internal/lint/rules/actionhandler on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-action-handler <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-action-handler.sh [--strict] <src-root>...
#   Row:  FAIL  lint-action-handler  <file>:<line>  @NiagaraAction "x" has no matching do method ... (ComplexIntrospector.java:316-343 ...)
#         WARN  lint-action-handler  <file>:<line>  doX(..., Context) never runs: ...
#         WARN  lint-action-handler  <file>:<line>  @NiagaraAction "x" has no do method in this file: handler may be inherited from <Super>: not judged ...
#   Exit: 0  clean or WARN-only · 1  any FAIL, or any WARN under --strict · 3  usage/env, a root with no *.java or a
#         sub-directory find cannot enter
# Mutation: AH1 -- dropping the return-type comparison lets a do method that returns another type than the action through
# Mutation: AH2 -- counting a non-public do method as the handler hides the missing one
# Mutation: AH3 -- reading annotations or do methods from comment or string text flags or clears a quoted declaration
# Mutation: AH4 -- dropping the doX()/doX(Context) pair check lets the Context overload that never runs through
# Mutation: AH5 -- flagging `this::doX` of a method that is no declared action reports an ordinary helper
# Mutation: AH6 -- reading an unknown parameterType as no parameter FAILs a valid handler that takes the parameter
# Mutation: AH7 -- FAILing a class whose superclass is declared in another file reports a handler that may be inherited
# Mutation: AH8 -- firing the pair WARN for a non-public overload or an undeclared action reports a pair the station never sees
# VCS-free by design.
set -u
# -h/--help prints this header (the comment lines above), never a code line.
case "${1:-}" in -h | --help) sed -n '2,/^set -u$/ { /^#/ s/^# \{0,1\}//p }' "${BASH_SOURCE[0]}"; exit 0 ;; esac
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-action-handler: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-action-handler "$@"
