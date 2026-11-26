#!/usr/bin/env bats
# lint-empty-root.bats — a bash lint never reports green on input it did not read. An EMPTY root (or one
# with no file the lint scans: a typo'd sub-path, a wrong module) used to exit 0 with no output on most
# lints. The Go lint engine now owns the rule once: go/internal/lint Spec.EmptyError prints
# `ERROR  <name>  <root>  ...` and exits 3 (the bash libs that first held it, lib/awk-lint.sh and
# lib/scan-files.sh, are removed: every lint that sourced them is an n4lint wrapper).
# ER1 sweeps every lint wrapper over that engine; lint-structure.sh and the documented exceptions below
# are out of scope.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  EMPTY="$BATS_TEST_TMPDIR/empty"; mkdir -p "$EMPTY"
}

# Documented exceptions (ER5 pins them): lint-jasmine-ux.sh and lint-uberjar-api-conflict.sh are advisory
# with "Exit: 0 always" (report-module runs the latter on every module root), lint-lexicon-ascii.sh's
# LA5 contract is "no *.lexicon files exits 0", and its companion lint-lexicon-apostrophe.sh keeps the same
# contract (report-module runs it warnOnly on every artifact, go/internal/reportmodule/members.go). lint-structure.sh is out of scope (Go port). lint-manual-labels.sh
# takes a manual before its ui-dirs (one root argument is its usage error); tests/lint-manual-labels.bats pins it.
# lint-vendored.sh judges an rc dir's vendor/, and most rc dirs have none, so no vendor files exits 0 (#1123).
EXEMPT=" lint-jasmine-ux.sh lint-lexicon-apostrophe.sh lint-lexicon-ascii.sh lint-manual-labels.sh lint-structure.sh lint-uberjar-api-conflict.sh lint-vendored.sh "

# shared_layer_lints: every toolbelt lint-*.sh that is an n4lint wrapper (the Go lint engine prints the ERROR
# row, go/internal/lint Spec.EmptyError), minus EXEMPT; one basename per line.
shared_layer_lints() {
  local l b
  for l in "$KIT"/toolbelt/lint-*.sh; do
    b="${l##*/}"
    case "$EXEMPT" in *" $b "*) continue ;; esac
    # shellcheck disable=SC2016  # the wrapper's literal source text, not an expansion
    grep -qE 'go_tool_bin "\$GO_DIR" n4lint' "$l" || continue
    printf '%s\n' "$b"
  done
}

@test "ER1: every shared-layer bash lint exits 3 with an ERROR row naming an EMPTY root" {
  # Mutation: ER1 -- dropping the empty-root check (Spec.EmptyError) from the Go lint engine names the lints here.
  local b bad="" n=0
  for b in $(shared_layer_lints); do
    n=$((n + 1))
    run timeout 60 bash "$KIT/toolbelt/$b" "$EMPTY"
    if [ "$status" -ne 3 ] || ! printf '%s\n' "$output" | grep -qE "^(ERROR  [^ ]+  $EMPTY  |[^ ]+  ERROR  $EMPTY  )"; then
      bad+=" $b(rc=$status)"
    fi
  done
  [ "$n" -ge 24 ]  # one fewer per lint ported to Go (lint-session-store-lazy-evict, 2026-10-23)
  [ -z "$bad" ] || { printf 'silent on an empty root:%s\n' "$bad"; return 1; }
}

@test "ER2: a root holding only files no lint scans (a wrong sub-path) is the same ERROR, exit 3" {
  printf 'notes\n' > "$EMPTY/README.txt"
  local b bad=""
  for b in $(shared_layer_lints); do
    run timeout 60 bash "$KIT/toolbelt/$b" "$EMPTY"
    [ "$status" -eq 3 ] || bad+=" $b(rc=$status)"
  done
  [ -z "$bad" ] || { printf 'silent on a root with nothing to scan:%s\n' "$bad"; return 1; }
}

@test "ER5: the documented exit-0 exceptions stay exit 0 with no output on an empty root" {
  # Mutation: ER5 -- dropping EmptyOK from rules/uberjarapi (lint-uberjar-api-conflict) turns every report-module run ERROR.
  local b
  for b in lint-jasmine-ux.sh lint-lexicon-apostrophe.sh lint-lexicon-ascii.sh lint-uberjar-api-conflict.sh lint-vendored.sh; do
    run bash "$KIT/toolbelt/$b" "$EMPTY"
    [ "$status" -eq 0 ] && [ -z "$output" ] || { echo "$b: $status $output"; return 1; }
  done
}
