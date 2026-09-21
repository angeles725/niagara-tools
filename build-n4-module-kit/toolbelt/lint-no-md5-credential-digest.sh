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
LC_ALL=C
export LC_ALL

STRICT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --strict) STRICT=1; shift ;;
    --) shift; break ;;
    -*) printf 'lint-no-md5-credential-digest: unknown flag: %s\n' "$1" >&2
        printf 'usage: lint-no-md5-credential-digest.sh [--strict] <src-root>\n' >&2
        exit 3 ;;
    *) break ;;
  esac
done

if [ $# -lt 1 ]; then
  printf 'usage: lint-no-md5-credential-digest.sh [--strict] <src-root>\n' >&2
  exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
  printf 'lint-no-md5-credential-digest: not a directory: %s\n' "$ROOT" >&2
  exit 3
fi

had_warn=0
while IFS= read -r f; do
  # Strip // line comments before matching.
  stripped=$(sed 's://.*::' "$f")
  # Only files that contain a MessageDigest.getInstance("MD5") call (case-insensitive on MD5).
  printf '%s\n' "$stripped" | grep -qi 'MessageDigest\.getInstance("MD5")' || continue
  # Credential context: BPassword, BCredentials, or identifier containing password/credential/pin.
  printf '%s\n' "$stripped" | grep -qiE '(BPassword|BCredentials|password|credential|pin)' || continue
  # Report the MD5 line (1-indexed, against the stripped view = same line numbers).
  line=$(printf '%s\n' "$stripped" | grep -ni 'MessageDigest\.getInstance("MD5")' | head -1 | cut -d: -f1)
  src=$(sed -n "${line}p" "$f" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
  printf 'WARN  lint-no-md5-credential-digest  %s:%s  %s\n' "$f" "$line" "$src"
  had_warn=1
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print)

if [ "$had_warn" -eq 1 ] && [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
