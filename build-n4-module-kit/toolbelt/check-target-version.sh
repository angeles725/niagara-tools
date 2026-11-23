#!/usr/bin/env bash
# check-target-version.sh — is the installation a module jar is about to be copied into the Niagara version the
# jar was compiled against? A jar compiled against 4.14 goes into a 4.14 installation's modules/, not a 4.15 one
# (and the jars a mirror build made are NOT in the installation the mirror was made from until they are copied
# there). [ev: retro deploy-target-version Δ1]
#
# Implementation: the Go module at <repo>/go (go/internal/niagaraversion); go/go-tool.sh builds
# go/bin/check-target-version on demand, then this script execs it.
#
# Usage:  check-target-version.sh <module.jar> <niagara_home | its modules directory>
# Compile version: the highest Tridium <dependency vendorVersion="X.Y"/> floor in the jar's META-INF/module.xml
#         (the niagara-module plugin writes it from the installation the build ran against).
# Install version: vendorVersion of modules/baja.jar's META-INF/module.xml in the installation (never its directory name).
# Output: one row, PASS or FAIL, naming both versions; the major and the minor must be equal.
# Exit:   0 pass · 1 FAIL (another Niagara version) · 2 usage, or a version that cannot be read on either side
#         (not a jar, no Tridium dependency floor, no baja.jar there): nothing is compared, never a pass · 3 no Go toolchain
set -u
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'check-target-version: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" check-target-version)" || exit 3
exec "$BIN" "$@"
