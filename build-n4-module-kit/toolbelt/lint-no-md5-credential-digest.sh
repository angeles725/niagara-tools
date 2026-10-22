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
# Usage:  lint-no-md5-credential-digest.sh [--strict] <src-root>
#   Scans *.java under <src-root>, dot-dirs pruned. Line // comments are stripped before
#   matching, so commented-out code does not trigger. Per FILE: warns when the file contains
#   BOTH MessageDigest.getInstance("MD5") (case-insensitive on MD5) AND a credential context
#   — a token matching BPassword, BCredentials, or an identifier/slot containing
#   password/credential/pin (case-insensitive).
#   Row:   WARN  lint-no-md5-credential-digest  <file>:<line>  <trimmed source>
#   Exit:  0  no WARN (clean) or WARN-only · 1  any WARN under --strict · 3  usage/env
#
# Known limitation: a `//` inside a string literal can hide the call. VCS-free by design.
# Mutation: NMD2 -- drop the credential context guard so MD5-only files also WARN
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-no-md5-credential-digest first 'usage: lint-no-md5-credential-digest.sh [--strict] <src-root>' "$@"

# check <file>: the line of the first MD5 call when the file also has a credential context.
check() {
  # Only files that contain a MessageDigest.getInstance("MD5") call (case-insensitive on MD5).
  printf '%s\n' "$AL_TEXT" | grep -qi 'MessageDigest\.getInstance("MD5")' || return 0
  # Credential context: BPassword, BCredentials, or identifier containing password/credential/pin.
  printf '%s\n' "$AL_TEXT" | grep -qiE '(BPassword|BCredentials|password|credential|pin)' || return 0
  # Report the MD5 line (1-indexed, against the stripped view = same line numbers).
  printf '%s\n' "$AL_TEXT" | grep -ni 'MessageDigest\.getInstance("MD5")' | head -1 | cut -d: -f1
}

# The lib strips // comments into AL_TEXT, walks fail closed, prints the WARN row and the exit code.
awk_lint_strip_run check -name '*.java'
