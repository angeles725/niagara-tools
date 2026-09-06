#!/usr/bin/env bats
# C11 T4 RED — a meta-lint over the kit's own guards: toolbelt/lint-guard-pins.sh.
# Scope = toolbelt/lint-*.sh only (D9b dot-dir prune). Every lint must declare at least
# one header line `# Mutation: <fixture-id> -- <what it flips>` (ASCII --), and each id
# must resolve to an @test "<id>" (or "<id>:") in tests/*.bats. A lint with no # Mutation:
# line, or a mutation naming a missing fixture, is a latent unpinned guard — the defect
# this campaign kept catching by hand (S23-and, EW-s22-nondo, vacuous WP-stale/WP-drift
# decoys). Rows MATCH / WARN; exit 0 clean / 1 (--strict, any WARN) / 3 usage.
#
# Grammar (C11 design §D4): declaration ERE ^# Mutation: [A-Za-z][A-Za-z0-9_-]* -- .+$ ;
# id capture sed -E 's/^# Mutation: ([A-Za-z][A-Za-z0-9_-]*) -- .*/\1/'.
#
# RED on dab0807: toolbelt/lint-guard-pins.sh does not exist (command-not-found); and
# even present, 0 of the 9 lint-*.sh declare a # Mutation: line -> the real-kit smoke is
# 9 WARN. GREEN after PR4 adds the tool + retrofits the 10 lint headers.
# [ev: companero/investigador1 C11 T4; design build-n4-module-campaign11 §D4]

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  GP="$REPO/build-n4-module-kit/toolbelt/lint-guard-pins.sh"
}

# Build a throwaway kit-root: toolbelt/<script>.sh (optional # Mutation: <mid>) +
# tests/<script>.bats (optional @test "<def>").
_mklint() {  # dir  script-basename  mutation-id(or "")  defined-test-id(or "")
  local d="$1" s="$2" mid="$3" def="$4"
  mkdir -p "$d/toolbelt" "$d/tests"
  { printf '#!/usr/bin/env bash\n# %s\n' "$s"
    if [ -n "$mid" ]; then printf '# Mutation: %s -- drops the guard\n' "$mid"; fi
  } > "$d/toolbelt/$s.sh"
  { printf '#!/usr/bin/env bats\n'
    if [ -n "$def" ]; then printf '@test "%s: a guard" { true; }\n' "$def"; fi
  } > "$d/tests/$s.bats"
  return 0
}

@test "T4-usage: no kit-root argument -> exit 3" {
  run "$GP"
  [ "$status" -eq 3 ]
}

@test "T4-match: a lint whose # Mutation: id resolves to an @test -> MATCH row, 0 WARN, exit 0" {
  d="$BATS_TEST_TMPDIR/match"; _mklint "$d" lint-foo FOO1 FOO1
  run "$GP" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  [[ "$output" == *"MATCH"* && "$output" == *"mutation FOO1 -> tests/lint-foo.bats:"* ]]
}

@test "T4-nofixture: a # Mutation: id with no matching @test -> WARN naming the id, exit 0" {
  d="$BATS_TEST_TMPDIR/nofix"; _mklint "$d" lint-bar BAR9 ""
  run "$GP" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* && "$output" == *"mutation BAR9 -> no fixture"* ]]
}

@test "T4-nomutation: a lint-*.sh with NO # Mutation: line -> WARN (a lint cannot be vacuously clean)" {
  d="$BATS_TEST_TMPDIR/nomut"; _mklint "$d" lint-baz "" ""
  run "$GP" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* && "$output" == *"lint-baz.sh"* && "$output" == *"no # Mutation: line"* ]]
}

@test "T4-scope: a NON-lint script (report-module.sh) is out of scan -> never flagged for a missing mutation" {
  d="$BATS_TEST_TMPDIR/scope"; _mklint "$d" lint-ok OK1 OK1
  # add a non-lint script with no # Mutation: line
  printf '#!/usr/bin/env bash\n# report-module\n' > "$d/toolbelt/report-module.sh"
  run "$GP" "$d"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  [[ "$output" != *"report-module"* ]]
}

@test "T4-strict: --strict promotes any WARN to exit 1" {
  d="$BATS_TEST_TMPDIR/strict"; _mklint "$d" lint-baz "" ""
  run "$GP" --strict "$d"
  [ "$status" -eq 1 ]
  [[ "$output" == *"WARN"* ]]
}

@test "T4-smoke: after PR4 the real kit has 10 lint-*.sh each with a resolved # Mutation: (>=10 MATCH rows over 10 distinct scripts, 0 WARN, lint-timers S21-misparse -> tests/lint-timers.bats:436). RED on dab0807 (0 lints declare a mutation)." {
  run "$GP" "$REPO"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^MATCH')" -ge 10 ]
  [ "$(printf '%s\n' "$output" | grep '^MATCH' | awk '{print $3}' | cut -d: -f1 | sort -u | grep -c .)" -eq 10 ]
  [[ "$output" == *"mutation S21-misparse -> tests/lint-timers.bats:436"* ]]
}
