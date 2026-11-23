#!/usr/bin/env bash
# lint-poll-scroll-restore.sh — WARNs a poll repaint that clears a container in a file with a scroller and no scroll restore.
#
# A page that rebuilds a scrolling view on every poll puts the operator back at the top unless the repaint saves and
# restores the scroll position (types/frontend-standard.md §5: after the render, after an async render, on later frames;
# operator input cancels the restore). One WARN per *.html/*.js file, at the first clear, when all hold:
#   1. a function named as the callback of a `setInterval(<name>,` / `setTimeout(<name>,` poll, or a function that poll
#      calls (one level), clears a container: `.innerHTML|.textContent|.innerText = ""`, `.replaceChildren()`,
#      jQuery `.empty()` or `.html("")`;
#   2. the file shows a scrolling container: CSS `overflow: auto|scroll` (also -x/-y) or `style.overflow[XY] = "auto|scroll"`;
#   3. the file reads or sets no scroll position at all: scrollTop, scrollLeft, scrollTo, scrollBy, scrollIntoView, scroll.
# Comments and string contents are not code (go/internal/jssrc); an .html file is read through its <script> contents.
# [ev: issue #909] (retro R9 ΔB1)
#
# Known limitations (advisory, one lexical file at a time): the false-negative side is wide on purpose. A scroller
# styled in a separate stylesheet shows no container here and is not judged; any scroll call anywhere in the file
# silences the WARN, whatever it scrolls; only a callback named in the poll call and the functions it calls by name
# are read. Skipped: node_modules, vendor and dist trees. Not a report-module member: a WARN-only advisory run by hand.
#
# Implementation: rule package go/internal/lint/rules/pollscrollrestore on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-poll-scroll-restore <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-poll-scroll-restore.sh [--strict] <ux-src-root>
#   Row:  WARN  lint-poll-scroll-restore  <file>:<line>  poll repaint '<fn>()' clears a container ...
#   Exit: 0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, a root with no *.html/*.js
# Mutation: PSR1 -- ignoring a scroll call anywhere in the file WARNs a page that restores its scroll
# Mutation: PSR2 -- WARNing without a scrolling container in the file flags every page that clears a small list
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-poll-scroll-restore: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-poll-scroll-restore "$@"
