#!/usr/bin/env bash
# lint-duplicate-helper.sh — flags a second copy of a helper in one module instead of one shared helper (G41).
#
# Logic that already exists in the module (parsing, decoding, formatting, status mapping, ORD building) is
# called or extracted into one shared helper, never pasted as a second copy. Contract: the deferred G41 spec
# in ledger.d/2026-10-22-modular-line-efficient-quality.md. Doctrine: METHODOLOGY.md § rt "One shared helper
# per concern, before a second copy lands"; recipe types/logic.md § Composition & organization.
# [ev: retro modular-line-efficient-quality Δ1] [ev: retro go-lint-duplicate-helper Δ1]
# Advisory WARN: exit 0, 1 under --strict.
#
# Files: every *.java under a -rt, -ux or -wb part's src/ of each module root (the part is matched on the
# listed path: a part's own src/ as the root is compared alone, a GROUP directory compares its modules);
# srcTest/ and build/ are not entered; each root is compared on its own. Read through the shared scanner
# (go/internal/javasrc Strip, Tokens, Decls): comments stripped, a literal or a text block is one token, so a
# copy in a comment or a text block is never code. A method whose name line lies in a slotomatic
# BEGIN/END BAJA AUTO GENERATED CODE region (marker in a comment) is skipped.
# A later definition (walk order, then source order) is a row when
#   (1) its body has the same normalized token sequence (whitespace and comments dropped, identifiers and
#       literals kept) as an earlier body of 8 or more statements (`;` tokens; a for header's count as none), or
#   (2) it is a helper (private or static) with the same name and parameter types as an earlier helper of
#       another class. Not (2): a public or package instance method (an @Override, or a supertype's method
#       implemented without @Override), an anonymous-class method, a constructor, a static factory returning
#       its own class, a one-statement delegate calling the same-named method on another type.
# A row names the first definition only: N copies give N-1 rows. Renamed identifiers are another shape.
# Declared differences from the spec: the check column is the lint ID (kit lint-<name> convention), the row
# adds `(<name>: same body, N statements)` or `(<name>(<types>): same name and parameter types)`, and (2) is
# narrowed as above (retro go-lint-duplicate-helper Δ1). Not in n4lint --all (module roots, Spec.ModuleRoot).
#
# Implementation: rule package go/internal/lint/rules/duplicatehelper on the shared engine go/internal/lint
# (walker, parser, row formatter; go/README.md). go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is
# missing or older than a Go source, then this script execs `n4lint lint-duplicate-helper <args>`. Needs `go`
# on PATH to build (exit 3 otherwise). VCS-free by design.
#
# Usage:  lint-duplicate-helper.sh [--strict] <module-root>...
#   Row:  WARN  lint-duplicate-helper  <file>:<line>  same shape as <file2>:<line2> (<detail>)
#   Exit: 0  no row, or WARN rows without --strict · 1  a WARN row under --strict · 3  usage/env, an empty root, a root with no part src/ file or a sub-directory find cannot enter
# Empty root: a root with no *.java under a part's src/ is an env error (exit 3), never a clean pass; stdout gets
# `ERROR  lint-duplicate-helper  <root>  no *.java under a -rt, -ux or -wb part's src/ (wrong path: pass the module root)`.
# Mutation: DH1 -- dropping the body-shape match passes two identical 10-statement readCp437 bodies clean
# Mutation: DH2 -- dropping the signature match passes two static scale(int,double) helpers clean
# Mutation: DH3 -- dropping the instance-method exemption flags two @Override run() implementations
# Mutation: DH-strict -- giving the rule a non-WARN severity exits 0 under --strict with a row
# Mutation: DH-K20 -- dropping the no-part ErrEmptyRoot reports a wrong-path root as a clean exit 0
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-duplicate-helper: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-duplicate-helper "$@"
