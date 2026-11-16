#!/usr/bin/env bash
# run-pure-test.sh — run a ZERO-Baja pure-logic JUnit test standalone in WSL.
#
# WHY: niagaraTest does not run in WSL (needs native bin/test + a dev license), and
# plugin 7.6.17 discovers 0 tests anyway (build-verify.md §Unit tests). The only
# runnable coverage of control/safety logic is a Baja-free `final class` decision core
# tested with JUnit 4 via JUnitCore. The recipe was documented but the jar paths were
# re-hunted every session (retro 2026-09-04-junit-standalone-cached-jar-locations...).
# This removes the hunt and the per-session copy-paste.
#
# USAGE: run-pure-test.sh <module-rt-dir> <pkg.TestClass>
#   <module-rt-dir>  e.g. .../CompPan/CompPan-rt   (holds src/ and srcTest/)
#   <pkg.TestClass>  e.g. com.angeles.CompPan.CompressorControlTest
#
# Compiles the test + any pure sources it references (via -sourcepath) into a TEMP dir
# — never writes into the module tree (a parallel session's working tree is off-limits)
# — then runs JUnitCore.
#
# EXIT: 0 tests passed · 1 a test FAILED (the bite) · 2 usage · 3 environment
#   (no JDK / junit not in the gradle cache — run one gradle build to fetch it — or the test does not
#   compile: a compile error is never reported as the exit-1 bite). [ev: audit-2026-10-03 A1]
#   A JDK without `java` on PATH is exit 127, as a shell reports a missing command.
#
# Go port: with several copies of a jar under ~/.gradle the first in name order is used (find's directory
# order chose it before, so the pick depended on the filesystem).
#
# Implementation: the Go module at <repo>/go (go/cmd/run-pure-test, go/internal/puretest). This script is only
# the launcher: go/go-tool.sh `go_tool_bin` builds go/bin/run-pure-test when the binary is missing or older
# than a Go source, then this script execs it. No Go toolchain or a failed build is also exit 3.
set -u

# readlink -f: resolve a symlinked copy of this file to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'run-pure-test: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" run-pure-test)" || exit 3
# The kit's Java helpers (java/n4kit/…, e.g. the logic-series runner) join the javac source path.
export RUN_PURE_TEST_KIT_JAVA="$here/java"
exec "$BIN" "$@"
