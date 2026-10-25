#!/usr/bin/env bash
# lint-bats-errexit.sh — FAILs a bats assertion errexit cannot see: a mid-test && list of assertions or a bare `! cmd`.
#
# bats runs a test body under errexit, and bash ignores errexit for every command of an && list except the last
# one and for a command negated with !. Mid-test, `[ "$status" -eq 1 ] && [[ "$output" == *x* ]]` therefore never
# fails its test when the status is wrong, and `! grep -q x f` never fails it either: only the test's last
# command counts. Fix: one assertion per line; `run ! cmd` (bats 1.5+) or `! cmd || return 1` for a negation.
# Left alone: the test's last command, an || tail (`|| fail`, `|| skip`, `|| true`), a guard whose last element
# is an action (`[ -n "$p" ] && ln -s "$p" d`), conditions, `run bash -c '...'` strings and heredoc bodies.
# Only @test bodies are read (not setup or helper functions). [ev: retro r10-bats-list-hazard Δ1]
#
# Implementation: rule package go/internal/lint/rules/batserrexit on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-bats-errexit <args>`.
# Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-bats-errexit.sh [--strict] <bats-dir>...
#   Row:  FAIL  bats-and-list  <file>:<line>  mid-test && list of assertions: ...
#         FAIL  bats-negation  <file>:<line>  mid-test ! negation: ...
#   Exit: 0  clean · 1  any FAIL · 3  usage/env, a root with no *.bats (testdata/ and fixtures/ pruned), an unreadable file
# Mutation: BEX1 -- accepting a mid-test && list of assertions passes the fixture's split-me test
# Mutation: BEX2 -- accepting a bare mid-test negation passes the fixture's negate-me test
# Mutation: BEX3 -- flagging the last command or an || tail FAILs the fixture's clean forms
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-bats-errexit: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-bats-errexit "$@"
