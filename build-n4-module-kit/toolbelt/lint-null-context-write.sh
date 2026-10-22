#!/usr/bin/env bash
# lint-null-context-write.sh — flags component writes/invokes made with a null Context.
#
# A write with a null Context — `parent.set(prop, val, null)` or `comp.invoke(act, arg, null)` —
# is a DOUBLE hazard: (1) it skips the AuditHistory record (no who-changed-what), and (2) it
# grants BPermissions.all, bypassing RBAC. A servlet/dashboard write must carry the invoking
# user's Context: `new BasicContext(BUser.getUserFromSubject(...))`. See types/security.md §1.
# Advisory (WARN): some framework-internal writes legitimately pass null. [ev: retro 2026-09-19-security-model]
#
# A bql: ORD resolved with no caller Context — `BOrd.make("…bql:…").get(base)`, `.resolve(base)`, a bare
# `.get()`/`.resolve()` or a `null` Context — runs the query with no user, so it sees every component regardless
# of permissions (BOrd.java:90-107 pass null; corpus B1237 §1237.7 rule 4). The window starts at the string
# literal holding `bql:` and ends at the first `;` or `{`; only the get/resolve called right on that ORD
# (`"…bql:…" [+ …]) .get(…)`) is judged, so another one-argument call in the statement is not. A query string
# held in a variable is not followed (owed Go lint G4). -wb parts are skipped by this rule only (client code,
# the session carries the user); the null-Context write rule scans every part.
# See types/logic-authoring.md § BQL from code. [ev: retro n4-course-drivers-bql-ords-files Δ1]
# Java is read through the shared scanner (go/internal/javasrc Strip: comments out, a // or /* inside a literal
# is code), then literal contents are blanked for matching; the bql literal is found before blanking. A null
# Context is judged inside the get/resolve call's own parentheses only: `foo(ord.get(base, cx), null)` is clean
# (issue #293).
#
# Implementation: rule package go/internal/lint/rules/nullcontext on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-null-context-write <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-null-context-write.sh <src-root>
#   Row:  WARN  lint-null-context-write  <file>:<line>  null-Context write: <source>
#   Row:  WARN  lint-null-context-write  <file>:<line>  bql: ORD resolved without the caller's Context (pass get(base, cx)): <source>
#   Exit: 0  always (advisory) · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: NCW2 -- drop the ,null match and NCW2 stops warning on set(...,null)
# Mutation: NCW-finderr -- ignoring the walk error (internal/lint Walk) skips an unreadable sub-directory and reports clean
# Mutation: NCW-bql1 -- dropping the one-argument get/resolve match passes get(Sys.getStation()) clean
# Mutation: NCW-bql2 -- checking only the bql: line misses .get(base, null) on a later line
# Mutation: NCW-literal -- matching the unblanked Strip text WARNs on a write quoted inside a string literal
# Mutation: NCW-wb-write -- skipping -wb files for both rules drops the -wb null-Context write row
# Mutation: NCW-bql-unrelated -- matching a one-argument get( anywhere in the window flags params.get("x") beside a clean get(base, cx)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-null-context-write: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-null-context-write "$@"
