#!/usr/bin/env bash
# lint-absolute-claims.sh — reports the kit's absolute claims (never, always, silently, forbidden, throws) that cite
# only corpus inference or retros, for a human check against Tridium-authored sources (METHODOLOGY K33).
#
# The evidence of a line is its claim unit: the Markdown bullet or paragraph that holds it (a bullet runs to the next
# bullet, heading or blank line). A unit is strong when one of its [ev: ...] tags starts with code, course, devguide,
# CERT or javap, or its text holds a `.java:<n>` line cite. A row is each line of a unit that is not strong holding one
# of the words (whole word, any case), with the first word, the line (cut at 120 characters) and the unit's tags (`-`
# for none). The report has no verdict: rows are INFO and the exit is 0, --strict included.
# Scans every *.md under each root, normally build-n4-module-kit/types. [ev: retro n4-course-as-exemplar-source Δ3]
#
# Implementation: rule package go/internal/lint/rules/absoluteclaims on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-absolute-claims <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-absolute-claims.sh [--strict] <types-dir>...
#   Row:  INFO  lint-absolute-claims  <file>:<line>  <word>: <line text>  <[ev: ...] tags | ->
#   Exit: 0  always after a scan (a report, no verdict) · 3  usage/env, a root with no *.md
# Mutation: AC1 -- reading the tags of the line alone lists a bullet whose code tag sits on its continuation line
# Mutation: AC2 -- counting a corpus tag as strong drops the corpus-only claim from the report
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-absolute-claims: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-absolute-claims "$@"
