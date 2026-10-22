#!/usr/bin/env bats
# script-dir.bats — a toolbelt script resolves its own directory, whatever the invocation (issue #310).
# `${BASH_SOURCE[0]%/*}` returns the script NAME when BASH_SOURCE carries no slash (`bash lint-x.sh` run from
# the toolbelt dir), so `cd` fails, the lib path becomes `/lib/<lib>.sh`, and the lint dies or, inside `$(...)`,
# silently skips its check. The fix is the dirname form: `$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)`.
# SD1/SD2 pin the behaviour on a real lint; SD3 is the ratchet that keeps the idiom out of the toolbelt.
# A PATH lookup (`lint-x.sh` or `bash lint-x.sh` from another dir) always hands bash a path with a slash, so
# only the from-own-dir invocation reproduces the defect.

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  TOOLBELT="$ROOT/build-n4-module-kit/toolbelt"
  FIX="$ROOT/tests/fixtures/script-dir/src"
}

# assert_timer_finding: lint-timers FAILs the fixture's Owner class (exit 1, finding named), no lib error.
assert_timer_finding() {
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"*"Owner"* ]]
  if [[ "$output" == *"No such file"* || "$output" == *"Not a directory"* ]]; then return 1; fi
}

@test "SD1: bash <name> from the toolbelt dir (BASH_SOURCE has no slash) still sources its libs and FAILs" {
  # Mutation: SD1 -- restoring \${BASH_SOURCE[0]%/*} in lint-timers.sh makes cd fail and the lib path /lib/...
  cd "$TOOLBELT"
  run bash lint-timers.sh "$FIX"
  assert_timer_finding
}

@test "SD2: a bare name on PATH, run from an unrelated dir, still FAILs the fixture" {
  cd "$BATS_TEST_TMPDIR"
  PATH="$TOOLBELT:$PATH" run lint-timers.sh "$FIX"
  assert_timer_finding
}

# strip_idiom_rows <file>...: prints `<file>:<line>:<text>` for every `${BASH_SOURCE%…}`, `${BASH_SOURCE[0]%…}`
# or `${0%…}` suffix strip; a script dir must come from dirname, which returns `.` for a slash-less name.
# A comment line that names the idiom (lint-structure.sh explains why it avoids it) is not a use; exit 1 = none.
strip_idiom_rows() { grep -nHE '\$\{(BASH_SOURCE(\[0\])?|0)%' "$@" | grep -vE '^[^:]*:[0-9]+:[[:space:]]*#'; }

@test "SD3: no toolbelt script or lib derives its directory by a suffix strip of BASH_SOURCE or \$0" {
  # Mutation: SD3 -- reintroducing one \${BASH_SOURCE[0]%/*} in any toolbelt/*.sh or lib/*.sh names it here
  run strip_idiom_rows "$TOOLBELT"/*.sh "$TOOLBELT"/lib/*.sh
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "SD3-neg: the scan names each suffix-strip variant and passes the dirname form and a comment" {
  cat > "$BATS_TEST_TMPDIR/bad.sh" <<'SH'
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$(cd "${BASH_SOURCE[0]%/*}" && pwd)/lib/a.sh"
. "${BASH_SOURCE%/*}/b.sh"
d=${0%/*}
  # comment only: ${BASH_SOURCE[0]%/*} is not used here
SH
  run strip_idiom_rows "$BATS_TEST_TMPDIR/bad.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"bad.sh:2:"* && "$output" == *"bad.sh:3:"* && "$output" == *"bad.sh:4:"* ]]
  if [[ "$output" == *"bad.sh:1:"* || "$output" == *"bad.sh:5:"* ]]; then return 1; fi
}
