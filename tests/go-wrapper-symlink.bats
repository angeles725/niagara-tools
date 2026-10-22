#!/usr/bin/env bats
# go-wrapper-symlink.bats — a Go-backed toolbelt wrapper finds <repo>/go from its REAL path, so it still runs
# when invoked through a symlinked toolbelt copy. report-module's tests build that copy with
# helpers/stub-toolbelt.bash, which symlinks every toolbelt FILE individually into a temp dir; the old
# `here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"` then yields the temp dir, `$here/../../go` does not
# exist, and the wrapper exits 3 "go-tool.sh missing" (RM40, RM41, RM-floor-schema, RM-wp-nomatrix broke
# that way when a lint became a Go wrapper). `pwd -P` alone does not help: the directory is real, the FILE
# is the symlink. The fix resolves the file first: `$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)`.
# GWS1 pins the behaviour on a real wrapper, GWS2 sweeps every Go wrapper, GWS3 is the static ratchet.

load helpers/stub-toolbelt

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$ROOT/build-n4-module-kit"
  COPY="$BATS_TEST_TMPDIR/toolbelt-copy"
  # The same copy report-module's tests run: every entry symlinked, one unrelated member stubbed.
  stub_toolbelt "$COPY" lint-timers.sh 'exit 0'
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
