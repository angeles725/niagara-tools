#!/usr/bin/env bash
# stored-repack.sh — repackage a signed N4 module jar with every entry STORED (no compression).
#
# WHY: Workbench's JarFileSigner re-deflates each entry while re-signing; when the JDK that built
# the jar (WSL OpenJDK 8) and the Workbench JDK (Windows Zulu 8) deflate to different sizes it aborts
# with "ZipException: invalid entry compressed size (expected N but got M)". A clean rebuild does NOT
# fix it (same deflater, same size) and a local `jarsigner` cannot detect it (same deflater again).
# STORED entries make the mismatch impossible by construction and keep the content bytes identical,
# so the existing signature stays valid. Field-confirmed (operator re-signed STORED jars, 2026-09-01).
#
# Entry order matters for a valid signed jar: META-INF/MANIFEST.MF first, then every
# META-INF/<ALIAS>.SF with its .RSA/.DSA/.EC block (NIAGARA4, CODESIGN, SERVER1, ...), then everything else. Cost: a bigger jar (deflate ratio lost).
#
# Usage: stored-repack.sh <in.jar> <out.jar>      (refuses to overwrite an existing out.jar)
# Check: verify-module.sh --stored <out.jar>      (0 "Defl:" entries)
#
# Exits: 0 ok; 1 an input that is no readable jar (missing, not a zip, no manifest, a bad entry), an existing
# output (a file, a directory or a symlink) or an output that cannot be written; 2 usage, or the Go tool
# cannot be built (the bash's "missing tool: zip/unzip" exit 2: the Go port needs neither tool).
# The output is the contract, not its bytes: the entry names and order above, every entry STORED with the
# input's bytes and its sizes in the local header, no directory entries; the rest keep their input order.
#
# Implementation: go/internal/storedrepack (golden runs recorded from the bash body at 91b9e3d6 by
# go/internal/lint/rules/golden/record/record.sh with go/internal/storedrepack/testdata/setup.sh). go/go-tool.sh
# `go_tool_bin` builds go/bin/stored-repack when it is missing or older than a Go source, then this script
# execs it; no Go toolchain or a failed build is exit 2.

set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'stored-repack: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 2; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" stored-repack)" || exit 2
exec "$BIN" "$@"
