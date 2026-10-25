#!/usr/bin/env bash
# run-niagara-test.sh — decide PASS or FAIL for a Niagara test run from its captured output, never its exit code.
#
# WHY: `bin\test.exe <module> -v:2` and gradle `niagaraTest` both report success (exit 0, BUILD SUCCESSFUL) on
# runs that ran no test or whose test module the test NRE rejected (types/moduleTest.md § Known gotchas;
# retro testing-parity-btestng). This tool reads the output instead (kit spec C40, Go tool G33).
#
# Usage: run-niagara-test.sh verdict [<log>|-]     (no <log> or - reads standard input)
#        run-niagara-test.sh verdict --testng <testng-results.xml> [<log>|-]
# Contract: PASS only when the log holds at least one `Total tests run: N` line and every one has N > 0 and
# `Failures: 0`. FAIL on a summary with 0 tests, failures or no Failures count; a `Failures:` key outside a
# summary with a count above 0 or a value that is no count; `invalid test target(s)`; `failed signature
# validation`; a malformed summary; no summary at all; or a line longer than 1 MiB (the rest is unread). One `PASS|FAIL  run-niagara-test  <detail>` row per
# summary line and per other reason, then the verdict row.
# --testng: `test.exe <module> -v:2 -output:<dir>` prints its `Total tests run` summaries on STDERR; its stdout
# holds only shutdown thread dumps. <dir>/testng-results.xml is the authoritative verdict; a console log
# (capture stderr, or 2>&1, never stdout alone) is a cross-check. PASS only when its root has total > 0,
# failed = 0 and skipped = 0, no test-method (config methods included: a failed setupStation skips its tests) has status
# FAIL, and the root counts match the test-methods. A missing, malformed, truncated or over-64 MiB file is
# FAIL (exit 1), with a row per failed test-method (20 at most, then "and N more"). A <log>, when given, still
# fails the run on its rejections and failure counts; its missing summary line is no reason.
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
