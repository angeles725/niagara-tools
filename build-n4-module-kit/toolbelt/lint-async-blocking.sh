#!/usr/bin/env bash
# lint-async-blocking.sh — WARNs an ASYNC action that blocks the shared engine thread (ledger G28).
#
# An action declared with Flags.ASYNC runs on the ONE engine thread that every timer callback and every other
# ASYNC action in the station shares; a do<Name> that blocks there stalls them all (corpus B1238 §1238.3-§1238.5,
# types/actions.md § 2). The sanctioned offload is an ASYNC action whose post(Action, ...) override hands
# `new Invocation(this, action, arg, cx)` to the module's own executor, which calls invoke() on it.
# WARN rows only, never FAIL (a plain ASYNC handler is fine for short bounded work: the framework puts ASYNC on
# control-output, history and alarm actions on purpose, and the course exam accepted an ASYNC history export):
#   - an ASYNC action of a class with NO post(Action...) override whose do<Name> body calls Thread.sleep(,
#     openConnection(, getOutputStream(, sendSync(, names HttpClient, or calls .get( with a "bql:" string literal
#     in the body (a BQL ord resolved on the engine thread); one row per action, on the do method's line;
#   - a post(Action...) override that runs the handler itself (`this::doX` or `-> doX(`), which skips the action
#     pipeline: no links, no action event, no audit, no Context.
# Only `flags = ... Flags.ASYNC` (or a static-imported ASYNC) is read, not a numeric flags value; a non-ASYNC action
# is never judged. Source is read as the sibling lints read it (go/internal/javasrc, comments stripped, literals
# one token). [ev: ledger.d/2026-10-23-r8-p1-slots.md G28] [ev: corpus B1238 §1238.3-§1238.5]
#
# Implementation: rule package go/internal/lint/rules/asyncblock on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-async-blocking <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-async-blocking.sh [--strict] <src-root>...
#   Row:  WARN  lint-async-blocking  <file>:<line>  ASYNC action "x": doX blocks the shared engine thread (Thread.sleep() line N) ...
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.java or a sub-directory
#         find cannot enter
# Mutation: AB1 -- judging a non-ASYNC action flags a handler the lint must not judge
# Mutation: AB2 -- ignoring a post override flags the sanctioned Invocation offload
# Mutation: AB3 -- reading blocking calls from comment or string text flags a quoted sleep
# Mutation: AB4 -- dropping the `this::doX` check in post() lets the direct run through
# VCS-free by design.
set -u
# -h/--help prints this header (the comment lines above), never a code line.
case "${1:-}" in -h | --help) sed -n '2,/^set -u$/ { /^#/ s/^# \{0,1\}//p }' "${BASH_SOURCE[0]}"; exit 0 ;; esac
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-async-blocking: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-async-blocking "$@"
