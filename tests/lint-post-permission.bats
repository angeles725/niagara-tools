#!/usr/bin/env bats
# lint-post-permission.bats — a request, RPC or action handler that calls post( with no permission check before it FAILs
# (types/security.md § 2.4: the queued action runs with no RBAC Context). The rule's golden cases live in
# go/internal/lint/rules/postpermission; this suite runs the toolbelt wrapper over the Java fixtures.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-post-permission.sh"
  FIX="$BATS_TEST_DIRNAME/fixtures/lint-post-permission"
  ROW="runs the queued action with no RBAC Context and no permission check comes before it in this method"
}

@test "PP-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-post-permission.sh <src-dir> [--strict]" ]
}

@test "PP-help: --help prints the header (the rule, the accepted checks, the waiver) and exits 0" {
  run "$L" --help
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "lint-post-permission.sh — FAILs a request, RPC or action handler that calls post( with no permission check before it." ]
  [[ "$output" == *"post-permission: ok <reason>"* ]]
  [[ "$output" == *"checkInvoke("* ]]
}

@test "PP-K20: a root with no *.java is the ERROR row, exit 3" {
  mkdir -p "$BATS_TEST_TMPDIR/empty"
  run "$L" "$BATS_TEST_TMPDIR/empty"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-post-permission  $BATS_TEST_TMPDIR/empty  no files to scan (K20: wrong path or empty scaffold)" ]
}

@test "PP-pass: a handler that checks the caller before post( is clean" {
  run "$L" "$FIX/pass"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PP-fail: an unchecked post( in a handler FAILs at the post line, exit 1, also under --strict" {
  run "$L" "$FIX/fail"
  [ "$status" -eq 1 ]
  [[ "${lines[0]}" == "FAIL  lint-post-permission  Panel.java:5  post() in 'doPost' $ROW"* ]]
  run "$L" "$FIX/fail" --strict
  [ "$status" -eq 1 ]
  [[ "${lines[0]}" == "FAIL  lint-post-permission  Panel.java:5  post() in 'doPost' $ROW"* ]]
}

@test "PP1: a check after the post( guards nothing" {
  # Mutation: PP1 -- accepting a permission check anywhere in the method, not only before the post(, passes a post-then-check handler.
  run "$L" "$FIX/fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Panel.java:9  post() in 'doPut' "* ]]
}

@test "PP2: an internal timer post (no Context, no request) is not flagged" {
  # Mutation: PP2 -- judging every method, not only handlers, FAILs an internal timer post (started(), doTick(), a private helper).
  run "$L" "$FIX/internal"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "PP3: a waiver with a reason accepts the post(; a bare waiver does not" {
  # Mutation: PP3 -- accepting a waiver with no reason waives a bare `// post-permission: ok`.
  run "$L" "$FIX/waiver"
  [ "$status" -eq 1 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "$output" == "FAIL  lint-post-permission  Panel.java:9  post() in 'doPut' "* ]]
}

@test "PP4: a guard named in a comment or a string is not a guard" {
  # Mutation: PP4 -- reading a guard name in a comment or a string literal as a guard passes an unchecked post(.
  run "$L" "$FIX/fail"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Panel.java:16  post() in 'doGet' "* ]]
}

@test "PP-rows: the fail fixture has exactly the three rows" {
  run "$L" "$FIX/fail"
  [ "${#lines[@]}" -eq 3 ]
}
