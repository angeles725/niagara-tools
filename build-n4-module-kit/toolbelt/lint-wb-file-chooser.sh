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
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-wb-file-chooser first 'usage: lint-wb-file-chooser.sh [--strict] <wb-src-root>' "$@"

# check <file>: the line of the file-chooser dialog call, when the file has no station-space picker.
check() {
  local line
  # Only files that call BWbFieldEditor.dialog( are candidates.
  printf '%s\n' "$AL_TEXT" | grep -q 'BWbFieldEditor\.dialog(' || return 0
  # Only files whose dialog call may receive BOrd.NULL (file-space default) are candidates.
  printf '%s\n' "$AL_TEXT" | grep -q 'BOrd\.NULL' || return 0
  # Exempt if the file already uses a station-component chooser or targetType facet.
  printf '%s\n' "$AL_TEXT" | grep -q 'BComponentChooser' && return 0
  printf '%s\n' "$AL_TEXT" | grep -q 'targetType' && return 0
  # Report the first line that has both the dialog call and BOrd.NULL on the same line;
  # fall back to the first dialog line if the argument spans multiple lines.
  line=$(printf '%s\n' "$AL_TEXT" | grep -n 'BWbFieldEditor\.dialog(' | grep 'BOrd\.NULL' | head -1 | cut -d: -f1)
  if [ -z "$line" ]; then
    line=$(printf '%s\n' "$AL_TEXT" | grep -n 'BWbFieldEditor\.dialog(' | head -1 | cut -d: -f1)
  fi
  printf '%s\n' "$line"
}

# The lib strips // comments into AL_TEXT, walks fail closed, prints the WARN row and the exit code.
awk_lint_strip_run check -name '*.java'
