#!/usr/bin/env bash
# lint-no-md5-credential-digest.sh — flags MD5 used for credential/password hashing.
#
# MD5 is cryptographically broken and must not be used for credential or password hashing.
# SHA-256 (MessageDigest.getInstance("SHA-256")) is the minimum for any security-sensitive
# digest. MD5 has legitimate non-security uses (checksums, caches), so this lint checks for
# BOTH a MessageDigest.getInstance("MD5") call AND a credential context in the SAME file
# before warning. See types/security.md §9.
# [ev: retro wb-vendor-ux-wave3-vendor-drivers-deltas Δ13]
#
# Implementation: go/internal/lint/rules/nomd5credential on the Go lint engine (go/README.md); this wrapper execs
# `n4lint lint-no-md5-credential-digest <args>`, built on demand by go/go-tool.sh (needs `go` on PATH, exit 3 otherwise). An
# unreadable sub-directory prints the env line, then the listed files' rows, exit 3 (as awk_lint_strip_run did).
#
# Usage:  lint-no-md5-credential-digest.sh [--strict] <src-root>
#   Scans *.java under <src-root>, dot-dirs pruned. Per FILE: warns when the file contains
#   BOTH MessageDigest.getInstance("MD5") AND a credential context — BPassword, BCredentials,
#   or an identifier or literal (a slot name) containing password/credential/pin; both are
#   matched ASCII case-insensitively over the whole text. Comments and Javadoc never count
#   (literal text does), and a `//` inside a string never hides the rest of a line.
#   Row:   WARN  lint-no-md5-credential-digest  <file>:<line>  <trimmed source>
#   Exit:  0  no WARN (clean) or WARN-only · 1  any WARN under --strict · 3  usage/env
#
# VCS-free by design.
# Mutation: NMD5 -- matching the raw line (no comment blanking) WARNs a commented MD5 call or a comment-only context
# Mutation: NMD2 -- drop the credential context guard so MD5-only files also WARN
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-no-md5-credential-digest: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-no-md5-credential-digest "$@"
