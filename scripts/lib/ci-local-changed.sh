#!/usr/bin/env bash
# shellcheck shell=bash
# lib/ci-local-changed.sh — `scripts/ci-local.sh --changed [--range <from>..<to>]`: the CI lints over the files a
# change touches, before it is pushed. Sourced only, by ci-local.sh, after lib/ci-local-tools.sh, with ROOT,
# CI_FILE, CI_STEPS, WORK and RANGE set.
#
# #684 failed two CI jobs that were reproducible locally (the kit's bats lint BEX-kit, shellcheck 0.10) because
# the writer had run shellcheck 0.11 and not the kit's lints. (retro 2026-11-15 coordination-stretch C3)
# `ci-steps changed` lists the changed files (deleted ones left out) and writes the ones a check needs under
# $WORK/tree: with --range the bytes of the range's end commit (what a push sends; .githooks/pre-push passes
# the pushed range), without it the work tree, untracked files included, against where HEAD left origin/main.
# Which files shellcheck reads comes from ci.yml's own shellcheck step. The checks, from $WORK/tree:
#   - the pinned shellcheck v0.10.0 over the changed files of CI's shellcheck set together with the files they
#     source and the files that source them (a sourced lib is read with its sourcer), as the change leaves them;
#     CI's whole set, with a stderr line naming <file>:<line>, when a source path cannot be resolved (T14);
#   - lint-bats-errexit, the BEX-kit lint, over the changed tests/**/*.bats (fixtures/ and testdata/ pruned);
#   - lint-bats-heredoc: a here-document line that starts with the test keyword (C8) in those files;
#   - gofmt -l over the changed go/**/*.go;
#   - the gates the changed files trigger (lib/ci-local-gates.sh, run_gates): the CI checks that read the tree.
# Each check runs even after another failed. Exit 0 all clean or nothing to check, 1 a check failed,
# 2 a workflow ci-steps cannot read, 3 a git, write or tool failure.

# changed_check <name> <cmd...> — run one check from the materialized tree and print PASS/FAIL like a step.
changed_check() {
  local name="$1" rc=0
  shift
  printf '== %s\n' "$name"
  (cd "$WORK/tree" && "$@") || rc=$?
  if [ "$rc" -eq 0 ]; then printf 'PASS %s\n' "$name"; else printf 'FAIL %s (exit %d)\n' "$name" "$rc"; CHANGED_FAILED=$((CHANGED_FAILED + 1)); fi
}

# gofmt_clean <file...> — gofmt -l exits 0 on unformatted files, so a listed file is the failure; a syntax error is
# its non-zero exit (an empty list alone would read it as clean: tests/fail-open-idioms.bats FMT-STDERR).
gofmt_clean() {
  local list="$WORK/gofmt.list"
  gofmt -l "$@" > "$list" || return 3
  if [ -s "$list" ]; then printf 'gofmt: not formatted: %s\n' "$(tr '\n' ' ' < "$list")"; return 1; fi
}

run_changed() {
  local plan="$WORK/changed.plan" rc=0 kind file sc=() bt=() gf=()
  CHANGED_FAILED=0
  (cd "$ROOT" && "$CI_STEPS" changed --ci-file "$CI_FILE" --tree "$WORK/tree" --gates ${RANGE:+--range "$RANGE"}) > "$plan" || rc=$?
  case "$rc" in 0) ;; 2) exit 2 ;; *) exit 3 ;; esac
  while IFS=$'\t' read -r kind file; do
    case "$kind" in
      shellcheck) sc+=("$file") ;;
      bats) bt+=("$file") ;;
      gofmt) gf+=("$file") ;;
    esac
  done < "$plan"
  if [ $((${#sc[@]} + ${#bt[@]} + ${#gf[@]})) -eq 0 ]; then
    printf 'ci-local: --changed: no shellcheck, bats-lint or gofmt file changed; running the tree gates\n'
  fi
  if [ "${#sc[@]}" -gt 0 ]; then
    provision_shellcheck
    changed_check "shellcheck ${SC_VER} (${#sc[@]} files)" shellcheck "${sc[@]}"
  fi
  if [ "${#bt[@]}" -gt 0 ]; then
    changed_check "lint-bats-errexit (${#bt[@]} changed bats files)" bash "$ROOT/build-n4-module-kit/toolbelt/lint-bats-errexit.sh" tests
    changed_check "lint-bats-heredoc (${#bt[@]} changed bats files)" "$CI_STEPS" lint-bats-heredoc "${bt[@]}"
  fi
  if [ "${#gf[@]}" -gt 0 ]; then
    changed_check "gofmt -l (${#gf[@]} changed Go files)" gofmt_clean "${gf[@]}"
  fi
  run_gates
  printf 'ci-local: --changed: %d check(s) failed\n' "$CHANGED_FAILED"
  [ "$CHANGED_FAILED" -eq 0 ]
}
