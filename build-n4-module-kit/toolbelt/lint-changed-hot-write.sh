#!/usr/bin/env bash
# lint-changed-hot-write.sh — flags a hot changed() callback that re-runs a heavy control
# method at INPUT RATE and writes a persisted (non-transient) slot on every call.
#
# Shape (RUN8, B1158): a `changed(Property p, Context cx)` override with N>=6 `p == <prop>`
# branches guarding a call to a heavy `execute()`/control method, with NO rate guard (no
# `Clock.millis()` delta gate, no `Clock.schedule` coalescing ticket) IN THE CHANGED() BODY,
# where the class also writes >=1 NON-TRANSIENT @NiagaraProperty slot (declared without
# Flags.TRANSIENT) reachable from changed() (directly, or from its execute() callee).
# Under a churning field bus (a flapping IO-34 relay drove ~18 msg/s), this ran execute()
# 25-90x over its intended tick rate on the single engine thread, marking the station config
# dirty on every call — a tenured-heap growth path distinct from the timer/subscriber/null-
# Context leak shapes the kit already lints. Fix: leading-edge time debounce on the callback
# path (Clock.millis() gate), or move periodic writes to a tick and keep changed() input-only.
# See types/issues-and-gotchas.md §I1. [ev: retro live-diagnosis-hardening-deltas Δ1]
# [ev: corpus B1158; BCompressorControl.java:1850-1871,2019-2021]
#
# Usage:  lint-changed-hot-write.sh [--strict] <src-root>
#   Scans *.java under <src-root> (dot-dirs pruned) for a `changed(Property <name>, ...)`
#   override. Only the raw slot-o-matic declaration form
#   `Property <name> = newProperty(<flags>, ...)` is recognized for the non-transient check
#   (the @NiagaraProperty annotation form is not parsed by this lint — false-negative, not
#   false-positive, documented limitation). Reachability is one hop: changed()'s own body,
#   or a zero-arg `execute()` method it calls.
#   Row:  WARN  lint-changed-hot-write  <file>:<line>  <detail>
#   Exit: 0  no WARN (or WARN without --strict) · 1  any WARN under --strict · 3  usage/env, an unscannable source file or a sub-directory find cannot enter
# Empty root: a root with no *.java is `ERROR  lint-changed-hot-write  <root>  no files to scan (K20: wrong path or
# empty scaffold)` on stdout, exit 3.
#
# Implementation: rule package go/internal/lint/rules/changedhotwrite on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md), sharing changed(), bodies, callees and properties with its sibling
# lint through go/internal/lint/rules/hotwrite. This wrapper is the stable entry point: go/go-tool.sh
# `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-changed-hot-write <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# Truth fix T1 (2026-10-24): code is read with comments blanked and literal contents removed (go/internal/javasrc
# Strip + BlankLiterals, shared with the sibling lint): the bash/awk cut each line at its first "//" (inside a
# string too) and read block comments, Javadoc and strings as code, so a Javadoc `{@code execute()}` was taken
# for execute()'s declaration and a "}" in a string ended a body early.
# Truth fix T2 (2026-10-24): changed(), execute() and every callee body come from javasrc.Methods (a zero-arg
# method's body is the one whose signature holds `name()`); the awk took the first `name()` not followed by ";"
# as the declaration (`if (execute())`, `pending(tick())`, `this.device() + "..."`) and counted braces from there.
# Truth fix T4 (2026-10-24): every `Property <name> = newProperty(` of a line is read, its flags from its own
# "newProperty(" to the first "," (across lines when the call wraps); the awk read one declaration per line and its
# flags after the line's last "newProperty(", on that line only.
# VCS-free by design (kit-links L2).
# Mutation: CHW8 -- reading one declaration per line, flags after the last newProperty(, misses bHours and misreads its flags
# Mutation: CHW7 -- taking the first execute() not followed by ";" as the declaration reads if (execute()) instead of the body
# Mutation: CHW6 -- reading comments as code takes /* changed(Property q ...) */ as the callback and misses the real one
# Mutation: CHW2 -- drop the branch-count threshold (N>=6) so a 1-branch changed() false-WARNs
# Mutation: CHW-reset -- a bare `set<X>(` substring also matches `reset<X>(` and false-WARNs
# Mutation: CHW-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
# Mutation: CHW-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-changed-hot-write: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-changed-hot-write "$@"
