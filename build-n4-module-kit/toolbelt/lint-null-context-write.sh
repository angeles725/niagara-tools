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
#   Exit: 0  always (advisory) · 3  usage/env
# VCS-free by design.
set -u
LC_ALL=C
export LC_ALL

if [ $# -lt 1 ]; then
    printf 'usage: lint-null-context-write.sh <src-root>\n' >&2
    exit 3
fi
ROOT="$1"
if [ ! -d "$ROOT" ]; then
    printf 'lint-null-context-write: not a directory: %s\n' "$ROOT" >&2
    exit 3
fi

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
done < <(find "$ROOT" -type d -name '.*' -prune -o -type f -name '*.java' -print 2>/dev/null)

exit 0
