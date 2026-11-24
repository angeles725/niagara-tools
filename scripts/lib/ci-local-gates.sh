#!/usr/bin/env bash
# shellcheck shell=bash
# lib/ci-local-gates.sh — the tree gates of `scripts/ci-local.sh --changed`. Sourced only, by ci-local.sh, after
# lib/ci-local-changed.sh, with ROOT, WORK, RANGE, NAMES, BODIES and CHANGED_FAILED set.
#
# CI failed on checks the pre-push replay did not run: go/internal/filesize (#969), go/internal/cardwidth (the
# 120-character kit-doc limit; PR #978 broke in CI only, #983), gofmt, the PR size. They now come from one table,
# go/internal/cisteps/gates.go (change -> check), through `ci-steps changed --gates`: one `gate` row per gate a
# changed file triggers. A row runs a `go test` of the packages that read the tree, the body of the ci.yml step it
# stands for (replayed as CI runs it, so the check cannot differ), or a command. TestEveryCIStepAndTreeGateHasALocalTwin
# fails when a ci.yml run step or a tree-reading Go package has no row there, so a new gate cannot be forgotten here.
# A failed gate counts in run_changed's summary; a soft gate (pr-size: the size:exception label decides in CI) prints
# WARN and counts nothing, whatever its exit (1: over the limit; any other: the size was not measured).

# gate_run <name> <soft|hard> <command> — run one gate from the repo root and print PASS, WARN or FAIL like a step.
gate_run() {
  local name="$1" mode="$2" cmd="$3" out="$WORK/gate.out" rc=0 i hit=0 base=origin/main head=HEAD
  [ -z "$RANGE" ] || { base="${RANGE%%..*}"; head="${RANGE#*..}"; }
  printf '== %s\n' "$name"
  mkdir -p "$WORK/runner-temp"
  if [[ "$cmd" == step:* ]]; then
    for i in "${!NAMES[@]}"; do
      # shellcheck disable=SC2053  # the step name is matched against a glob on purpose
      [[ "${NAMES[$i]}" == ${cmd#step:} ]] || continue
      hit=1
      (cd "$ROOT" && CI=true RUNNER_TEMP="$WORK/runner-temp" bash -e -c "${BODIES[$i]}") > "$out" 2>&1 < /dev/null || rc=$?
      [ "$rc" -eq 0 ] || break
    done
    [ "$hit" -eq 1 ] || { printf 'SKIP %s (this workflow has no step %s)\n' "$name" "${cmd#step:}"; return 0; }
  else
    (cd "$ROOT" && CI_BASE="$base" CI_HEAD="$head" bash -c "$cmd") > "$out" 2>&1 < /dev/null || rc=$?
  fi
  if [ "$rc" -eq 0 ]; then
    [ "$mode" != soft ] || grep -m 1 '^pr-size: total' "$out" || true
    printf 'PASS %s\n' "$name"
    return 0
  fi
  cat "$out"
  if [ "$mode" = soft ]; then
    # A soft gate never blocks a push: exit 1 is its finding, any other exit (no origin/main, an unresolvable --range)
    # means the check did not run to a verdict.
    if [ "$rc" -eq 1 ]; then printf 'WARN %s (exit %d)\n' "$name" "$rc"
    else printf 'WARN %s (exit %d: the size was not measured)\n' "$name" "$rc"; fi
    return 0
  fi
  printf 'FAIL %s (exit %d)\n' "$name" "$rc"
  CHANGED_FAILED=$((CHANGED_FAILED + 1))
}

# run_gates — every `gate` row of the plan (changed.plan, written by run_changed), in table order.
run_gates() {
  local kind name mode cmd
  while IFS=$'\t' read -r kind name mode cmd; do
    if [ "$kind" = gate ]; then gate_run "$name" "$mode" "$cmd"; fi
  done < "$WORK/changed.plan"
}
