#!/usr/bin/env bash
# lint-niagara-rpc.sh — FAILs an @NiagaraRpc method that is `unrestricted` or that writes with no write check.
#
# An RPC (the bajaux `baja.rpc` channel, over fox, box or web) runs for any authenticated user unless the annotation
# or the method body checks a permission; the browser is never the security boundary (types/security.md §2.7).
# Checks (all FAIL): `unrestricted` = permissions "unrestricted" (any case) or "" on the annotation or a nested
# @Protected (NiagaraRpcUtil.java:138-151, :161-166); `unguarded-write` = a body that calls .set(/.invoke(/a typed
# setter with two or more arguments, or a doXxx( call, with no OPERATOR_WRITE, ADMIN_WRITE, hasOperatorWrite(,
# hasAdminWrite(, checkWrite(, checkCanWrite(, requireTargetWrite( or canWrite( before it, and no w/W in a literal
# permissions = "..." of the annotation (the station enforces that one itself, NiagaraRpcUtil.java:138-171).
# Annotation attributes: transports, permissions (default "I"), protectedTargets, isSecure (NiagaraRpc.java:15-23).
# Comments and literal text are no code. Limits: a Slot-o-Matic setter c.setFoo(v) and a write in a helper are not
# seen; a reflection invoke or a JDBC setInt(i, v) with a variable index is a false positive. Waive one RPC with
# `// lint-niagara-rpc: reviewed <reason>` on the line above its @NiagaraRpc or on any line through the method name
# (the reason is mandatory; a bare marker is its own FAIL, `waiver-no-reason`). The rule's reading is the Go
# package go/internal/lint/rules/niagararpc (doc comment and goldens). [ev: corpus B752 §752.2]
#
# Implementation: that rule package on the shared engine go/internal/lint (go/README.md). This wrapper is the stable
# entry point: go/go-tool.sh `go_tool_bin` builds go/bin/n4lint when it is missing or older than a Go source, then this
# script execs `n4lint lint-niagara-rpc <args>`. Needs `go` on PATH to build (exit 3 otherwise).
#
# Usage:  lint-niagara-rpc.sh [--help] <src-root>...
#   Row:  FAIL  lint-niagara-rpc  <file>:<line>  unrestricted|unguarded-write|waiver-no-reason: <detail>
#   Exits:  0  clean · 1  any FAIL · 3  usage/env: no root, an unknown option, a root that is not a directory, a root
#           with no *.java (stdout gets `ERROR  lint-niagara-rpc  <root>  no *.java sources found ...`), a sub-directory
#           the walk cannot enter or an unreadable file
# --help prints the usage and the rules and exits 0. VCS-free by design (kit-links L2).
# Mutation: NRPC-case -- matching "unrestricted" case-sensitively passes permissions = "Unrestricted"
# Mutation: NRPC-guard-order -- accepting a write check anywhere in the body passes the write that precedes it
# Mutation: NRPC-comment -- matching raw text FAILs the quoted @NiagaraRpc and lets a guard named in a comment pass
# Mutation: NRPC-waiver -- ignoring the marker FAILs the three reviewed RPCs
# Mutation: NRPC-waiver-bare -- accepting a bare marker passes the unrestricted RPC without a stated reason
# Mutation: NRPC-finderr -- ignoring the walk error skips an unreadable directory and reports clean
set -u
# Resolve symlinks: a stubbed toolbelt (tests/helpers/stub-toolbelt.bash) links this script from another dir.
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
case "${1:-}" in
  --help|-h)
    printf '%s\n' 'usage: lint-niagara-rpc.sh [--help] <src-root>...' '' \
      'FAIL an @NiagaraRpc method that is unrestricted or that writes without a write check.' \
      '  unrestricted     permissions = "unrestricted" (any case) or permissions = "".' \
      '  unguarded-write  .set(/.invoke(/typed setter with 2+ args, or a doXxx( call, with no OPERATOR_WRITE,' \
      '                   hasOperatorWrite(, checkWrite(, checkCanWrite(, requireTargetWrite( or canWrite( before' \
      '                   it, and no w/W in a literal permissions = "..." of the annotation.' \
      '  Waive one RPC: // lint-niagara-rpc: reviewed <reason>   (line above the @NiagaraRpc, or on any line of it' \
      '  through the method name; the reason is mandatory).' \
      'Rows: FAIL  lint-niagara-rpc  <file>:<line>  <detail>.  Exit 0 clean, 1 any FAIL, 3 usage or env.' \
      'See types/security.md §2.7.'
    exit 0 ;;
esac
GO_DIR="$(cd "$here/../.." && pwd)/go"
[ -f "$GO_DIR/go-tool.sh" ] || { printf 'lint-niagara-rpc: no Go module at %s (go-tool.sh missing)\n' "$GO_DIR" >&2; exit 3; }
# shellcheck disable=SC1091  # go/ helper, resolved at runtime via GO_DIR
. "$GO_DIR/go-tool.sh"
BIN="$(go_tool_bin "$GO_DIR" n4lint)" || exit 3
exec "$BIN" lint-niagara-rpc "$@"
