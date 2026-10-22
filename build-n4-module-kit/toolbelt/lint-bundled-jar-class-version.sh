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
#   Exit:  0  no FAIL (clean) · 1  any FAIL · 3  usage/env or a sub-directory find cannot enter
#
# Mutation: BJCV2 -- drop the major > 52 check and BJCV2 stops flagging Java 9+ bytecode
# Mutation: BJCV-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-bundled-jar-class-version raw 'usage: lint-bundled-jar-class-version.sh <module-root>' "$@"
awk_lint_list --prune build -name '*.jar'

if ! command -v unzip >/dev/null 2>&1; then
    printf 'lint-bundled-jar-class-version: unzip not found\n' >&2
    exit 3
fi
if ! command -v od >/dev/null 2>&1; then
    printf 'lint-bundled-jar-class-version: od not found\n' >&2
    exit 3
fi

SAMPLE_LIMIT=20
fail=0

while IFS= read -r jar; do
    n=0
    while IFS= read -r entry; do
        n=$((n + 1))
        [ "$n" -gt "$SAMPLE_LIMIT" ] && break
        # Read class-file major version: bytes 6-7 (after magic CAFEBABE[4] + minor[2])
        _bytes=$(unzip -p "$jar" "$entry" 2>/dev/null | od -An -tu1 -j6 -N2)
        _major=$(printf '%s' "$_bytes" | awk '{
            if (NF >= 2) { m = $1 * 256 + $2 } else if (NF == 1) { m = $1 + 0 } else { m = -1 }
            if (m > 52) { print m }
        }')
        if [ -n "$_major" ]; then
            printf 'FAIL  lint-bundled-jar-class-version  %s  class %s is major %s (> 52 = Java 8) — bundled Java 9+ bytecode throws UnsupportedClassVersionError at load\n' \
                "$jar" "$entry" "$_major"
            fail=1
            break
        fi
    done < <(unzip -Z1 "$jar" 2>/dev/null | grep '\.class$' | grep -v '^META-INF/versions/' | head -n "$SAMPLE_LIMIT")
done < "$AL_TMP/files"

awk_lint_exit "$fail"
