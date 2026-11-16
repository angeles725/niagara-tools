#!/usr/bin/env bats
# ci-local-host.bats — scripts/ci-local.sh, the host-budget gate it runs before any step (CL10-CL15b).
# Split from tests/ci-local.bats (same setup), which holds the step-replay cases. [ev: retro 2026-10-22-ci-local-replay]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  CL="$ROOT/scripts/ci-local.sh"
  FX="$ROOT/tests/fixtures/ci-local"
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  # Neutral host-budget limits so the gate cases read only fixtures: the fake /proc, an empty Go cache and a
  # disk limit no mount can exceed. Without them CL10-CL12 would read this machine's threads, cache and disks.
  mkdir -p "$TMPDIR_T/gocache"
  HB_NEUTRAL="--proc-root $ROOT/tests/fixtures/host-budget/proc --gocache $TMPDIR_T/gocache --disk-pct 100 --disk-min-free 1"
}
teardown() { rm -rf "$TMPDIR_T"; }

# Host pids guard (quality round 8 T41): on 2026-10-22 the WSL init.scope cgroup hit pids.max 48,129
# ("fork rejected by pids controller") and every session on the host aborted. The replay refuses to
# start its steps while any cgroup under the root is above half of its pids.max. [ev: retro r8-t41-host-safety]
fake_cgroup() { # fake_cgroup <name> <current> <max>
  mkdir -p "$TMPDIR_T/cg/$1"
  printf '%s\n' "$2" > "$TMPDIR_T/cg/$1/pids.current"
  printf '%s\n' "$3" > "$TMPDIR_T/cg/$1/pids.max"
}

@test "CL10: a cgroup above 50% of its pids.max refuses the run (exit 4) before any step runs" {
  # Mutation: CL10 -- dropping the pre-run pids check lets the step run on a host near its fork limit.
  fake_cgroup init.scope 30000 48129
  fake_cgroup system.slice 16 max
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [ "$status" -eq 4 ]
  [[ "$output" == *"init.scope"*"30000/48129"* ]]
  [[ "$output" != *"ran-after"* ]]
}

@test "CL11: cgroups at or below 50% of pids.max, or unlimited, let the run proceed" {
  # Mutation: CL11 -- comparing against the literal 'max' or using >= refuses a healthy host.
  fake_cgroup init.scope 24064 48128
  fake_cgroup system.slice 99999 max
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS runs after a failure"* ]]
}

@test "CL12: --list is a read-only classification and never runs the pids check" {
  fake_cgroup init.scope 48000 48129
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" run "$CL" --list --ci-file "$FX/semantics.yml"
  [ "$status" -eq 0 ]
}

@test "CL13: a missing cgroup root is a stderr note and the run proceeds (hosts without /sys/fs/cgroup)" {
  # Mutation: CL13 -- treating a missing root as a host-budget error (exit 3) stops ci-local on macOS and
  # cgroup-less containers, which the pre-T47 inline loop let run.
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/no-such-root" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" \
    run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [ "$status" -eq 0 ]
  [[ "$output" == *"ci-local: no cgroup root at $TMPDIR_T/no-such-root; host budget not checked"* ]]
  [[ "$output" == *"PASS runs after a failure"* ]]
}

@test "CL14: a cgroup root that exists but cannot be read as one stays a tool error (exit 3)" {
  # Mutation: CL14 -- skipping the check on any unreadable root hides a broken host behind a green run.
  printf 'not a directory\n' > "$TMPDIR_T/cg-file"
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg-file" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL" \
    run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [ "$status" -eq 3 ]
  [[ "$output" == *"host-budget failed (exit 3: 3 = cannot read host / usage, other = crash/signal)"* ]]
  [[ "$output" != *"ran-after"* ]]
}

@test "CL15: CI_LOCAL_HOST_BUDGET_ARGS is appended to the host-budget call" {
  # Mutation: CL15 -- dropping the extra args makes CL10-CL12 read the real disks, threads and Go cache.
  fake_cgroup init.scope 16 48129
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" CI_LOCAL_HOST_BUDGET_ARGS="$HB_NEUTRAL --disk-min-free 999999T --mount $TMPDIR_T" \
    run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [ "$status" -eq 4 ]
  [[ "$output" == *"FAIL disk:$TMPDIR_T "* ]]
}

@test "CL15b: the host-budget call fails a disk above 95% used, not 90%, and the row names the free bytes" {
  # Mutation: CL15b -- dropping --disk-pct 95 from check_host_budget brings back the 90% default that refused a
  # 1 TB C: with 90 GB free. The user's final rule of 2026-11-16 is 95% used for /mnt/c (it replaced a 40 GB, then a
  # 101 GB free-space floor). (retro 2026-11-15 coordination-stretch T2)
  fake_cgroup init.scope 16 48129
  # no floor of the test's own: the disk rows carry the script's floor (their verdict depends on this host's disks)
  CI_LOCAL_CGROUP_ROOT="$TMPDIR_T/cg" \
    CI_LOCAL_HOST_BUDGET_ARGS="--proc-root $ROOT/tests/fixtures/host-budget/proc --gocache $TMPDIR_T/gocache --mount $TMPDIR_T" \
    run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [[ "$output" == *"disk:$TMPDIR_T "*"% used, "*"GB free = "*" bytes, fail above 95%)"* ]]
  [[ "$output" != *"fail below"* ]]   # no free-space floor by default
}
