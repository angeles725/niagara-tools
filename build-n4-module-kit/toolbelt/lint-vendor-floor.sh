#!/usr/bin/env bash
# lint-vendor-floor.sh — parses every vendored browser library under <rc-dir>/vendor/**/*.js at the
# Chromium 83 panel floor (acorn, ecmaVersion 2020, classic script) and scans its tokens for APIs
# above that floor. A library that does not parse at the floor breaks the whole page on the HMI
# panel; an API above the floor throws only where it runs (it may be feature-guarded, so WARN).
# Run on every library bump and record the verdict next to the pin in rc/vendor/THIRD-PARTY.md.
# [ev: retro dashboard-frontend-standard Δ16]
#
# Checks (row id `vendor-floor`):
#   syntax   FAIL  the file does not parse as an ES2020 classic script (e.g. `??=` `||=` `&&=`
#                  ES2021, private methods / static blocks / class fields ES2022, top-level await).
#                  An ES2020 file that parses only as an ES module is WARN (load it with
#                  type="module" or vendor the UMD/global build — the panel standard is classic scripts).
#   api      WARN  a call to an API above the floor: .replaceAll( Promise.any( WeakRef
#                  FinalizationRegistry (C84/85), .at( (C92), Object.hasOwn( (C93), .findLast(
#                  .findLastIndex( (C97), structuredClone( (C98), AbortSignal.timeout( (C103),
#                  Object.groupBy( (C117), .toSorted( .toReversed( .toSpliced( (C110). Tokens, not
#                  text: a name inside a comment or a string is never flagged. Tokenized as a classic
#                  (sloppy) script first, as a module only when that fails, so a sloppy-only token
#                  (legacy octal, `with`) does not end the scan early.
#   --strict promotes every WARN row to FAIL.
#
# Tools: node + acorn. Resolution: KIT_NODE (node binary) else `node` on PATH; KIT_ACORN (acorn
# package dir) else toolbelt/eslint/node_modules/acorn (`npm install --prefix toolbelt/eslint`)
# else node's own require resolution (NODE_PATH). A missing tool is never a pass: one SKIP row
# names it and the exit is 4. An rc dir with no vendor/ needs no tool and exits 0.
#
# Walk: a vendor/ that is a symlink is followed (as `find -H` follows a root); a symlink below it is neither
# followed nor listed. The list fails closed, before any tool is asked: an unreadable sub-directory, a
# vendor/ link that leads nowhere, or an rc dir the process may not search (so nobody can tell whether it
# holds a vendor/) is one `lint-vendor-floor: env fault: ...` line naming the path and exit 3.
# [ev: retro go-lint-vendor-floor Δ1]
#
# Usage:  lint-vendor-floor.sh <rc-dir> [--strict]
#   Row:   FAIL|WARN  vendor-floor  <path under rc-dir>:<line>  syntax|api: <reason>
#          SKIP  vendor-floor  <rc-dir>  unavailable: <tool> (<hint>)
# Exit:   0 no FAIL (WARN-only is still 0) · 1 any FAIL · 3 usage/env (a vendor/ that cannot be looked for or
#         listed whole included) · 4 tool unavailable (K20)
#
# Mutation: VF3 -- treat a missing node as a clean pass (exit 0) so VF3 no longer sees exit 4
# Mutation: VF5 -- parse at ecmaVersion latest so the ??= fixture parses and VF5 exits 0
# Mutation: VF6 -- drop the API table so the replaceAll/hasOwn/findLast rows vanish
# Mutation: VF9 -- tokenize as a module only so a sloppy script's API calls after a legacy octal vanish
#
# Implementation: the Go module at <repo>/go (go/internal/vendorfloor). The Go command owns the
# arguments, the tool resolution, the vendored-file list, the API table (apis.go), the verdicts and the
# exit codes; node + acorn stay external tools and run the embedded driver.js, which only parses and
# tokenizes. This wrapper is the stable entry point: go/go-tool.sh `go_tool_bin` builds
# go/bin/lint-vendor-floor when the binary is missing or older than a Go source, then this script execs
# it. Needs `go` on PATH to build; no Go toolchain or a failed build is exit 3. The pins above live in
# the Go source and tests as well.
set -u

# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-vendor-floor: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" lint-vendor-floor)" || exit 3
# The toolbelt this script was run from (not resolved): its eslint/node_modules/acorn is the default acorn.
LINT_VENDOR_FLOOR_TOOLBELT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export LINT_VENDOR_FLOOR_TOOLBELT
exec "$BIN" "$@"
