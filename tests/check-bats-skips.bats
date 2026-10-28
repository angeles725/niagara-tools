#!/usr/bin/env bats
# check-bats-skips.sh — the global bats SKIP allowlist assertion CI runs over the TAP of the whole suite
# (audit A6): every `# skip <reason>` must match one ERE line of the allowlist, so a suite cannot start
# skipping for a new, unreviewed reason (a lost tool install, a broken fixture) and still read green.

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  CHK="$REPO/scripts/check-bats-skips.sh"
  TAP="$BATS_TEST_TMPDIR/run.tap"
  ALLOW="$BATS_TEST_TMPDIR/allow.txt"
  printf '%s\n' '# comment lines and blank lines are ignored' '' '^tool [a-z]+ not installed$' > "$ALLOW"
}

@test "SK1: every skip reason matches an allowlist line -> exit 0" {
  printf '%s\n' '1..3' 'ok 1 a' 'ok 2 b # skip tool eslint not installed' 'not ok 3 c' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 0 ]
  [[ "$output" == *"1 skip(s), all allowlisted"* ]]
}

@test "SK2: a skip reason no allowlist line matches -> exit 1 naming the test and the reason" {
  # Mutation: SK2-any-skip-ok -- accepting every reason lets a suite skip for a new reason and stay green
  printf '%s\n' '1..2' 'ok 1 a # skip tool eslint not installed' 'ok 2 b-name # skip junit jar missing' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 1 ]
  [[ "$output" == *"UNLISTED skip: ok 2 b-name — junit jar missing"* ]]
}

@test "SK3: an allowlist line matches the WHOLE reason, not a substring" {
  # Mutation: SK3-unanchored -- an unanchored match lets 'tool x not installed (and more)' through
  printf '%s\n' 'tool [a-z]+ not installed' > "$ALLOW"   # no ^ or $: the whole-reason match is the check's job
  printf '%s\n' '1..1' 'ok 1 a # skip tool eslint not installed, and the fixture is gone' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 1 ]
}

@test "SK4: a skip with no reason is unlisted unless a line allows the empty reason" {
  printf '%s\n' '1..1' 'ok 1 a # skip' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 1 ]
  [[ "$output" == *"UNLISTED skip: ok 1 a — (no reason)"* ]]
}

@test "SK5: a TAP with no plan, or an unreadable TAP or allowlist, is exit 3 (never a pass)" {
  # Mutation: SK5-no-plan -- an empty or truncated TAP (bats never ran) would read as 'no skips'
  : > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 3 ]
  run bash "$CHK" "$BATS_TEST_TMPDIR/missing.tap" "$ALLOW"
  [ "$status" -eq 3 ]
  printf '1..0\n' > "$TAP"
  run bash "$CHK" "$TAP" "$BATS_TEST_TMPDIR/missing.txt"
  [ "$status" -eq 3 ]
  run bash "$CHK" "$TAP"
  [ "$status" -eq 3 ]
}

@test "SK6: a bad ERE in the allowlist is exit 3, not a silent non-match" {
  printf '%s\n' '1..1' 'ok 1 a # skip tool x not installed' > "$TAP"
  printf '%s\n' '^tool (x not installed$' > "$ALLOW"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 3 ]
}

@test "SK7: the committed allowlist accepts every skip the CI bats job is known to take" {
  # the reasons CI took on main before the npm tools were installed, minus the eslint/acorn ones the bats job
  # now installs for; anonymized shapes of the client-tree reasons
  printf '%s\n' '1..9' \
    'ok 1 a # skip campaign-9 close gate — run with C9_CLOSE=1 at close' \
    'ok 2 b # skip C11 close gate — set C11_CLOSE=1 to run' \
    'ok 3 c # skip client read tree not on this machine (set C9_CLIENT_ROOT)' \
    'ok 4 d # skip no niagara_home/JDK 8 for the real round-trip (local-only bless evidence, never a fake PASS)' \
    'ok 5 e # skip no Chrome (set KIT_CHROME or PUPPETEER_EXECUTABLE_PATH)' \
    'ok 6 f # skip SiteA bog not on this machine (set C8_SITEA_BOG)' \
    'ok 7 g # skip the pinned eslint is installed under toolbelt/eslint; the PATH fallback is not reached' \
    'ok 8 h # skip launcher not installed on this machine' \
    'ok 9 i # skip ModA-ux rc not on this machine (local-only real smoke)' > "$TAP"
  run bash "$CHK" "$TAP" "$REPO/tests/bats-skip-allowlist.txt"
  [ "$status" -eq 0 ] || { printf '%s\n' "$output" >&2; return 1; }
  printf '%s\n' '1..2' 'ok 1 a # skip acorn not installed' \
    'ok 2 b # skip eslint not installed (npm install --prefix build-n4-module-kit/toolbelt/eslint)' > "$TAP"
  run bash "$CHK" "$TAP" "$REPO/tests/bats-skip-allowlist.txt"
  [ "$status" -eq 1 ]
}

@test "SK8: a skip whose reason quotes a later ' # skipped' is still read (the last VALID marker splits)" {
  # Mutation: SK8-last-marker -- taking the last ' # skip' even when a letter follows drops the whole skip line
  printf '%s\n' '1..1' 'ok 1 a # skip junit jar missing # skipped twice' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 1 ]
  [[ "$output" == *"UNLISTED skip: ok 1 a — junit jar missing # skipped twice"* ]]
}

@test "SK9: every allowlist line is its own pattern (an alternation in one line never spans two)" {
  # two lines, each anchored on its own: a reason made of the end of one and the start of the other is unlisted
  printf '%s\n' '^alpha one$' '^beta two$' > "$ALLOW"
  printf '%s\n' '1..3' 'ok 1 a # skip alpha one' 'ok 2 b # skip beta two' 'ok 3 c # skip one beta' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 1 ]
  [[ "$output" == *"UNLISTED skip: ok 3 c — one beta"* ]]
  [[ "$output" == *"1 of 3 skip(s)"* ]]
}

@test "SK10: an allowlist whose last line has no trailing newline still allows that line" {
  # Mutation: SK10-last-line -- a plain `while read` drops a final line with no newline, so its skip reads UNLISTED
  printf '%s' '^junit jar missing$' > "$ALLOW"
  printf '%s\n' '1..1' 'ok 1 a # skip junit jar missing' > "$TAP"
  run bash "$CHK" "$TAP" "$ALLOW"
  [ "$status" -eq 0 ]
  [[ "$output" == *"1 skip(s), all allowlisted"* ]]
}
