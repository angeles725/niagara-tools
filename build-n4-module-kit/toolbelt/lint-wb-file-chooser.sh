#!/usr/bin/env bash
# lint-wb-file-chooser.sh — flags a -wb BWbFieldEditor.dialog(...BOrd.NULL) that opens the
# file-system chooser (C:\) instead of a station-component chooser.
#
# A BWbFieldEditor.dialog(parent, target, BOrd.NULL) falls back to the generic file-system
# chooser (opens C:\). In a station context this is useless — the user cannot navigate to a
# station ORD via the file dialog. The correct pattern is either a BComponentChooser (explicit
# station-component picker) or the targetType facet that narrows the dialog automatically.
# See types/wb-widgets.md "station-space picker" rule.
# [ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8]
#
# Usage: lint-wb-file-chooser.sh [--strict] <wb-src-root>
#   Scans *.java under <wb-src-root>, dot-dirs pruned. Line // comments stripped before matching
#   so commented-out code does not trigger.
#   Per FILE: a line with BWbFieldEditor.dialog( + BOrd.NULL anywhere in the file, when the same
#   file has neither BComponentChooser nor targetType, is the file-chooser shape.
#   Row:   WARN  lint-wb-file-chooser  <file>:<line>  <trimmed source>
#   Exit:  0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env
#
# Heuristic (advisory, hence WARN not FAIL): a file that uses BComponentChooser or targetType
# is assumed to be using the correct station-space picker; a file that calls the dialog with only
# BOrd.NULL is the file-chooser candidate a reviewer must confirm.
# VCS-free by design; kit-links.bats L2 enforces the no-VCS rule.
# Mutation: WFC2 -- remove the BComponentChooser/targetType-absent guard so a file WITH BComponentChooser still WARNs
set -u
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-wb-file-chooser: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-wb-file-chooser.sh [--strict] <wb-src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-wb-file-chooser.sh [--strict] <wb-src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-wb-file-chooser: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

had_warn=0
while IFS= read -r f; do
  # Strip // line comments before matching (a leading-dot // is a rare miss, documented).
  stripped=$(sed 's://.*::' "$f")
  # Only files that call BWbFieldEditor.dialog( are candidates.
  printf '%s\n' "$stripped" | grep -q 'BWbFieldEditor\.dialog(' || continue
  # Only files whose dialog call may receive BOrd.NULL (file-space default) are candidates.
  printf '%s\n' "$stripped" | grep -q 'BOrd\.NULL' || continue
  # Exempt if the file already uses a station-component chooser or targetType facet.
  printf '%s\n' "$stripped" | grep -q 'BComponentChooser' && continue
  printf '%s\n' "$stripped" | grep -q 'targetType' && continue
  # Report the first line that has both the dialog call and BOrd.NULL on the same line;
  # fall back to the first dialog line if the argument spans multiple lines.
  line=$(printf '%s\n' "$stripped" | grep -n 'BWbFieldEditor\.dialog(' | grep 'BOrd\.NULL' | head -1 | cut -d: -f1)
  if [ -z "$line" ]; then
    line=$(printf '%s\n' "$stripped" | grep -n 'BWbFieldEditor\.dialog(' | head -1 | cut -d: -f1)
  fi
  src=$(sed -n "${line}p" "$f" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
  printf 'WARN  lint-wb-file-chooser  %s:%s  %s\n' "$f" "$line" "$src"
  had_warn=1
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print)

if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
