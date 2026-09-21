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
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-subscribe-without-unsubscribe: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-subscribe-without-unsubscribe.sh [--strict] <src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-subscribe-without-unsubscribe.sh [--strict] <src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-subscribe-without-unsubscribe: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

had_warn=0
while IFS= read -r f; do
  # Strip // line comments before matching (a leading-string // is a rare miss, documented).
  stripped=$(sed 's://.*::' "$f")
  # Only files that actually subscribe are candidates.
  printf '%s\n' "$stripped" | grep -q '\.subscribe(' || continue
  # Paired if the file unsubscribes anywhere (unsubscribe / unsubscribeAll).
  printf '%s\n' "$stripped" | grep -q 'unsubscribe' && continue
  # Report the first subscribe line (1-indexed, against the stripped view = same line numbers).
  line=$(printf '%s\n' "$stripped" | grep -n '\.subscribe(' | head -1 | cut -d: -f1)
  src=$(sed -n "${line}p" "$f" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
  printf 'WARN  lint-subscribe-without-unsubscribe  %s:%s  %s\n' "$f" "$line" "$src"
  had_warn=1
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print)

if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
