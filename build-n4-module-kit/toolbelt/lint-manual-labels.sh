#!/usr/bin/env bash
# lint-manual-labels.sh — every quoted UI label in an operator manual exists in the UI source (advisory).
#
# A manual is code that nobody compiles: when a UI label is renamed and the manual still quotes
# the old one, nothing fails. This lint takes every quoted label of the manual — "…", “…” and
# «…» (HTML entities &quot; &ldquo; &rdquo; &laquo; &raquo; decoded; tags and their attribute
# values dropped, so href="x.html" is never a label) — and looks for it in the UI source:
# *.html *.htm *.js *.mjs *.cjs *.json *.lexicon *.properties *.java *.xml under each <ui-dir>
# (dot-dirs and node_modules pruned; \uNNNN escapes and HTML entities decoded). The label must
# EQUAL one whole UI text unit — an HTML text node, a string literal, or a lexicon/properties
# value — or be its prefix ending at a word boundary ("Rearranque" matches "Rearranque 1",
# "Guardado" matches "Guardado (" + n + ")"), ignoring case, accents, collapsed whitespace and a
# trailing ':'. A mid-unit substring is not a match: a manual quoting "Corte" after the UI
# renamed it "Temp. de corte" is the drift to catch. A label the UI builds at runtime is found too:
# a string literal ending in a space, "(" or ":" (a stem) followed, at a word boundary, by one more
# word ("Guardado (N)" for "Guardado (" + n + ")", "HTTP 400" for "HTTP " + status).
# A label not found is a WARN — labels may be paraphrased on purpose, so a human reads each row.
#
# Usage:  lint-manual-labels.sh [--strict] <manual.html|manual.md> <ui-dir> [<ui-dir>...]
#   <ui-dir> is an rc/ tree, a -ux/-rt profile src dir or a lexicon dir (several allowed).
# Exits:  0 clean or WARN-only · 1 any WARN under --strict · 3 usage/env (bad manual or ui-dir, unreadable file,
#         a ui-dir with no UI source file: ERROR row, K20)
#
# Row format:
#   WARN  lint-manual-labels  <manual>:<line>  label "<label>" not found in the UI source
#   SUMMARY  lint-manual-labels  checked=<n> missing=<m>
#   (ML10 pins these prefixes against the emitted rows.)
# VCS-free by design; version control is never invoked (kit-links L2).
# [ev: retro operator-manual-lockstep Δ3]
# Mutation: ML2 -- dropping the not-found check stops the WARN on a renamed label the manual still quotes
# Mutation: ML4 -- dropping the \uNNNN decode makes an accented lexicon label look missing
# Mutation: ML8 -- dropping the word-boundary test lets "Guard" match the unit "Guardar"
# Mutation: ML11 -- opting out of the empty-root ERROR reports every label missing on a typo'd ui-dir (exit 0)
# Mutation: ML12 -- dropping the stem match WARNs on a label the UI joins to a runtime value
#
# Implementation: rule package go/internal/lint/rules/manuallabels on the shared engine go/internal/lint
# (walker, Mode FileFirst parser, Joined TreeRule, row formatter; go/README.md) and go/internal/htmltext
# (Python's errors="replace" read, html.unescape, markup dropping, NFKD/casefold/\s/\w facts generated from
# Python 3.15's unicodedata). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-manual-labels <args>`. Needs `go` on PATH to build (exit 3 otherwise); python3 is no longer used.
# Declared differences from the python3 lint (port 2026-10-24): *-worktrees directories are pruned like dot
# directories; a symlink below a ui-dir is neither followed nor read; a directory or file that cannot be read
# is an env error naming it (exit 3) where python skipped the directory or crashed on the file; tag names of
# an HTML manual's script/style elements match in ASCII case only.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-manual-labels: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-manual-labels "$@"
