#!/usr/bin/env bats
# host-budget.bats — black-box tests of go/cmd/host-budget, the shared-host gate scripts/ci-local.sh runs
# before any CI step (quality round 8 T47; spec: build-n4-module-kit/ledger.d/2026-10-22-r8-t47-host-budget.md,
# retro build-n4-module-kit/retros/2026-10-22-r8-t47-host-budget.md).
# The binary is built through go/go-tool.sh go_tool_bin, the same path ci-local.sh uses. Fixtures are fake
# cgroup v2 roots and a fake /proc; the disk rows stat a real tmpdir, so every case passes --disk-pct 100
# except the disk case, and the injected-statfs cases (95% → FAIL) live in the Go table tests.
# [ev: retro r8-t47-host-budget]

setup_file() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime via ROOT
  . "$ROOT/go/go-tool.sh"
  HB="$(go_tool_bin "$ROOT/go" host-budget)"
  export HB
}

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  FX="$ROOT/tests/fixtures/host-budget"
  CACHE="$BATS_TEST_TMPDIR/gocache"
  mkdir -p "$CACHE"
}

# hb <cgroup-fixture> [args...] — run host-budget on a fixture host with an empty cache and neutral disk limits.
hb() {
  local cg="$1"; shift
  run "$HB" --cgroup-root "$FX/$cg" --proc-root "$FX/proc" --gocache "$CACHE" --mount "$BATS_TEST_TMPDIR" \
    --disk-pct 100 "$@"
}

@test "HB1: a cgroup above 50% of pids.max is a FAIL row and exit 1 (init.scope 30000/48129)" {
  # Mutation: HB1 -- comparing against the WARN threshold only reports the incident shape as WARN, exit 0.
  hb cg-incident
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL pids:init.scope 30000/48129"* ]]
}

@test "HB2: exactly half of pids.max is not a FAIL (WARN above 40%, exit 0); --strict turns WARN into exit 1" {
  # Mutation: HB2 -- using >= instead of > fails the exact half.
  hb cg-half
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN pids:init.scope 24064/48128"* ]]
  [[ "$output" != *FAIL* ]]
  hb cg-half --strict
  [ "$status" -eq 1 ]
}

@test "HB3: pids.max = max is unlimited and produces no row" {
  hb cg-calm
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS pids:init.scope 700/48129"* ]]
  [[ "$output" != *system.slice* ]]
  [[ "$output" == *"PASS threads 900/320861"* ]]
}

@test "HB4: a 21 GB (sparse) GOCACHE is a WARN with the go clean -cache hint, exit 0" {
  truncate -s 21G "$CACHE/00-sparse-d"
  hb cg-calm
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN gocache "*"hint: go clean -cache"* ]]
}

@test "HB5: a mount above --disk-pct is a FAIL row, and / is always checked" {
  run "$HB" --cgroup-root "$FX/cg-calm" --proc-root "$FX/proc" --gocache "$CACHE" --mount "$BATS_TEST_TMPDIR" \
    --disk-pct 0
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL disk:$BATS_TEST_TMPDIR "* ]]
  [[ "$output" == *" disk:/ "* ]]
}

@test "HB6: an unreadable cgroup root and a usage error exit 3" {
  hb no-such-root
  [ "$status" -eq 3 ]
  [[ "$output" == *"host-budget: cannot read cgroup root"* ]]
  run "$HB" --pids-pct 150
  [ "$status" -eq 3 ]
  run "$HB" stray-argument
  [ "$status" -eq 3 ]
}

@test "HB7: toolbelt/host-budget.sh, called through a symlink from another cwd, runs the same Go tool" {
  # Mutation: HB7 -- deleting the wrapper (or an unresolved here=) loses the toolbelt entry point.
  # [ev: retro r8-t47-host-budget Δ1]
  local w="$ROOT/build-n4-module-kit/toolbelt/host-budget.sh"
  [ -f "$w" ]
  ln -s "$w" "$BATS_TEST_TMPDIR/hb-link.sh"
  cd "$BATS_TEST_TMPDIR"
  run bash "$BATS_TEST_TMPDIR/hb-link.sh" --cgroup-root "$FX/cg-incident" --proc-root "$FX/proc" --gocache "$CACHE" \
    --mount "$BATS_TEST_TMPDIR" --disk-pct 100
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL pids:init.scope 30000/48129"* ]]
  run bash "$BATS_TEST_TMPDIR/hb-link.sh" stray-argument
  [ "$status" -eq 3 ]
}
