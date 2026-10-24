#!/usr/bin/env bash
# lint-servlet.sh — BWebServlet security lint for Niagara N4 Java modules (Campaign 8 PR12, B813).
#
# Scans *.java files under <src-dir> that extend BWebServlet or declare doGet/doPost/service
# handlers. Dot-directories and *-worktrees sub-directories are pruned (D9b). Reports per-check rows.
#
# Checks:
#   auth          FAIL  a doGet/doPost/service handler body contains a write (.setDouble/.setFloat/
#                       .setInt/.setLong/.setRelTime/.setString( or a set( not preceded by an identifier character) with no
#                       getRemoteUser()/.getUser() gate in the same method body.
#   input-400     FAIL  Double.parseDouble( or Integer.parseInt( that no try statement encloses (its block or a
#                       try-with-resources specification, go/internal/javasrc TryBlocks) whose try keyword is in
#                       the same method or class-level lambda body (javasrc Scopes) (fixed 2026-10-24: the 5-line
#                       look-back took a try of another handler, or one already closed, as the guard, and the
#                       `try {` regex missed try-with-resources).
#   unbounded-set WARN  .setDouble/.setFloat/.setInt/.setLong/.setRelTime( in a handler body with no
#                       getMin()/getMax()/getFacets()/MIN/MAX/clamp in the same body.
#   catch-no-400  WARN  the catch that answers a Double.parseDouble(/Integer.parseInt(/Long.parseLong(/
#                       Double.valueOf( call has no 400 response in its own body (400, SC_BAD_REQUEST,
#                       sendError(, setStatus(, jsonError( or sendJson(); the row is at the catch line. That
#                       catch is read structurally (go/internal/javasrc TryBlocks: try-with-resources, catch
#                       clauses attached to the try block's closing brace): the first clause catching
#                       NumberFormatException or a supertype on the innermost enclosing try, outward past a
#                       try without one and past a clause that throws; when none catches it, the first clause
#                       of the innermost try that has one (fixed 2026-10-24: the bash/awk took the first catch
#                       within 4 lines after the try's end, unrelated or of an outer try, and missed
#                       try-with-resources and the right multi-catch clause).
#   cache-nofinger WARN Cache-Control max-age>0 in the file without a fingerprinted resource
#                       (static detection: Cache-Control and max-age=[1-9] on one line).
#   log-in-handler WARN LOG.info( inside a doGet/doPost/service handler body.
#   csrf-xrw-only WARN  X-Requested-With check in the file with no CsrfUtil/x-niagara-csrfToken/
#                       csrfToken reference — XHR-only guard is weaker than Niagara CSRF token.
#                       The row anchors at the first code line naming the header.
#                       [ev: corpus B813]
#   api-response-headers INFO  a doGet/doPost/service handler that calls getWriter()/getOutputStream()
#                       but the class has no setHeader("X-Content-Type-Options",...) and no
#                       setApiHeaders()/applyHeaders() helper call anywhere. Information only: the
#                       station's header providers reach every module servlet through
#                       TridiumSecurityFilter, and a servlet-level header REPLACES them.
#                       [ev: corpus B1133 UXS1] [ev: corpus B1349 C1]
#
# Every check reads code, never a // or /* */ comment (a Javadoc included). Code shapes (handler and
# class declarations, calls, braces, try/catch, the 400 indicators) also ignore string and char literal
# contents, so "}" or "Integer.parseInt(" in a string is no code; the header names (X-Requested-With,
# X-Content-Type-Options, the CSRF token) and the Cache-Control value are read with literal text kept.
#
# WARN -> FAIL under --strict; INFO rows never escalate. Deferred: R12.2 step-up re-auth (B803 — no static primitive).
#
# Implementation: rule package go/internal/lint/rules/servlet on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-servlet <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-servlet.sh <src-dir> [--strict]
#
#   Row format:  FAIL|WARN|INFO  <check>  <file>:<line>  <detail>     (<file> relative to <src-dir>)
#   Exits:       0  no FAIL (WARN-only is still 0) · 1  any FAIL · 3  usage/env (K20), incl. a
#                sub-directory the walk could not enter (no row is printed) or an unreadable source
#                file (the other files are still scanned)
#   Empty root:  a <src-dir> with no *.java under it is an env error (exit 3), never a clean pass; stdout
#                gets `ERROR  lint-servlet  <src-dir>  no files to scan (K20: wrong path or empty
#                scaffold)`, the lib/scan-files.sh K20 row (go/internal/lint Spec.EmptyError keeps it).
#
# This script is VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro campaign8-lint-servlet] [ev: retro go-lint-servlet]
# Mutation: LSV1 -- removes auth gate check, allowing a doPost with write but no getRemoteUser() to pass
# Mutation: LSV4b -- reading the X-Requested-With flag from raw lines reports a guard documented only in a Javadoc
# Mutation: LSV2c -- matching the parse calls on raw lines FAILs input-400 on a commented-out Double.parseDouble(
# Mutation: LSV2d -- starting the catch search after the try's closing line passes "} catch (E e) { v = 0; }"
# Mutation: LSV2e -- a 5-line look-back for "try {" lets doGet's try guard doPost's Integer.parseInt(
# Mutation: LSV2f -- taking the first catch within 4 lines of the try's end judges a multi-catch by its IOException clause
# Mutation: LSV2g -- finding the try by the "try {" regex FAILs a parse inside try (Reader r = open()) { ... }
# Mutation: LSV7 -- removes api-response-headers check, so a doGet with getWriter() but no X-Content-Type-Options emits no INFO row
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-servlet: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-servlet "$@"
