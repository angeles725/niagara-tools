#!/usr/bin/env bash
# lint-vendored.sh — WARNs a vendored file of <rc-dir>/vendor/ whose version the known-defects table lists, and
# a vendored file that rc/vendor/VENDORED.md does not list.
#
# A vendored library carries defects that the build, the tests and lint-vendor-floor never show (a vendored
# three.js r160 OrbitControls zoomed wrong at devicePixelRatio < 1, which users reported; three.js fixed it in
# r161, per issue #1123). rc/vendor/VENDORED.md records each vendored file; this lint reads it and the kit's
# known-defects table (data/vendored-defects.md) and says which file to upgrade.
# [ev: issue #1123]
#
# VENDORED.md is a markdown table, one row per vendored file (path below rc/vendor/, forward slashes):
#   | File | Library | Version | Source URL |
#   |---|---|---|---|
#   | OrbitControls.js | three.js | r160 | https://example.org/three/r160/examples/OrbitControls.js |
# Version is an r-number (r161) or a semver (1.4.2). THIRD-PARTY.md, LICENSE*, LICENCE*, COPYING* and NOTICE*
# files are notes, not vendored code. Fix a WARN by upgrading the library or by adding the missing row.
#
# Rows (all WARN, row id lint-vendored):
#   a file at a version in the table          names the library, the version, the defect and the fix
#   a file no row lists                        says which row to add
#   vendor/ files and no VENDORED.md           one row, says how to create it
#   a row with an empty cell, or a version that is no r-number or semver
#   an r-number against a semver range         "check by hand" (they have no order)
# An rc dir with no vendor/ files is clean. The table must be readable whole: an unreadable or malformed one is
# exit 3. Pass --defects <file> to use another table (the last one given wins).
#
# Implementation: rule package go/internal/lint/rules/vendored on the shared engine go/internal/lint
# (go/README.md). This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when
# it is missing or older than a Go source, then this script execs `n4lint lint-vendored --defects <table> <args>`.
# Needs `go` on PATH to build (exit 3 otherwise). A RootRule, so n4lint --all leaves it out.
#
# Usage:  lint-vendored.sh [--strict] [--defects <file>] <rc-dir>
#   Row:  WARN  lint-vendored  <rc-dir>/vendor/<file>  <library> <version> (VENDORED.md:<line>) has a known defect: ...; fix: ...
# Exit:   0  clean or WARN-only · 1  any WARN under --strict · 3  usage/env, an unreadable table, a directory it cannot list
# Mutation: VN1 -- read the version as text so r99 sorts above r161 and r160 stops being flagged
# Mutation: VN2 -- count THIRD-PARTY.md and LICENSE files as vendored code and flag them unlisted
# VCS-free by design.
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-vendored: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-vendored --defects "$here/../data/vendored-defects.md" "$@"
