#!/usr/bin/env bash
# lint-guard-pins.sh — Guard-pin meta-check for the kit's own lint scripts (C11 T4, B833).
# Scope: toolbelt/lint-*.sh only (D4b — non-lint toolbelt scripts are out of T4's scan).
#
# Every lint-*.sh must declare at least one header line (its leading comment block):
#   # Mutation: <fixture-id> -- <what it flips>
# Each declared id must resolve to an @test "<id>:" or "@test "<id> " entry in tests/*.bats.
# A lint with no declaration is a latent unpinned guard — the defect C11 kept catching by hand
# (S23-and, EW-s22-nondo, vacuous WP-stale decoys). [ev: design build-n4-module-campaign11 §D4]
#
# Mutation: T4-nomutation -- removing this line makes lint-guard-pins.sh itself a vacuously-clean guard
#
# Row format:
#   MATCH  lint-guard-pins  <script>:<line>  mutation <id> -> tests/<batsfile>.bats:<line>
#   WARN   lint-guard-pins  <script>:<line>  mutation <id> -> no fixture
#   WARN   lint-guard-pins  <script>  no # Mutation: line (K24(7): add '# Mutation: <fixture-id> -- <what it flips>')
#
# Usage:  lint-guard-pins.sh [--strict] <repo-root>
#   <repo-root> holds tests/ and build-n4-module-kit/ (or toolbelt/); a kit root alone is exit 3.
# Exits:  0 clean or WARN-only (no --strict) · 1 any WARN under --strict · 3 usage/env (K20)
# Dot-directories pruned (D9b). VCS-free by design (kit-links L2). LC_ALL=C.
# [ev: retro campaign11-lint-guard-pins]
#
# Implementation: the Go module at <repo>/go (go/internal/guardpins). This wrapper is the stable entry
# point: go/go-tool.sh `go_tool_bin` builds go/bin/lint-guard-pins when the binary is missing or older
# than a Go source, then this script execs it. Needs `go` on PATH to build; no Go toolchain or a failed
# build is exit 3. The T4-nomutation pin lives in the Go source and tests as well.
set -u

# readlink -f: resolve a symlinked copy of this file (report-module test toolbelts) to the real kit, so ../../go exists.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-guard-pins: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" lint-guard-pins)" || exit 3
exec "$BIN" "$@"
