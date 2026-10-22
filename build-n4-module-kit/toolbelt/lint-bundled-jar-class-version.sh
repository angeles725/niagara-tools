#!/usr/bin/env bash
# lint-bundled-jar-class-version.sh — checks bundled ext-jars under a module source tree for
# Java 9+ bytecode (class-file major version > 52).
#
# N4 modules must target Java 8 (class-file major = 52). A bundled ext-jar containing Java 9+
# bytecode (major > 52) throws a RAW UnsupportedClassVersionError at module load — a LinkageError
# that bypasses catch(Exception) and fails the entire station load.
# [ev: retro module-hardening-failure-modes-deltas Δ11 BLD2]
#
# Usage:  lint-bundled-jar-class-version.sh <module-root>
#   Scans *.jar under <module-root>, PRUNING build/, .git, and other dot-dirs (vendored/bundled
#   jars in the source tree — NOT gradle build outputs). If none found → clean exit 0.
#   Multi-release jars: META-INF/versions/ entries are skipped to avoid false positives (a
#   multi-release jar legitimately bundles higher-version bytecode under META-INF/versions/<N>/).
#   Samples up to 20 class entries per jar; stops after the first FAIL in each jar.
#   Row:   FAIL  lint-bundled-jar-class-version  <jar>  class <entry> is major <N> (> 52 = Java 8) — bundled Java 9+ bytecode throws UnsupportedClassVersionError at load
#   Exit:  0  no FAIL (clean) · 1  any FAIL · 3  usage/env, a sub-directory find cannot enter, or a jar that
#          cannot be read (not a zip, unreadable, or a sampled class it cannot extract: "cannot scan <jar>: <reason>")
#
# Mutation: BJCV2 -- drop the major > 52 check and BJCV2 stops flagging Java 9+ bytecode
# Mutation: BJCV-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
#
# Implementation: the Go module at <repo>/go (go/internal/jarversion, archive/zip: no unzip/od/awk per
# class). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/lint-bundled-jar-class-version when the binary is missing or older than a Go source, then this
# script execs it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3. The BJCV2
# and BJCV-finderr pins live in the Go source and tests as well.
set -u

here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-bundled-jar-class-version: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" lint-bundled-jar-class-version)" || exit 3
exec "$BIN" "$@"
