#!/usr/bin/env bash
# lint-null-context-write.sh — flags component writes/invokes made with a null Context.
#
# A write with a null Context — `parent.set(prop, val, null)` or `comp.invoke(act, arg, null)` —
# is a DOUBLE hazard: (1) it skips the AuditHistory record (no who-changed-what), and (2) it
# grants BPermissions.all, bypassing RBAC. A servlet/dashboard write must carry the invoking
# user's Context: `new BasicContext(BUser.getUserFromSubject(...))`. See types/security.md §1.
# Advisory (WARN): some framework-internal writes legitimately pass null. [ev: retro 2026-09-19-security-model]
#
# Usage:  lint-null-context-write.sh <src-root>
#   Row:  WARN  lint-null-context-write  <file>:<line>  null-Context write: <source>
#   Exit: 0  always (advisory) · 3  usage/env or a sub-directory find cannot enter
# VCS-free by design.
# Mutation: NCW2 -- drop the ,null match and NCW2 stops warning on set(...,null)
# Mutation: NCW-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
set -u
# shellcheck disable=SC1091  # sibling lib, resolved at runtime via BASH_SOURCE
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/awk-lint.sh"
awk_lint_parse lint-null-context-write raw 'usage: lint-null-context-write.sh <src-root>' "$@"
awk_lint_list -name '*.java'

while IFS= read -r f; do
    ln=0
    while IFS= read -r line || [ -n "$line" ]; do
        ln=$((ln + 1))
        code=${line%%//*}
        if printf '%s' "$code" | grep -Eq '\.(set|setInt|setBoolean|setDouble|invoke)\(.*,[[:space:]]*null[[:space:]]*\)'; then
            trimmed=$(printf '%s' "$line" | sed 's/^[[:space:]]*//')
            printf 'WARN  lint-null-context-write  %s:%s  null-Context write: %s\n' "$f" "$ln" "$trimmed"
        fi
    done < "$f"
done < "$AL_TMP/files"

awk_lint_exit 0
