#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# RED-FIRST tests for build-n4-module-kit/toolbelt/orient-guard.sh
# (worktree-location-guard).
#
# Contract: sdd/build-n4-module-worktree-location-guard spec + design.
#
# orient-guard.sh <module-root>
#
# Env seams:
#   BUILD_N4_LEGAL_ROOT_PREFIX  -- override the legal prefix (always emits WARN + test seam)
#   BUILD_N4_CLIENTE_OVERRIDE   -- '1' → WARN+exit 0; any other value → fail-closed
#
# Row format: PASS|FAIL|WARN  orient-guard  <detail>
# Exit: 0 (PASS/WARN) · 1 (FAIL) · 3 (usage error)
#
# Named mutations (each OGn flips exactly one test after GREEN; revert after proof):
#   OG1: inside-prefix path returns FAIL instead of PASS
#   OG2: prefix compare without trailing slash (ClienteX/ false-positive)
#   OG3: BUILD_N4_CLIENTE_OVERRIDE=0 treated as =1 (bypass without override)
#   OG4: canonicalization skipped (raw input compared)

setup() {
  TMPDIR_T="$(mktemp -d)"
  # Portable legal prefix via the test seam
  PREFIX_DIR="${TMPDIR_T}/Cliente/"
  mkdir -p "$PREFIX_DIR"
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SCRIPT="$KIT/toolbelt/orient-guard.sh"
}

teardown() {
  rm -rf "$TMPDIR_T"
}

# ---------------------------------------------------------------------------
# OG1 — inside legal prefix → PASS / exit 0
# Named mutation: OG1 — remove prefix check so any path returns PASS
# ---------------------------------------------------------------------------
@test "OG1: path inside legal prefix (via seam) -> PASS row + exit 0" {
  target="${PREFIX_DIR}MyModule"
  mkdir -p "$target"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" "$SCRIPT" "$target"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS"*"orient-guard"* ]]
}

# ---------------------------------------------------------------------------
# OG2 — outside prefix, no override → FAIL / exit 1
# FAIL detail must name the resolved path AND BUILD_N4_CLIENTE_OVERRIDE=1
# ---------------------------------------------------------------------------
@test "OG2: path outside legal prefix, no override -> FAIL row + exit 1" {
  outside="${TMPDIR_T}/other_location"
  mkdir -p "$outside"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" "$SCRIPT" "$outside"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"orient-guard"* ]]
  # Resolved path must appear in the FAIL detail
  [[ "$output" == *"$outside"* ]]
  # Override var named verbatim in FAIL detail
  [[ "$output" == *"BUILD_N4_CLIENTE_OVERRIDE=1"* ]]
}

# ---------------------------------------------------------------------------
# OG3 — BUILD_N4_CLIENTE_OVERRIDE=1 outside legal prefix → WARN / exit 0
# Real resolved path must appear in the WARN output
# ---------------------------------------------------------------------------
@test "OG3: BUILD_N4_CLIENTE_OVERRIDE=1 outside prefix -> WARN row + exit 0" {
  outside="${TMPDIR_T}/override_location"
  mkdir -p "$outside"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" BUILD_N4_CLIENTE_OVERRIDE=1 "$SCRIPT" "$outside"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"*"orient-guard"* ]]
  # Real resolved path must appear in WARN output (audit trace)
  [[ "$output" == *"$outside"* ]]
}

# ---------------------------------------------------------------------------
# OG4 — BUILD_N4_CLIENTE_OVERRIDE=0 (non-'1') → fail-closed → FAIL / exit 1
# Named mutation: OG3 — treat =0 as =1 → exit 0 instead of 1
# ---------------------------------------------------------------------------
@test "OG4: BUILD_N4_CLIENTE_OVERRIDE=0 (non-1) outside prefix -> FAIL + exit 1 (fail-closed)" {
  outside="${TMPDIR_T}/override_zero"
  mkdir -p "$outside"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" BUILD_N4_CLIENTE_OVERRIDE=0 "$SCRIPT" "$outside"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"orient-guard"* ]]
}

# ---------------------------------------------------------------------------
# OG5 — symlink arg pointing outside prefix → resolves canonically → FAIL
# Named mutation: OG4 — skip realpath → symlink not resolved → false PASS
# ---------------------------------------------------------------------------
@test "OG5: symlink pointing outside prefix -> canonical resolve -> FAIL + exit 1" {
  real_outside="${TMPDIR_T}/real_outside"
  mkdir -p "$real_outside"
  symlink="${PREFIX_DIR}sneaky_link"
  ln -s "$real_outside" "$symlink"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" "$SCRIPT" "$symlink"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"orient-guard"* ]]
  # Canonical (real) path must appear, not the symlink path
  [[ "$output" == *"$real_outside"* ]]
}

# ---------------------------------------------------------------------------
# OG6 — relative path arg → canonicalized → PASS when resolved inside prefix
# ---------------------------------------------------------------------------
@test "OG6: relative path arg resolved inside prefix -> PASS + exit 0" {
  target="${PREFIX_DIR}RelModule"
  mkdir -p "$target"
  # Run from a directory outside the prefix; pass a path that resolves inside
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" \
    bash -c "cd '$TMPDIR_T' && '$SCRIPT' 'Cliente/RelModule'"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS"*"orient-guard"* ]]
}

# ---------------------------------------------------------------------------
# OG7 — trailing slash on arg → correct compare, no false result
# A trailing slash on the arg must not break the prefix check
# ---------------------------------------------------------------------------
@test "OG7a: trailing slash on inside-prefix arg -> PASS + exit 0" {
  target="${PREFIX_DIR}SlashModule"
  mkdir -p "$target"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" "$SCRIPT" "${target}/"
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS"*"orient-guard"* ]]
}

@test "OG7b: trailing slash on outside-prefix arg -> FAIL + exit 1 (no false PASS)" {
  outside="${TMPDIR_T}/slash_outside"
  mkdir -p "$outside"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" "$SCRIPT" "${outside}/"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"orient-guard"* ]]
}

# ---------------------------------------------------------------------------
# OG8 — ClienteX/ sibling of Cliente/ → FAIL / exit 1 (no false PASS)
# Tests that trailing-slash append prevents the prefix false-positive.
# Named mutation: OG2 — remove trailing-slash append → ClienteX/ matches Cliente/ → false PASS
# ---------------------------------------------------------------------------
@test "OG8: ClienteX sibling directory (e.g. ClienteX/) -> FAIL + exit 1 (not a false PASS)" {
  # seam prefix ends in /Cliente/; ClienteX/ must NOT match it
  clientex_dir="${TMPDIR_T}/ClienteX/"
  mkdir -p "${clientex_dir}Module"
  # Use a prefix that ends in Cliente/ (the seam appends / if missing)
  prefix_cliente="${TMPDIR_T}/Cliente/"
  mkdir -p "$prefix_cliente"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$prefix_cliente" "$SCRIPT" "${clientex_dir}Module"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"orient-guard"* ]]
}

# ---------------------------------------------------------------------------
# OG9 — usage errors → exit 3
# ---------------------------------------------------------------------------
@test "OG9a: zero args -> usage error exit 3" {
  run "$SCRIPT"
  [ "$status" -eq 3 ]
}

@test "OG9b: two args -> usage error exit 3" {
  run "$SCRIPT" "/some/path" "/extra/arg"
  [ "$status" -eq 3 ]
}

# ---------------------------------------------------------------------------
# OG10 — the default legal root is derived from $HOME (no user path in the kit)
# Named mutation: OG10 — a hardcoded home directory as the default prefix
# ---------------------------------------------------------------------------
@test "OG10a: no seam -> the default legal root is \$HOME/modulos_niagara_n4/Cliente/, PASS with no WARN" {
  # Mutation: OG10 -- a hardcoded home directory as the default prefix fails this PASS under a temp HOME
  target="${TMPDIR_T}/modulos_niagara_n4/Cliente/Mod"
  mkdir -p "$target"
  run env -u BUILD_N4_LEGAL_ROOT_PREFIX HOME="$TMPDIR_T" "$SCRIPT" "$target"
  [ "$status" -eq 0 ]
  [ "$output" = "PASS  orient-guard  $(cd "$target" && pwd -P)" ]
}

@test "OG10b: no seam, outside \$HOME's legal root -> FAIL names the derived prefix" {
  outside="${TMPDIR_T}/elsewhere"
  mkdir -p "$outside"
  run env -u BUILD_N4_LEGAL_ROOT_PREFIX HOME="$TMPDIR_T" "$SCRIPT" "$outside"
  [ "$status" -eq 1 ]
  [[ "$output" == *"must be under ${TMPDIR_T}/modulos_niagara_n4/Cliente/;"* ]]
}

@test "OG10c: HOME unset or empty with no seam -> env error exit 3, never a PASS" {
  # Mutation: OG10c -- deriving the prefix from an empty HOME makes it /modulos_niagara_n4/Cliente/
  run env -u BUILD_N4_LEGAL_ROOT_PREFIX -u HOME "$SCRIPT" /tmp
  [ "$status" -eq 3 ]
  [[ "$output" == *"HOME is not set"* ]]
  run env -u BUILD_N4_LEGAL_ROOT_PREFIX HOME= "$SCRIPT" /tmp
  [ "$status" -eq 3 ]
}

@test "OG10d: the seam still wins over \$HOME and stays loud (WARN row)" {
  target="${PREFIX_DIR}Mod"
  mkdir -p "$target"
  run env BUILD_N4_LEGAL_ROOT_PREFIX="$PREFIX_DIR" HOME="$TMPDIR_T/nowhere" "$SCRIPT" "$target"
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "WARN  orient-guard  non-default legal root ${PREFIX_DIR} (test seam)" ]]
  [[ "${lines[1]}" == PASS* ]]
}
