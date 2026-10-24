#!/usr/bin/env bats
# go-wrapper-symlink.bats — a Go-backed toolbelt wrapper finds <repo>/go from its REAL path, so it still runs
# when invoked through a symlinked toolbelt copy. report-module's tests build that copy with
# helpers/stub-toolbelt.bash, which symlinks every toolbelt FILE individually into a temp dir; the old
# `here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"` then yields the temp dir, `$here/../../go` does not
# exist, and the wrapper exits 3 "go-tool.sh missing" (RM40, RM41, RM-floor-schema, RM-wp-nomatrix broke
# that way when a lint became a Go wrapper). `pwd -P` alone does not help: the directory is real, the FILE
# is the symlink. The fix resolves the file first: `$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)`.
# GWS1 pins the behaviour on a real wrapper, GWS2 sweeps every Go wrapper, GWS3 is the static ratchet.
# GWS4 proves a wrapper swap left no code below the `exec` tail: ports land the tail first and delete the
# dead bash body in later commits, which a reviewer cannot locate findings on (removed lines only).

load helpers/stub-toolbelt

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$ROOT/build-n4-module-kit"
  COPY="$BATS_TEST_TMPDIR/toolbelt-copy"
  # The same copy report-module's tests run: every entry symlinked, one unrelated (bash, non-Go) member stubbed.
  # The stubbed member is a dummy the test creates, never a real script: a real bash member gets ported to Go
  # (rc-scan.sh did), and its stub then reads as a Go wrapper that lost its symlink (GWS2). stub_toolbelt
  # replaces an existing entry, so the dummy is created first.
  mkdir -p "$COPY"
  : > "$COPY/zz-plain-bash-member.sh"
  stub_toolbelt "$COPY" zz-plain-bash-member.sh 'exit 0'
}

# go_wrappers: every toolbelt script that sources go/go-tool.sh, one basename per line.
go_wrappers() { grep -l 'go-tool\.sh"' "$KIT"/toolbelt/*.sh | xargs -n1 basename; }

@test "GWS1: check-facets-card.sh run through a symlinked toolbelt copy from an unrelated cwd matches the card" {
  # Mutation: GWS1 -- restoring the unresolved here= in check-facets-card.sh exits 3 "go-tool.sh missing"
  [ -L "$COPY/check-facets-card.sh" ]
  cd "$BATS_TEST_TMPDIR"
  run "$COPY/check-facets-card.sh" "$KIT/types/logic-authoring.md" "$ROOT/tests/fixtures/facets-card/BFacets-4.15.6.20.keys"
  [ "$status" -eq 0 ]
  [[ "$output" == *"34 keys match"* ]]
}

@test "GWS2: every Go wrapper invoked through the symlinked copy reaches its Go module" {
  # Mutation: GWS2 -- restoring the unresolved here= in any Go wrapper names it here
  local w bad=""
  cd "$BATS_TEST_TMPDIR"
  for w in $(go_wrappers); do
    [ -L "$COPY/$w" ] || { bad+=" $w(not-a-symlink)"; continue; }
    # `bash <path>`: gen-lint-index.sh is tracked 100644, so a direct exec would fail before any lookup.
    # `timeout`: a wrapper that hangs or re-execs itself is named instead of stalling the suite.
    run timeout 60 bash "$COPY/$w" --definitely-not-an-option
    if [ "$status" -eq 124 ]; then bad+=" $w(timeout)"; continue; fi
    if [[ "$output" == *"go-tool.sh missing"* || "$output" == *"no Go module"* ]]; then bad+=" $w"; fi
  done
  [ -z "$bad" ] || { printf 'wrappers that lost go/ through the symlink:%s\n' "$bad"; return 1; }
}

# unresolved_here_rows <file>...: prints `<file>:<line>:<text>` for every `here=` assignment that does not
# resolve BASH_SOURCE through `readlink -f`; exit 1 = none.
# shellcheck disable=SC2016  # the pattern is the literal idiom text, not an expansion
unresolved_here_rows() { grep -nHE '^[[:space:]]*here=' "$@" | grep -vF 'readlink -f "${BASH_SOURCE[0]}"'; }

@test "GWS3: every toolbelt script sourcing go-tool.sh derives here= from its resolved real path" {
  # Mutation: GWS3 -- reverting one Go wrapper's here= to the unresolved dirname form names it here
  local files=() w
  for w in $(go_wrappers); do files+=("$KIT/toolbelt/$w"); done
  [ "${#files[@]}" -ge 6 ]
  run unresolved_here_rows "${files[@]}"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "GWS3-neg: the scan names an unresolved here= and passes the resolved form" {
  cat > "$BATS_TEST_TMPDIR/w.sh" <<'SH'
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
here="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
SH
  run unresolved_here_rows "$BATS_TEST_TMPDIR/w.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"w.sh:1:"* ]]
  if [[ "$output" == *"w.sh:2:"* ]]; then return 1; fi
}

# code_after_exec_rows <file>...: prints `<file>:<line>:<text>` for every non-blank, non-comment line below a
# file's first top-level `exec`, and for any `disable=SC2093` left over from a staged wrapper swap; exit 1 = none.
code_after_exec_rows() {
  awk 'FNR == 1 { tail = 0 }
       /disable=SC2093/ { print FILENAME ":" FNR ":" $0; found = 1 }
       tail && $0 !~ /^[[:space:]]*(#.*)?$/ { print FILENAME ":" FNR ":" $0; found = 1 }
       /^exec[[:space:]]/ { tail = 1 }
       END { exit found ? 0 : 1 }' "$@"
}

@test "GWS4: every Go wrapper ends at its exec line (no dead bash body below it)" {
  # Mutation: GWS4 -- leaving one line of the old bash body below a wrapper's exec names it here
  local files=() w
  for w in $(go_wrappers); do files+=("$KIT/toolbelt/$w"); done
  [ "${#files[@]}" -ge 6 ]
  run code_after_exec_rows "${files[@]}"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "GWS4-neg: the scan names code and a leftover SC2093 disable below exec, and passes comments and blanks" {
  # shellcheck disable=SC2016  # the fixture lines are literal wrapper text, not expansions
  printf '%s\n' 'set -u' '# shellcheck disable=SC2093' 'exec "$BIN" x "$@"' '' '# trailing note' 'old_body() { :; }' > "$BATS_TEST_TMPDIR/w.sh"
  # shellcheck disable=SC2016
  printf '%s\n' 'set -u' 'exec "$BIN" x "$@"' '' '# trailing note' > "$BATS_TEST_TMPDIR/ok.sh"
  run code_after_exec_rows "$BATS_TEST_TMPDIR/w.sh" "$BATS_TEST_TMPDIR/ok.sh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"w.sh:2:"* ]]
  [[ "$output" == *"w.sh:6:"* ]]
  if [[ "$output" == *"ok.sh"* ]]; then return 1; fi
}
