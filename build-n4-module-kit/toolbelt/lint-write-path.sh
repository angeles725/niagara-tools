#!/usr/bin/env bash
# lint-write-path.sh — write-path matrix coverage lint for Niagara N4 modules (Campaign 8 PR19, B816).
# [ev: retro campaign8-write-path]
#
# Checks that every @NiagaraProperty with Flags.OPERATOR (or "o") has a ROW in
# docs/write-path-matrix.md (D16: row-presence only; CHECK12 is bog-audit's concern).
# With --bog, adds link-traced dashboard/RoomPanel target slots to the required set; the bog (plain
# XML or a zipped config.bog) is read through go/internal/xmlsrc, so a malformed or unreadable bog is an
# ERROR row + exit 3, never an empty set (R8-D12).
#
# Usage:  lint-write-path.sh <module-root> [--bog <config.bog>] [--matrix <path>] [--strict]
#
# Module-root convention:
#   - If <root>/src exists: scan it directly (single-profile mode).
#   - If not: scan every *-rt / *-ux / *-wb / *-se profile subdir found immediately
#     under <root>, reporting each under its profile name.
#   - If no Java source is found anywhere: ERROR + exit 3.
#
# Matrix resolution (when --matrix is not given):
#   1. Walk up from <module-root> looking for a dir whose docs/write-path-matrix.md
#      contains at least one table row (line starting with |).
#   2. Walk stops at the first .git directory (vcs root) or filesystem root.
#   3. If no valid matrix is found: ERROR + exit 3.
#
# Audit-call check (WARN): every servlet write handler — a doPost / doPut / doDelete / doPatch
# method declared in a scanned src file — must record the write in the audit store before it
# answers success. Heuristic (file scope, comments stripped outside string literals, literal
# contents blanked): the handler file must contain a call
# to an identifier containing "audit" (e.g. appendAudit(, auditLog.record(, buildAuditEntry();
# getters (get*/is*) and comment mentions do not count; a handler that delegates the write to
# another class must audit there AND call it from the handler file. Advisory for legacy modules;
# --strict promotes it to exit 1. Contract: types/security.md § Servlet write audit.
# [ev: retro servlet-write-audit Δ2]
#
#   Row format:  FAIL  lint-write-path  <module>  slot <name>: no matrix row
#                WARN  lint-write-path  <file>:<line>  write handler <method>: no audit call
#   Error format: lint-write-path  ERROR  <module-root>  <reason>
#   Exits: 0 all covered (STALE/DRIFT/audit WARN advisory) · 1 any uncovered, or any STALE/DRIFT/audit WARN under --strict · 3 usage/env/missing-matrix or a sub-directory find cannot enter (K20)
#
# A comment mention of a slot name does NOT satisfy the matrix row requirement (R19.3).
# An unreadable Java or matrix file is an env error too (exit 3).
# Dot-directories pruned (D9b). VCS-free by design. The checks run in Go: go/internal/lint/rules/writepath
# (n4lint lint-write-path), byte-identical to this script's former bash/awk/python body on its goldens.
# kit-links.bats L2 enforces the no-version-control rule on all toolbelt scripts.
# [ev: corpus B816]
# Mutation: WP-stale-concept-decoy -- moves concept marker strip after the test, making an HTML-comment [concept] produce a false STALE
# Mutation: WP-stale-perrow -- removes per-row check, allowing a [concept]-marked row to exempt a co-resident plain stale row
# Mutation: WP-drift-decoy -- removes HTML-comment strip, causing a commented [concept] to produce a false DRIFT
# Mutation: WP-audit -- drops the audit-call WARN, so a servlet write handler that never records the write passes
# Mutation: WP-audit-ok -- drops the audit-call search, so an audited handler gets a false WARN
# Mutation: WP-audit-str-ok -- strips comments without tracking string literals, so "*/*" hides the audit call
# Mutation: WP-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports the slot covered
# Mutation: WP-finderr-matrix -- ignoring the matrix-root find status drops names from the covered set
set -u

# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-write-path: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-write-path "$@"
