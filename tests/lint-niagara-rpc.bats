#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# lint-niagara-rpc.sh — an @NiagaraRpc method must not be `unrestricted`, and a mutating one must check write
# permission. Every test builds its own source root from tests/fixtures/lint-niagara-rpc (Java files that are read,
# never compiled) and asserts the exact rows and exit code. Annotation attributes (transports, permissions,
# protectedTargets, isSecure): javax/baja/rpc/NiagaraRpc.java:15-23; the permission rule:
# com/tridium/util/NiagaraRpcUtil.java:138-171 (organized/baja/baja/decompiled).
# [ev: corpus B752 §752.2] [ev: user-approved improvement 2026-11-24]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$ROOT/build-n4-module-kit"
  L="$KIT/toolbelt/lint-niagara-rpc.sh"
  FIX="$ROOT/tests/fixtures/lint-niagara-rpc"
  SRC="$BATS_TEST_TMPDIR/src"; mkdir -p "$SRC"
  P="FAIL  lint-niagara-rpc  $SRC"
}

# stage <Fixture>... : copy fixtures into the source root under test
stage() { local f; for f in "$@"; do cp "$FIX/$f.java" "$SRC/"; done; }

# shellcheck disable=SC2154  # $stderr is set by bats run --separate-stderr
@test "NRPC-usage: no argument exits 3 with the usage on stderr, nothing on stdout" {
  run --separate-stderr "$L"
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [[ "$stderr" == "usage: lint-niagara-rpc.sh [--help] <src-root>..." ]]
}

@test "NRPC-help: --help prints the usage and the rules, exit 0" {
  run "$L" --help
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "usage: lint-niagara-rpc.sh [--help] <src-root>..." ]]
  [[ "$output" == *"unrestricted"* && "$output" == *"OPERATOR_WRITE"* && "$output" == *"lint-niagara-rpc: reviewed"* ]]
}

@test "NRPC-badroot: an unknown option or a missing directory exits 3" {
  run "$L" --nope "$SRC"
  [ "$status" -eq 3 ]
  run "$L" "$BATS_TEST_TMPDIR/absent"
  [ "$status" -eq 3 ]
}

@test "NRPC-empty: a root with no *.java is an ERROR row and exit 3, never a clean pass" {
  printf 'notes\n' > "$SRC/README.txt"
  run "$L" "$SRC"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-niagara-rpc  $SRC  no *.java sources found (wrong path or empty source tree)" ]
}

@test "NRPC-unrestricted: permissions = \"unrestricted\" FAILs at the attribute line (exit 1)" {
  stage BUnrestricted
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "$P/BUnrestricted.java:5  unrestricted: readLabel() declares permissions = \"unrestricted\": no framework permission check runs (types/security.md §2.7)" ]
}

@test "NRPC-case: the value is matched case-insensitively, as the station does (NiagaraRpcUtil.java:139)" {
  # Mutation: NRPC-case -- matching "unrestricted" case-sensitively passes permissions = "Unrestricted"
  stage BUnrestricted
  run "$L" "$SRC"
  [ "${lines[1]}" = "$P/BUnrestricted.java:11  unrestricted: readMixedCase() declares permissions = \"Unrestricted\": no framework permission check runs (types/security.md §2.7)" ]
}

@test "NRPC-emptyperm: permissions = \"\" is the same hole (NiagaraRpcUtil.java:138-151 treat it as no requirement)" {
  stage BUnrestricted
  run "$L" "$SRC"
  [ "${lines[2]}" = "$P/BUnrestricted.java:17  unrestricted: readEmpty() declares permissions = \"\": no framework permission check runs (types/security.md §2.7)" ]
  [ "${#lines[@]}" -eq 3 ]
}

@test "NRPC-unguarded: set, invoke, doXxx and a typed setter with no write check FAIL at the call line" {
  stage BUnguarded
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 4 ]
  [ "${lines[0]}" = "$P/BUnguarded.java:9  unguarded-write: setpoint() calls set() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
  [ "${lines[1]}" = "$P/BUnguarded.java:16  unguarded-write: run() calls invoke() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
  [ "${lines[2]}" = "$P/BUnguarded.java:22  unguarded-write: reset() calls doReset() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
  [ "${lines[3]}" = "$P/BUnguarded.java:29  unguarded-write: typed() calls setDouble() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
}

@test "NRPC-guarded: OPERATOR_WRITE, checkCanWrite, requireTargetWrite, hasOperatorWrite or a w permission pass" {
  stage BGuarded
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NRPC-readonly: read-only RPCs and look-alike calls are no row (JDBC setInt(1, v), one-argument set, text, non-RPC)" {
  stage BReadOnly
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NRPC-guard-order: a check that comes after the write does not guard it; one before it does" {
  # Mutation: NRPC-guard-order -- accepting a guard anywhere in the body passes the write that precedes it
  stage BGuardAfter
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$P/BGuardAfter.java:8  unguarded-write: late() calls set() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
}

@test "NRPC-comment: comments, Javadoc and string literals are not code (no phantom RPC, no phantom guard)" {
  # Mutation: NRPC-comment -- matching raw text FAILs the quoted @NiagaraRpc and lets the guard named in a comment pass
  stage BNotCode
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$P/BNotCode.java:16  unguarded-write: guardOnlyInTextIsNoGuard() calls set() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
}

@test "NRPC-waiver: a reviewed marker with a reason waives the RPC (line above, annotation line, signature line)" {
  # Mutation: NRPC-waiver -- ignoring the marker FAILs the three reviewed RPCs
  stage BWaived
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NRPC-waiver-bare: a marker with no reason is its own FAIL and waives nothing" {
  # Mutation: NRPC-waiver-bare -- accepting a bare marker passes the unrestricted RPC without a stated reason
  stage BWaiverBare
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "$P/BWaiverBare.java:5  waiver-no-reason: lint-niagara-rpc: reviewed needs a reason (types/security.md §2.7)" ]
  [ "${lines[1]}" = "$P/BWaiverBare.java:6  unrestricted: lexicon() declares permissions = \"unrestricted\": no framework permission check runs (types/security.md §2.7)" ]
}

@test "NRPC-multi: a qualified @javax.baja.rpc.NiagaraRpc, a bodiless interface method and a multi-line annotation" {
  stage BMulti
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "$output" = "$P/BMulti.java:9  unrestricted: open() declares permissions = \"unrestricted\": no framework permission check runs (types/security.md §2.7)" ]
}

@test "NRPC-nonliteral: a permissions value that is not a string literal is not judged (documented limit)" {
  stage BNonLiteral
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NRPC-edges: return doXxx(, a lambda write, a text block and an anonymous-class declaration" {
  stage BEdges
  run "$L" "$SRC"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 2 ]
  [ "${lines[0]}" = "$P/BEdges.java:15  unguarded-write: returned() calls doReset() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
  [ "${lines[1]}" = "$P/BEdges.java:34  unguarded-write: lambda() calls set() with no OPERATOR_WRITE check before it (types/security.md §2.7)" ]
}

@test "NRPC-tree: the whole fixture tree gives exactly the expected rows, in file order, over several roots" {
  run "$L" "$FIX"
  [ "$status" -eq 1 ]
  [ "$(printf '%s\n' "$output" | grep -c '^FAIL  lint-niagara-rpc  ')" -eq 14 ]
  [ "$(printf '%s\n' "$output" | grep -c 'unrestricted:')" -eq 5 ]
  [ "$(printf '%s\n' "$output" | grep -c 'unguarded-write:')" -eq 8 ]
  [ "$(printf '%s\n' "$output" | grep -c 'waiver-no-reason:')" -eq 1 ]
  mkdir -p "$BATS_TEST_TMPDIR/r2"; cp "$FIX/BGuarded.java" "$BATS_TEST_TMPDIR/r2/"; stage BUnrestricted
  run "$L" "$BATS_TEST_TMPDIR/r2" "$SRC"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 3 ]
}

@test "NRPC-dotdir: a dot-directory and a *-worktrees directory are pruned" {
  mkdir -p "$SRC/.hidden" "$SRC/m-worktrees"
  cp "$FIX/BUnrestricted.java" "$SRC/.hidden/"; cp "$FIX/BUnrestricted.java" "$SRC/m-worktrees/"
  stage BGuarded
  run "$L" "$SRC"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "NRPC-finderr: an unreadable sub-directory is an env error (exit 3, named), never a clean pass" {
  # Mutation: NRPC-finderr -- ignoring the find exit status skips an unreadable directory and reports clean
  mkdir -p "$SRC/locked"; cp "$FIX/BUnrestricted.java" "$SRC/locked/"; stage BGuarded
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$L" "$SRC"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"locked"* ]]
}
