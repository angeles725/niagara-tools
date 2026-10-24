#!/usr/bin/env bash
# lint-driver-scaffold.sh — flags NDriver wizard stubs left in shipped driver sources.
#
# The devkit NDriver wizard generates `public void doPing() { // TODO - add ping implementation  pingOk(); }`
# (devkit-wb.jar!gradle/ndriver/BNfooDevice.java.vm:102-111): left as is, every device reports UP
# forever, so a dead device never goes DOWN (silent success). That is a FAIL. The wizard's other
# placeholders compile and run but do nothing useful, so they WARN. See types/driver-authoring.md § 5.
# [ev: retro n4-course-handson-ndriver Δ2] [ev: retro n4-course-handson-ndriver Δ6]
#
# Usage:  lint-driver-scaffold.sh <src-root>     (scans *.java under it)
#   Row:  FAIL  lint-driver-scaffold  <file>:<line>  doPing() only calls pingOk() | doPing() still holds the wizard TODO
#   Row:  WARN  lint-driver-scaffold  <file>:<line>  wizard stub: <marker> | <method>() only returns null
#                                                    | doSubmitDiscoveryJob() submits with a null Context
#                                                    | method boundaries not trusted (<reason>)
#   Exit: 0  no FAIL (WARN-only is 0) · 1  any FAIL · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-driver-scaffold  <root>  no *.java sources found (wrong path or empty source tree)` (go/internal/lint
# Spec.EmptyError; declared difference 2026-10-23: the bash lint exited 0 with no row).
# A method body is the text between a member header's `{` and its matching `}`, at any class depth (a nested
# device class counts). The header is the code since the last `;`, `{` or `}`, with every annotation (its name
# and its balanced argument list, on one line or several) dropped, so `@SuppressWarnings({"x"}) public void
# doPing()`, `@Ann(v = f(x), w = g(y))` and a `throws` line still name doPing, and the row line is the line of
# the header's first `(`. `new X() {`, `new pkg.X() {` and `new Outer . Inner() {` open an anonymous class,
# never a method, so its members are checked (issue #282). Java is read through the shared scanner
# (go/internal/javasrc Strip: comments out, literal text kept), then literal contents are blanked, so a
# "tcp://" or "/*" string, a '{' char or a text block holding braces cannot move a boundary. A file whose
# braces do not balance, or that ends inside a block comment, a text block, an unterminated "..."/'...'
# literal or an annotation argument list, gets one WARN row: its boundaries are not trusted, so a stub in it
# may be missed. The path is printed as is (a backslash in it stays one). VCS-free by design.
# [ev: retro r8-276-driver-scaffold Δ2] [ev: retro go-lint-driver-scaffold Δ1]
#
# Implementation: rule package go/internal/lint/rules/driverscaffold on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-driver-scaffold <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# verify-module.sh --src runs the same rule on every driver module (its driver-scaffold rows, types/driver-authoring.md § 12).
# Mutation: LDS2 -- drop the pingOk-only body match and the TODO match, and the wizard doPing stops failing
# Mutation: LDS3 -- drop the TODO-in-doPing match and a half-edited wizard doPing passes
# Mutation: LDS4 -- drop the submit(null) match and the wizard discovery job stops warning
# Mutation: LDS6 -- stop blanking literal contents and the '{' char and the text block move the boundaries
# Mutation: LDS7 -- drop the end-of-file balance check and the truncated file reports clean
# Mutation: LDS9 -- name the method by the leftmost identifier( without dropping annotations and the annotated stubs stop failing
# Mutation: LDS10 -- escape-process the path (awk -v style) and its \n becomes a newline in the row
# Mutation: LDS-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-driver-scaffold: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-driver-scaffold "$@"
