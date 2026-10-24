#!/usr/bin/env bash
# lint-lexicon-ascii.sh — ASCII-purity lint for *.lexicon files under a module root.
#
# Niagara reads lexicon files as Latin-1 (ISO-8859-1), then serves the display
# strings to browsers as UTF-8.  Any non-ASCII byte — e.g. the UTF-8 encoding of
# "Presión" — arrives on-station as mojibake ("PresiÃ³n"), silently breaking labels.
# All lexicon content must use only ASCII (U+0000–U+007F); accented characters
# must be represented as Unicode escapes (\uNNNN) per the Java .properties spec.
#
# Real defect trigger: UmbrellaDashboard-rt/module.lexicon values for "Presion"
# were previously authored as "Presión" (UTF-8); this lint would have caught it.
#
# Usage:  lint-lexicon-ascii.sh <module-root>
#
#   Finds every *.lexicon under <module-root>, dot-dirs pruned (D9b).
#   Prints one row per non-ASCII occurrence:
#     FAIL  lexicon-ascii  <file>:<line>  non-ASCII: <hex bytes>
#   Exits: 0  no FAIL (clean) · 1  any FAIL · 3  usage/env
#
# Row format: FAIL  lint-lexicon-ascii  <path>:<line>  non-ASCII: 0x<HH> ...
# Empty root: a module with no *.lexicon is clean (exit 0, no output; LA5, tests/lint-empty-root.bats ER5).
# Implementation: rule package go/internal/lint/rules/lexiconascii on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin`
# builds go/bin/n4lint when it is missing or older than a Go source, then this script execs
# `n4lint lint-lexicon-ascii <args>`. Needs `go` on PATH to build (exit 3 otherwise).
# Declared difference from the bash lint (port 2026-10-24): an unreadable *.lexicon is an env error (exit 3,
# `cannot scan <file>` on stderr); the bash read loop printed the shell's error and went on, exit 0 or 1.
# Truth fix 2026-10-24: every line is scanned, the last one too when the file has no final newline (the bash
# `while read` skipped it, so a non-ASCII byte there passed clean); real trees unchanged (Cliente 7 rows,
# sources/decompiled 0, though 7 of its 13 lexicons have no final newline).
# VCS-free by design; version control is never invoked.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: retro 2026-09-17-umbrelladashboard-module-creation]
# Mutation: LA2 -- remove the non-ASCII byte scan and LA2 stops failing on a UTF-8 accented lexicon
# Mutation: LA-unreadable -- skipping an unreadable lexicon reports the module clean
# Mutation: LA-noeol -- reading lines like `while read` skips the last line when it has no final newline
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-lexicon-ascii: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-lexicon-ascii "$@"
