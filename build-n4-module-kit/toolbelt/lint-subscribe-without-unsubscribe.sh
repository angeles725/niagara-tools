#!/usr/bin/env bash
# lint-subscribe-without-unsubscribe.sh — flags a Subscriber leak: a class that calls
# `.subscribe(` but never `unsubscribe`.
#
# A `Subscriber.subscribe(BComponent)` in started() holds a BIDIRECTIONAL reference to the
# watched component: the Subscriber keeps a List<BComponent> and the component keeps the
# Subscriber. Without a matching unsubscribe() in stopped(), the watched component cannot be
# GC'd, the handler keeps firing after the subscriber is removed, and enable/disable cycles
# stack DUPLICATE subscriptions. Use lease() for a transient read; BLink self-manages its
# subscription. See types/logic-authoring.md §Subscribe / unsubscribe symmetry rule.
# [ev: retro module-hardening-failure-modes-deltas Δ3] [ev: corpus B1122]
#
# Implementation: go/internal/lint/rules/subscribeleak on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-subscribe-without-unsubscribe <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_strip_run did).
#
# Usage:  lint-subscribe-without-unsubscribe.sh [--strict] <src-root>
#   Scans *.java under <src-root>, dot-dirs pruned. Line // comments are stripped before
#   matching, so commented-out code does not trigger. Per FILE: a `.subscribe(` call with no
#   `unsubscribe` / `unsubscribeAll` anywhere in the same file is the leak shape.
#   Row:   WARN  lint-subscribe-without-unsubscribe  <file>:<line>  <trimmed source>
#   Exit:  0  no WARN (clean) or WARN-only · 1  any WARN under --strict · 3  usage/env
#
# Heuristic (advisory, hence WARN not FAIL): a file that both subscribes and unsubscribes is
# assumed paired; a file that only subscribes is the leak candidate a reviewer must confirm.
# `.subscribe(` (leading dot) is matched so it never collides with `.unsubscribe(`.
# VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# Mutation: SWU2 -- drop the unsubscribe-absent guard so a file WITH unsubscribe still WARNs
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-subscribe-without-unsubscribe: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-subscribe-without-unsubscribe "$@"
