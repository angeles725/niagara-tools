#!/usr/bin/env bash
# lint-post-permission.sh — FAILs a request, RPC or action handler that calls post( with no permission check before it.
#
# BComponent.post(action, arg, cx) queues the action; the queued invocation runs with a null Context, so the caller's
# permission is never checked and the action runs system-privileged. The caller's OPERATOR or ADMIN permission must be
# checked BEFORE post() (types/security.md § 2.4). Evidence in the extracted baja API (vineflower baja 4.14.0.162):
#   javax/baja/sys/BComponent.java:590-593      post(action, arg, cx) is `enqueueAction(this, action, argument)`: cx is dropped
#   com/tridium/sys/engine/EngineManager.java:293-299, :310   the queue runs `EngineUtil.doInvoke(comp, action, arg, null)`: a null Context
#   javax/baja/sys/BComponent.java:582, :595    invoke(action, arg, cx) and doInvoke(action, arg, cx) are where a real Context is checked
# [ev: retro module-hardening-failure-modes-deltas Δ4]
#
# A row is a FAIL when ALL hold, in one method (the unit; a method of an inner or anonymous class is judged on its own):
#   1. the method is a handler: doGet/doPost/doPut/doDelete/doHead/doOptions/service; or it takes a WebOp,
#      HttpServletRequest or HttpServletResponse; or it carries @NiagaraRpc; or it is a do<Action> method (do + capital)
#      that takes a Context (an action invoked with the caller's Context). A method with none of these (started(),
#      a private helper, a framework callback such as changed(Property, Context), a do<Action>() with no Context, the
#      run() of an inner Runnable) is an internal path and is never judged: a station timer post is not a row.
#   2. it calls post( : bare or this.post( with any argument count; or <receiver>.post( with two or more arguments
#      whose first is not a string literal (an HTTP client's post(url) and post("http://...", body) are not it).
#   3. no permission check came BEFORE that post( in the same method body (a lambda in the method is part of it).
#      Accepted checks, each present in the extracted API (the same corpus):
#        checkInvoke(  checkWrite(          javax/baja/user/BUser.java:914, :902 (operator or admin tier by the slot's OPERATOR flag)
#        getPermissionsFor(                 javax/baja/user/BUser.java:837
#        canInvoke(                         javax/baja/sys/BComponent.java:942 (canInvoke(OrdTarget))
#        hasOperatorInvoke(  hasAdminInvoke(  javax/baja/security/BPermissions.java:102, :114
#        check(  has(  whose arguments name BPermissions   BUser.java:887 check(target, BPermissions), BPermissions.java:90 has(BPermissions)
#      checkRead( is not accepted (reading is not invoking). cx.getUser() alone is not a check. security.md § 2.4 shows
#      checkInvoke; the other forms are the same API's own checks, so they are accepted by the same rule.
# A check, a post( or a waiver inside a comment or a string literal is not code. The check must precede the post( in token
# order; it is not proved to guard that post( (an unrelated check earlier in the method passes: lexical, one method at a time).
#
# Waiver: `// post-permission: ok <reason>` (or the same in a block comment) on the line of the post( or the line directly
# above it accepts that one post(. The reason is required: `// post-permission: ok` alone waives nothing. Use it when
# the permission is checked elsewhere (a filter, the caller, an invoke() that already ran checkInvoke on the same Context).
# There is no earlier source-comment waiver convention in the kit's Java lints (lint-null-context-write has none); this
# one is new and local to this rule.
#
# Limits: every post( in a handler is judged, whatever action it posts (the lint cannot tell an OPERATOR action from a
# harmless one: waive it with the reason). A post( in an anonymous class or a method that no handler rule names is not seen.
# `--strict` is accepted for the shared contract and changes nothing (a FAIL is already exit 1). `-h`/`--help` prints this header.
#
# Implementation: rule package go/internal/lint/rules/postpermission on the shared engine go/internal/lint (go/README.md).
# This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older
# than a Go source, then this script execs `n4lint lint-post-permission <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-post-permission.sh <src-dir> [--strict]
#   Row:  FAIL  lint-post-permission  <file>:<line>  post() in '<method>' runs the queued action with no RBAC Context ...   (<file> relative to <src-dir>)
# Exits:  0  clean · 1  any FAIL · 3  usage/env: a directory the walk cannot enter, an unreadable file, or a <src-dir> with no *.java
#         (stdout gets `ERROR  lint-post-permission  <src-dir>  no files to scan (K20: wrong path or empty scaffold)`)
# Mutation: PP1 -- accepting a permission check anywhere in the method, not only before the post(, passes a post-then-check handler
# Mutation: PP2 -- judging every method, not only handlers, FAILs an internal timer post (started(), doTick(), a private helper)
# Mutation: PP3 -- accepting a waiver with no reason waives a bare `// post-permission: ok`
# Mutation: PP4 -- reading a guard name in a comment or a string literal as a guard passes an unchecked post(
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
self="$(readlink -f "${BASH_SOURCE[0]}")"
case "${1:-}" in -h | --help) sed -n '2,/^set -u$/{/^set -u$/d;s/^# \{0,1\}//;p}' "$self"; exit 0 ;; esac
here="$(cd "$(dirname "$self")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-post-permission: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-post-permission "$@"
