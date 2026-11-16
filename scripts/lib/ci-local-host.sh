#!/usr/bin/env bash
# shellcheck shell=bash
# lib/ci-local-host.sh — the host-budget gate scripts/ci-local.sh runs before any step or --changed check (CL10-CL15b).
# Sourced only, by ci-local.sh, after go/go-tool.sh, with ROOT set. Split from ci-local.sh unchanged (retro 2026-11-15
# coordination-stretch T25: ci-local.sh past ~200 lines).

DISK_PCT=95  # host-budget disk limit, percent used, for / and /mnt/c (retro 2026-11-15 coordination-stretch T2)

# check_host_budget — refuse to start the steps while go/cmd/host-budget reports a FAIL row: a cgroup above
# 50% of a numeric pids.max (or the system thread count above 50% of threads-max), or / or /mnt/c above DISK_PCT
# (95%) used. WARN rows (pids above 40%, disk above 80%, GOCACHE above 20 GB) are printed and do not block.
# A cgroup root that does not exist at all (macOS, cgroup-less containers) is a stderr note and the run
# proceeds, as the pre-T47 inline loop did; a root that exists but cannot be read stays exit 3.
# CI_LOCAL_HOST_BUDGET_ARGS holds extra host-budget arguments (word-split; no spaces inside one argument),
# so the tests can point --proc-root, --gocache and --disk-pct at fixtures instead of this machine.
# Mutation: CL10 -- dropping this call lets the steps run on a host near its fork limit.
check_host_budget() {
  local bin rc=0 root="${CI_LOCAL_CGROUP_ROOT:-/sys/fs/cgroup}" extra=()
  if [ ! -e "$root" ]; then
    printf 'ci-local: no cgroup root at %s; host budget not checked\n' "$root" >&2
    return 0
  fi
  read -r -a extra <<< "${CI_LOCAL_HOST_BUDGET_ARGS:-}"
  # shellcheck disable=SC2153  # ROOT is set by ci-local.sh, not a misspelt root
  bin="$(go_tool_bin "$ROOT/go" host-budget)" || exit 3
  # The user's final disk rule of 2026-11-16: /mnt/c up to 95% used (a 1 TB C: at 90% still had 90 GB free; it
  # replaced a 40 GB, then a 101 GB free-space floor, which host-budget keeps as --disk-min-free).
  # Mutation: CL15b -- dropping --disk-pct restores the 90% default.
  "$bin" --cgroup-root "$root" --disk-pct "$DISK_PCT" ${extra[@]+"${extra[@]}"} >&2 || rc=$?
  case "$rc" in
    0) ;;
    1) printf 'ci-local: host budget exceeded (FAIL row above); wait for other sessions and retry\n' >&2; exit 4 ;;
    *) printf 'ci-local: host-budget failed (exit %d: 3 = cannot read host / usage, other = crash/signal)\n' "$rc" >&2; exit 3 ;;
  esac
}
