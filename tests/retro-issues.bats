#!/usr/bin/env bats
# retro-issues.bats — the launcher of retro-issues (retro deltas are GitHub issues, CONTRIBUTING.md § 6.3). The unit
# cases (scrub, dedup, grouping, --new, write-back) live in go/internal/retroissues with an injected gh; here a fake gh
# on PATH proves the launcher builds the binary, hands it the kit root, and that a dry run never writes.

setup() {
  T="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit/toolbelt/retro-issues.sh"
  FAKE="$BATS_TEST_TMPDIR/bin"
  mkdir -p "$FAKE" "$BATS_TEST_TMPDIR/retros"
  # shellcheck disable=SC2016  # the fake gh is a script body: its $ expands when IT runs
  printf '%s\n' '#!/usr/bin/env bash' 'printf "%s\n" "$*" >> "$GH_LOG"' 'case "$1 $2" in "issue list") echo "[]";; "label list") echo "[]";; esac' > "$FAKE/gh"
  chmod +x "$FAKE/gh"
  export GH_LOG="$BATS_TEST_TMPDIR/gh.log"
  : > "$GH_LOG"
  printf '%s\n' '\bsecretword\b' > "$BATS_TEST_TMPDIR/denylist.txt"
  export NIAGARA_CLIENT_DENYLIST="$BATS_TEST_TMPDIR/denylist.txt"
  # shellcheck disable=SC2016  # backticks are retro markdown, not a command substitution
  printf '%s\n' '## Proposed kit deltas' '| Δ | Delta | Target | Token |' '|---|---|---|---|' \
    '| Δ1 | **n4_foo drops a key.** Longer text | `go/internal/mcpn4/x.go` | t |' > "$BATS_TEST_TMPDIR/2026-01-02-sample.md"
}

@test "RI1: no retro is a usage error, exit 3" {
  run "$T"
  [ "$status" -eq 3 ]
  [[ "$output" == *"usage: retro-issues.sh"* ]]
}

@test "RI2: a dry run plans the issue and runs no gh command that writes" {
  PATH="$FAKE:$PATH" run "$T" --alias sample=ALPHA "$BATS_TEST_TMPDIR/2026-01-02-sample.md"
  [ "$status" -eq 0 ]
  [[ "$output" == *"create: mcp-n4: n4_foo drops a key"* ]]
  [[ "$output" == *"retro ALPHA Δ1"* ]]
  [[ "$output" != *"sample"* ]]
  run grep -q 'issue create\|issue comment\|label create' "$GH_LOG"
  [ "$status" -eq 1 ]
}

@test "RI3: --apply without a denylist refuses before any gh call" {
  PATH="$FAKE:$PATH" NIAGARA_CLIENT_DENYLIST="$BATS_TEST_TMPDIR/none.txt" run "$T" --apply --alias sample=ALPHA "$BATS_TEST_TMPDIR/2026-01-02-sample.md"
  [ "$status" -eq 3 ]
  [[ "$output" == *"denylist"* ]]
  [ ! -s "$GH_LOG" ]
}
