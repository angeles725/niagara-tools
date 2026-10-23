#!/usr/bin/env bash
# lint-service-types.sh — flags a service whose getServiceTypes() registers no type (an empty array or null)
# and WARNs when the returned array does not hold the service's own TYPE.
#
# ServiceManager.register iterates exactly the array getServiceTypes() returns: an empty array or null registers
# the service under no type, so Sys.getService(Type) never finds it. IntelliJ's "Implement methods" stub returns
# `new Type[0]`, which is how it ships by accident. Real services return `new Type[] { TYPE }` (BAlarmService) or
# add their interface type (`{ TYPE, BILoggingService.TYPE }`). See METHODOLOGY.md § rt.
# [ev: corpus B1419 §1419.1] [ev: code com/tridium/sys/service/ServiceManager.java:93-99]
#
# Structural rules (comments blanked by the shared scanner go/internal/javasrc Strip):
#   Scope     — a `Type[] getServiceTypes(...)` declaration with a body; an abstract/interface declaration
#               (`;` before `{`) and a file without one are out of scope.
#   Returned  — every `return <expr>;` of the body, the worst verdict wins (empty > other > unresolved > own).
#               A return inside a nested scope — a block lambda, an anonymous class, a local class — belongs to
#               that scope, not to getServiceTypes(), and is skipped; a switch-rule block (`case A, B -> {`,
#               `default -> {`, a guarded `case T t when (…) -> {`) is not one, a lambda on a case line is.
#               String/char/text-block contents are blanked first, so no literal opens a scope or holds a return;
#               a \ escapes the next character on its line (so \""" never closes a text block, \\""" does) (#286).
#               <expr> is an array literal (`new Type[] { … }`), `null`, `new Type[0]`, or a field (`a`, `Own.a`,
#               `(Type[]) a.clone()`) resolved through every `a = …;` assignment in the same file (a static
#               initializer counts), read from the assignment itself, not from the start of its line.
#   FAIL      — the returned array is empty (`new Type[0]`, `new Type[] {}`, `{}`) or null.
#   Own type  — `TYPE`, `this.TYPE`, `<OwnClass>.TYPE` or `getType()`/`this.getType()` (own class = first `class X`
#               in the file); `<OwnClass>.<field>` resolves like a bare field; outer parentheses are dropped.
#   WARN      — the array holds no own type; or the return cannot be resolved in this file (a super call, an
#               inherited field, a method).
#
# Implementation: rule package go/internal/lint/rules/servicetypes on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-service-types <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-service-types.sh [--strict] <src-root>
#   Row:  FAIL|WARN  lint-service-types  <file>:<line>  <detail>     (<line> = the getServiceTypes declaration)
#   Exit: 0  no FAIL (WARN only without --strict) · 1  any FAIL, or any WARN under --strict · 3  usage/env, an
#         unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-service-types  <root>  no *.java sources found (wrong path or empty source tree)` (go/internal/lint
# Spec.EmptyError; declared difference 2026-10-23: the bash lint exited 0 with no row).
# VCS-free by design (kit-links L2).
# Mutation: ST1 -- dropping the empty-array check passes the IntelliJ 'Implement methods' stub new Type[0]
# Mutation: ST2 -- requiring the array on the return line flags a multi-line initializer that holds TYPE
# Mutation: ST3 -- not resolving a returned field passes a serviceTypes field initialized to new Type[0]
# Mutation: ST4 -- accepting any X.TYPE as the own type passes a service registered only under another type
# Mutation: ST5 -- treating an unresolved return as clean passes return super.getServiceTypes()
# Mutation: ST9 -- resolving only the declaration line misses a field assigned in a static block to new Type[0]
# Mutation: ST10 -- checking only the first return passes a getServiceTypes() whose fallback branch returns null
# Mutation: ST11 -- reading the initializer from the line start stops at an earlier ';' and misses new Type[0]
# Mutation: ST8 -- dropping getType() from the own-type set WARNs the vendor form new Type[] { this.getType() }
# Mutation: ST12 -- counting nested-scope returns FAILs a lambda that returns null inside a method returning { TYPE }
# Mutation: ST13 -- ranking other below unresolved reports a foreign-only array as merely unresolved
# Mutation: ST14 -- classifying any '->' on a case line as a switch rule FAILs a lambda's return null on that line
# Mutation: ST15 -- reading literal contents lets "class A" in a condition hide the method's return null
# Mutation: ST6 -- matching inside comments flags a commented-out return new Type[0]
# Mutation: ST-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: ST-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-service-types: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-service-types "$@"
