#!/usr/bin/env bash
# run-niagara-test.sh — decide PASS or FAIL for a Niagara test run from its captured output, never its exit code.
#
# WHY: `bin\test.exe <module> -v:2` and gradle `niagaraTest` both report success (exit 0, BUILD SUCCESSFUL) on
# runs that ran no test or whose test module the test NRE rejected (types/moduleTest.md § Known gotchas;
# retro testing-parity-btestng). This tool reads the output instead (kit spec C40, Go tool G33).
#
# Usage: run-niagara-test.sh verdict [<log>|-]     (no <log> or - reads standard input)
# Contract: PASS only when the log holds at least one `Total tests run: N` line and every one has N > 0 and
# `Failures: 0`. FAIL on a summary with 0 tests, failures or no Failures count; a `Failures: N>0` line outside a
# summary; `invalid test target(s)`; `failed signature validation`; a malformed summary; no summary at all; or a
# line longer than 1 MiB (the rest of the log is unread). One `PASS|FAIL  run-niagara-test  <detail>` row per
# summary line and per other reason, then the verdict row.
# Exits: 0 PASS; 1 FAIL; 2 usage; 3 unreadable log, or the Go tool cannot be built.
# Phase 1 is the verdict only: running the test (scratch copy, :clean, test.exe, jar backup and restore) is owed.
#
# Implementation: go/internal/niagaratest (goldens in its testdata/, hand-written: no bash tool to record from).
# go/go-tool.sh `go_tool_bin` builds go/bin/run-niagara-test when it is missing or older than a Go source, then
# this script execs it; no Go toolchain or a failed build is exit 3.

set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'run-niagara-test: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" run-niagara-test)" || exit 3
exec "$BIN" "$@"
