#!/usr/bin/env bats
# lint-bats-errexit.bats — a mid-test && list of assertions or a bare `! cmd` never fails its bats test (errexit
# ignores every command of an && list but the last, and a negated command); lint-bats-errexit FAILs both. The
# rule's cases live in go/internal/lint/rules/batserrexit. [ev: retro r10-bats-list-hazard Δ1]
# shellcheck disable=SC2016  # the fixture lines and the row messages are literal bats source, never expanded here

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-bats-errexit.sh"
  D="$BATS_TEST_TMPDIR/suite"
  mkdir -p "$D"
}

# _fixture <name> <body line>...: writes $D/<name>.bats, one @test whose body is the given lines then `true`.
_fixture() {
  local name="$1"
  shift
  { printf '@test "%s" {\n' "$name"; printf '  %s\n' "$@"; printf '  true\n}\n'; } > "$D/$name.bats"
}

AND_MSG='mid-test && list of assertions: errexit checks only its last command, so an earlier false assertion never fails the test -- put one assertion per line'
NEG_MSG='mid-test ! negation: errexit never fires on a negated command, so it never fails the test -- use `run ! cmd` or `! cmd || return 1`'

@test "BEX-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-bats-errexit.sh [--strict] <bats-dir>..." ]
}

@test "BEX0: the hazard is real: bats passes a test whose first && assertion and whose ! assertion are false" {
  _fixture split-me '[ 1 -eq 2 ] && [ 1 -eq 1 ]' '! true'
  run bats "$D/split-me.bats"
  [ "$status" -eq 0 ]
  [[ "$output" == *"ok 1 split-me"* ]]
}

@test "BEX1: a mid-test && list of assertions FAILs on its line" {
  # Mutation: BEX1 -- accepting && lists turns this case green (exit 0, no row)
  _fixture split-me 'run printf x' '[ "$status" -eq 1 ] && [[ "$output" == *y* ]]'
  run "$L" "$D"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  bats-and-list  $D/split-me.bats:3  $AND_MSG" ]
}

@test "BEX2: a bare mid-test negation FAILs on its line" {
  # Mutation: BEX2 -- accepting a bare negation turns this case green (exit 0, no row)
  _fixture negate-me 'touch f' '! grep -q x f'
  run "$L" "$D"
  [ "$status" -eq 1 ]
  [ "$output" = "FAIL  bats-negation  $D/negate-me.bats:3  $NEG_MSG" ]
}

@test "BEX3: the split form, || tails, guards, conditions, strings, heredocs and the last line are clean" {
  # Mutation: BEX3 -- flagging the last command or an || tail makes this case FAIL
  _fixture clean-forms '[ "$status" -eq 0 ]' '[[ "$output" == *x* ]]' '! grep -q x f || return 1' 'run ! false' \
    '[ -f a ] || fail "no a"' '[ -n "$X" ] && [ -f "$X" ] || skip "no X"' '[ -f a ] || true' \
    '[ -n "$p" ] && ln -s "$p" d' 'if [ a = a ] && [ b = b ]; then :; fi' '[[ a == a && b == b ]]' \
    "run bash -c '[ a = b ] && [ c = d ]'"
  printf '@test "heredoc" {\n  cat > g <<'"'"'EOF'"'"'\n[ a = b ] && [ c = d ]\n! true\nEOF\n  true\n}\n' > "$D/heredoc.bats"
  printf '@test "last" {\n  true\n  [ a = a ] && [ b = b ]\n}\n' > "$D/last.bats"
  run "$L" --strict "$D"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "BEX-K20: a root with no *.bats is the ERROR row and exit 3; fixtures/ is not read" {
  mkdir -p "$D/fixtures"
  printf '@test "f" {\n  ! true\n  true\n}\n' > "$D/fixtures/f.bats"
  run "$L" "$D"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-bats-errexit  $D  no *.bats under the root (wrong path: pass a bats test directory)" ]
}

@test "BEX-kit: the kit's own tests/*.bats hold no assertion errexit cannot see" {
  run "$L" "$BATS_TEST_DIRNAME"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
