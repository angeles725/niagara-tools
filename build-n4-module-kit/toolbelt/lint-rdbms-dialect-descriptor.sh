#!/usr/bin/env bash
# lint-rdbms-dialect-descriptor.sh — flags a history export whose database dialect is fixed in code, and a
# Postgres JDBC URL in a module that ships no BRdbms dialect of its own (G88).
#
# The native rdb module ships four dialects (MySQL, Oracle, SQL Server, HSQLDB) and no Postgres one; a history
# export takes its descriptor from the database (`db.getHistories().getExportDescriptorType().getInstance()`),
# and every database type needs its own licence feature, so a descriptor named in code skips the dialect the
# station chose. Evidence and the fix: types/cloud-connector.md § 11. Contract: the G88 spec in
# ledger.d/2026-10-23-r8-u9-c68-c71.md. [ev: retro devguide415-r2-shipped-libraries Δ8] [ev: retro devguide415-r3-drivers-integration Δ6] [ev: retro doc-coverage-audit Δ16]
# Advisory WARN: exit 0, 1 under --strict.
#
# Files: every *.java under a -rt part's src/ of each module root (srcTest/ and build/ are not entered; each root
# is read on its own). Comments are blanked (go/internal/javasrc); the `new` match is on code with literals
# blanked too, so a copy in a comment, a Javadoc or a string never matches.
#   (1) `new B(SqlServer|MySQL|Oracle|HsqlDb)HistoryExport(`  -> `hard-coded dialect descriptor`
#   (2) a `jdbc:postgresql:` string literal, when no class of the module `extends BRdbms`  -> `postgres jdbc url`
# Declared difference from the spec: the check column is the lint ID (kit lint-<name> convention), and each row
# ends with the guide section. Not in n4lint --all (module roots, Spec.ModuleRoot) and not run by report-module.
#
# Implementation: rule package go/internal/lint/rules/rdbmsdialect on the shared engine go/internal/lint (walker,
# parser, row formatter; go/README.md). go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or
# older than a Go source, then this script execs `n4lint lint-rdbms-dialect-descriptor <args>`. Needs `go` on PATH
# to build (exit 3 otherwise). VCS-free by design.
#
# Usage:  lint-rdbms-dialect-descriptor.sh [--strict] <module-root>...
#   Row:  WARN  lint-rdbms-dialect-descriptor  <file>:<line>  <hard-coded dialect descriptor|postgres jdbc url>: <fix> (types/cloud-connector.md § 11)
#   Exit: 0  no row, or WARN rows without --strict · 1  a WARN row under --strict · 3  usage/env, an empty root or a root with no -rt part src/ file
# Empty root: a root with no *.java under a -rt part's src/ is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-rdbms-dialect-descriptor  <root>  no *.java under a -rt part's src/ (wrong path: pass the module root)`.
# Mutation: RD1 -- dropping the fixed-dialect match passes `new BSqlServerHistoryExport()` clean
# Mutation: RD2 -- ignoring a BRdbms subclass WARNs a Postgres URL of a module that ships its own dialect
# Mutation: RD-K20 -- dropping the no-part ErrEmptyRoot reports a wrong-path root as a clean exit 0
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-rdbms-dialect-descriptor: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-rdbms-dialect-descriptor "$@"
