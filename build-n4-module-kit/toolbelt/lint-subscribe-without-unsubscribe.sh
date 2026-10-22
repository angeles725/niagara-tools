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
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-subscribe-without-unsubscribe first 'usage: lint-subscribe-without-unsubscribe.sh [--strict] <src-root>' "$@"

# check <file>: the line of the first subscribe when the file never unsubscribes.
check() {
  # Only files that actually subscribe are candidates.
  printf '%s\n' "$AL_TEXT" | grep -q '\.subscribe(' || return 0
  # Paired if the file unsubscribes anywhere (unsubscribe / unsubscribeAll).
  printf '%s\n' "$AL_TEXT" | grep -q 'unsubscribe' && return 0
  # Report the first subscribe line (1-indexed, against the stripped view = same line numbers).
  printf '%s\n' "$AL_TEXT" | grep -n '\.subscribe(' | head -1 | cut -d: -f1
}

# The lib strips // comments into AL_TEXT, walks fail closed, prints the WARN row and the exit code.
awk_lint_strip_run check -name '*.java'
