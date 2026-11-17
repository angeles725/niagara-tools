#!/usr/bin/env bash
# lint-servlet-write-guards.sh — flags a servlet write path that checks permissions on the servlet or the
# user instead of the written target, writes with no target check, or verifies no CSRF token.
#
# A write endpoint must check the caller's permission on the TARGET it writes: BIProtected.getPermissions(cx)
# is cx.getUser().getPermissionsFor(this) (BIProtected.java:44,53 javadoc; BComponent.java:896-903), so
# getPermissionsFor(this) or this.getPermissions(cx) in a servlet reads the servlet's own permissions, and
# BUser.getPermissions(cx) is the caller's permission ON THE USER object (BUser.java:810-835).
# The template and its rationale: types/servlet-write.md. [ev: retro 2026-11-10-mcpn4-history-batch-tools Δ6]
# [ev: corpus B612 §612.2] [ev: corpus B813 §813.4] [ev: corpus B763 §763.2-763.3] [ev: corpus B602 §602.3]
#
# Checks (web code only: a file naming BWebServlet, HttpServlet, WebOp or HttpServletRequest; comments and
# string literals are not code, go/internal/javasrc):
#   self-permission       FAIL  getPermissionsFor(this | user | <x>.getUser()), this.getPermissions(cx),
#                               super.getPermissions(cx), an unqualified getPermissions(cx), user.getPermissions(cx) /
#                               getUser().getPermissions(cx), also through a local bound in the method to the user or the
#                               servlet (BUser caller …; caller = op.getUser(); self = this; then caller.getPermissions(cx))
#   no-target-permission  WARN  a slot write (.set / .setDouble / … ) or .invoke(…) with no target check earlier in the
#                               same method: target.getPermissions(cx), getPermissionsFor(target), ordTarget.canWrite()
#                               or the template's requireTargetWrite(target, …). The check must name a variable of the
#                               written receiver (other.getPermissions(cx) does not guard target.set) and its result must
#                               be used in a condition or an assignment (a bare `target.getPermissions(cx);` guards nothing);
#                               a self-permission line is no check. A requireTargetWrite declared in the same file counts
#                               only when its body checks a permission on its first parameter; one declared elsewhere is
#                               trusted by name (the lint reads one file). A receiver it cannot read (getTarget().set(…))
#                               matches any check.
#   no-csrf               WARN  a BWebServlet that writes and calls no verifyCsrfToken( and names no CsrfProtectedFilter
#                               (a token header or a local named csrfToken alone is no check; an X-Requested-With guard
#                               is lint-servlet csrf-xrw-only's row)
#   Scope limits (false negatives): a check in another method than the write (wrap it as requireTargetWrite and call
#   it in the writing method); the global user map `getUser().getPermissions().has(OPERATOR_WRITE)` is no target check.
#   no-csrf reads the servlet file only: an X-Requested-With guard in a separate dispatch class (types/dashboard.md
#   DUX1) is not seen, and the row is still true there, since X-Requested-With is no session-bound token (B813 §813.4).
#
# Implementation: go/internal/lint/rules/servletwriteguard on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-servlet-write-guards <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise).
#
# Usage:  lint-servlet-write-guards.sh [--strict] <src-root>
#   Row:  FAIL|WARN  <check>  <file>:<line>  <detail>
#   Exit: 0  no FAIL and no WARN (or WARN without --strict) · 1  any FAIL, or any WARN under --strict · 3  usage/env or an unscannable source file
# Empty root: a root with no *.java under it is an env error (exit 3), never a clean pass.
# VCS-free by design (kit-links L2).
# Mutation: SWG1 -- a types/servlet-write.md template whose requireTargetWrite call reads getPermissionsFor(this) fails SWG1
# Mutation: SWG3 -- dropping `this` from the self-permission pattern passes getPermissionsFor(this) (SWG3 red)
# The rule's own mutation pins (user receiver, aliases, guard order, receiver match, the template helper, X-Requested-With,
# the csrf call, web scope) live in
# the Go source with its golden cases (go/internal/lint/rules/servletwriteguard).
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-servlet-write-guards: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-servlet-write-guards "$@"
