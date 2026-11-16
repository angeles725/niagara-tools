#!/usr/bin/env bats
# ci-local-pins.bats — scripts/ci-local.sh, the tool versions ci.yml pins and the options around them (CL20-CL22).
# Split from tests/ci-local.bats (same setup), which holds the step-replay cases. [ev: retro 2026-10-22-ci-local-replay]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  CL="$ROOT/scripts/ci-local.sh"
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
}
teardown() { rm -rf "$TMPDIR_T"; }

# C8: CI pins bats-core 1.10.0, which refuses a file bats 1.14 accepts (a here-document line starting with the
# test keyword is read as a duplicate case). The replay runs the version ci.yml's install step names.
# pinned_bats_ci <version> <sha256> — a workflow whose install step pins that bats and whose one step runs it.
pinned_bats_ci() {
  printf 'name: CI\non: pull_request\njobs:\n  bats:\n    runs-on: ubuntu-latest\n    steps:\n' > "$TMPDIR_T/pin.yml"
  printf '      - name: Install bats-core %s\n        run: |\n          BATS_VER=%s\n          BATS_SHA256=%s\n          sudo true\n' \
    "$1" "$1" "$2" >> "$TMPDIR_T/pin.yml"
  printf '      - name: bats run\n        run: bats --formatter tap tests/x.bats\n' >> "$TMPDIR_T/pin.yml"
}

# fake_bats_tarball <version> — a bats-core-v<version>.tar.gz in the cache whose bin/bats names itself; prints its sha256.
fake_bats_tarball() {
  local d="$TMPDIR_T/src/bats-core-$1/bin"
  mkdir -p "$d" "$TMPDIR_T/cache"
  printf '#!/bin/sh\necho "pinned bats %s: $*"\n' "$1" > "$d/bats"
  chmod +x "$d/bats"
  tar -czf "$TMPDIR_T/cache/bats-core-v$1.tar.gz" -C "$TMPDIR_T/src" "bats-core-$1"
  sha256sum "$TMPDIR_T/cache/bats-core-v$1.tar.gz" | cut -d' ' -f1
}

@test "CL20: a bats step runs the bats version the install step pins, not the bats on PATH" {
  # Mutation: CL20 -- dropping the provision_bats call runs the PATH bats (1.14 here, 1.10 in CI).
  local sha; sha="$(fake_bats_tarball 9.9.9)"
  pinned_bats_ci 9.9.9 "$sha"
  mkdir -p "$TMPDIR_T/bin"
  printf '#!/bin/sh\necho "path bats: $*"\n' > "$TMPDIR_T/bin/bats"
  chmod +x "$TMPDIR_T/bin/bats"
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$TMPDIR_T/pin.yml" --cache-dir "$TMPDIR_T/cache" --only 'bats run'
  [ "$status" -eq 0 ]
  [[ "$output" == *"pinned bats 9.9.9: --formatter tap tests/x.bats"* ]]
  [[ "$output" != *"path bats"* ]]
}

@test "CL21: a cached bats tarball with another sha256 is refused (exit 3) before the step runs" {
  fake_bats_tarball 9.9.9 >/dev/null
  pinned_bats_ci 9.9.9 0000000000000000000000000000000000000000000000000000000000000000
  run "$CL" --ci-file "$TMPDIR_T/pin.yml" --cache-dir "$TMPDIR_T/cache" --only 'bats run'
  [ "$status" -eq 3 ]
  [[ "$output" == *"sha256 mismatch"*"bats-core-v9.9.9.tar.gz"* ]]
  [[ "$output" != *"pinned bats"* ]]
}

@test "CL22: --range without --changed, and --changed with --list or --only, are usage errors (exit 2)" {
  run "$CL" --range a..b
  [ "$status" -eq 2 ]
  run "$CL" --changed --list
  [ "$status" -eq 2 ]
  run "$CL" --changed --only x
  [ "$status" -eq 2 ]
  [[ "$output" == *"--changed [--range <from>..<to>]"* ]]
}

# T12: `--only "Go gofmt"` replays the Go step without the npm step before it, but the step exports
# KIT_REQUIRE_JS_TOOLS=1, so a host without acorn failed the vendorfloor tests. (retro 2026-11-15 coordination-stretch T12)
# js_root <node exit> — a copy of the replay in a root with no npm tools installed, a node that answers
# require.resolve with <node exit>, a workflow with CI's npm step and a Go step that exports the variable.
js_root() {
  R="$TMPDIR_T/root"
  rm -rf "$R"
  mkdir -p "$R/scripts/lib" "$R/.github/workflows" "$TMPDIR_T/nodebin" "$TMPDIR_T/cg/init.scope"
  cp "$ROOT/scripts/ci-local.sh" "$R/scripts/"
  cp "$ROOT"/scripts/lib/*.sh "$R/scripts/lib/"
  ln -s "$ROOT/go" "$R/go"
  # neutral host-budget limits (see tests/ci-local-host.bats): this case reads only fixtures
  mkdir -p "$TMPDIR_T/gocache"
  HB_NEUTRAL="--proc-root $ROOT/tests/fixtures/host-budget/proc --gocache $TMPDIR_T/gocache --disk-pct 100 --disk-min-free 1"
  printf '#!/bin/sh\nexit %s\n' "$1" > "$TMPDIR_T/nodebin/node"
  chmod +x "$TMPDIR_T/nodebin/node"
  printf '16\n' > "$TMPDIR_T/cg/init.scope/pids.current"
  printf '48129\n' > "$TMPDIR_T/cg/init.scope/pids.max"
  # shellcheck disable=SC2016  # the workflow text keeps its ${...} for the replayed step to expand
  printf '%s\n' 'name: CI' 'on: pull_request' 'jobs:' '  go:' '    runs-on: ubuntu-latest' '    steps:' \
    '      - name: Install the pinned npm tools (npm ci from toolbelt/eslint/package-lock.json)' \
    '        run: echo npm-ran npm ci --prefix x' \
    '      - name: Go gofmt + vet + test (go/)' '        run: |' '          export KIT_REQUIRE_JS_TOOLS=1' \
    '          echo "require=${KIT_REQUIRE_JS_TOOLS:-unset}"' > "$R/.github/workflows/ci.yml"
}

@test "CL23: --only of the Go step on a host without acorn unsets KIT_REQUIRE_JS_TOOLS with a NOTE; with acorn it stays" {
  # Mutation: CL23 -- dropping js_tools_guard keeps require=1 and the vendorfloor tests fail on a host without acorn.
  js_root 1
  PATH="$TMPDIR_T/nodebin:$PATH" CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" \
    run "$R/scripts/ci-local.sh" --only 'Go gofmt'
  [ "$status" -eq 0 ]
  [[ "$output" == *"NOTE: node or acorn is not installed and the npm step is not part of this run"*"KIT_REQUIRE_JS_TOOLS unset for 'Go gofmt + vet + test (go/)'"* ]]
  [[ "$output" == *"require=unset"* ]]
  [[ "$output" != *"npm-ran"* ]]
  # a host that resolves acorn keeps CI's strict setting, with no NOTE
  js_root 0
  PATH="$TMPDIR_T/nodebin:$PATH" CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" \
    run "$R/scripts/ci-local.sh" --only 'Go gofmt'
  [ "$status" -eq 0 ]
  [[ "$output" == *"require=1"* ]]
  [[ "$output" != *"NOTE: node or acorn"* ]]
  # the whole replay runs the npm step first: nothing to change there
  js_root 1
  PATH="$TMPDIR_T/nodebin:$PATH" CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" \
    run "$R/scripts/ci-local.sh"
  [[ "$output" == *"npm-ran"* ]]
  [[ "$output" == *"require=1"* ]]
}
