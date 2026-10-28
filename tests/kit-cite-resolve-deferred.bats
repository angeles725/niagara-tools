#!/usr/bin/env bats
# kit-cite-resolve-deferred.bats — an unterminated `[deferred:` tag is still a strict finding: citeparse marks the
# pointer Deferred AND Unterminated (Kind unparsed), and no consumer (citeresolve, buildstatefields) skips a
# Deferred pointer before its unparsed status (review fix-forward of cb022662).

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  R="$REPO/build-n4-module-kit/toolbelt/kit-cite-resolve.sh"
  D="$BATS_TEST_TMPDIR"
  mkdir -p "$D/research" "$D/modules" "$D/clients" "$D/memory" "$D/doc"
  ROOTS=(--kit "$REPO/build-n4-module-kit" --research "$D/research" --modules "$D/modules" --clients "$D/clients" --memory "$D/memory")
}

@test "KCRD1: an unterminated [deferred: tag is an UNPARSED strict finding, exit 1" {
  printf '%s\n' '# t' '' 'A claim. [deferred: retro x' 'next line' > "$D/doc/t.md"
  run bash "$R" --strict "${ROOTS[@]}" "$D/doc/t.md"
  [ "$status" -eq 1 ]
  [[ "$output" == *$'\tUNPARSED\t[unchecked: unterminated tag; deferred]'* ]]
  [[ "$output" == *"strict findings: LIVE 1, HISTORY 0"* ]]
}

@test "KCRD2: a closed [deferred: tag with no finding is not reported as unterminated" {
  printf '%s\n' '# t' '' 'A claim. [deferred: retro x Δ1]' > "$D/doc/t.md"
  run bash "$R" "${ROOTS[@]}" "$D/doc/t.md"
  [[ "$output" != *"unterminated"* ]]
}
