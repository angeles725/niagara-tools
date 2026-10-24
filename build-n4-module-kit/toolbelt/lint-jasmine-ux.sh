#!/usr/bin/env bash
# lint-jasmine-ux.sh — a -ux module that ships a browser rc/ tree should also ship JS specs.
#
# A -ux profile serving bajaux widgets or a SPA under src/rc/ can pass the whole kit gate
# with ZERO JavaScript tests. WARN when src/rc exists but there is no srcTest/rc/spec/*.js
# (the Jasmine/Karma/grunt-niagara test seam). Advisory: exit 0, WARN row. See dashboard.md DUX-TEST1.
# [ev: retro 2026-09-19-observability]
#
# Usage:  lint-jasmine-ux.sh <ux-module-root>   (a -ux part root containing src/)
#   Row:  WARN  lint-jasmine-ux  <root>  ux ships src/rc but no srcTest/rc/spec/*.js
#   Exit: 0  always (advisory) · 3  usage/env or a spec sub-directory find cannot enter
# Implementation: rule package go/internal/lint/rules/jasmineux on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/n4lint when it is missing or older than a Go source, then this script execs `n4lint lint-jasmine-ux <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
# VCS-free by design.
# Mutation: JUX2 -- drop the empty-specs check and JUX2 stops warning on a rc/ module without specs
# Mutation: JUX-finderr -- ignoring the find exit status reads an unreadable spec directory as no specs (or hides it)
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-jasmine-ux: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-jasmine-ux "$@"
