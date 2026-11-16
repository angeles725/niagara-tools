#!/usr/bin/env bats
# ci-local.bats — scripts/ci-local.sh replays .github/workflows/ci.yml locally (quality round 8 T8g).
# Why: local shellcheck drifts newer than the CI pin (0.11 vs 0.10.0: SC2015 missed, SC2329 extra),
# and GitHub `run:` steps use `bash -e {0}` (errexit, no pipefail). The replay derives its steps from
# ci.yml, so a CI step it neither runs nor skips on purpose is a drift failure, never a silent gap.
# [ev: retro 2026-10-22-ci-local-replay]

setup() {
  ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  CL="$ROOT/scripts/ci-local.sh"
  CI="$ROOT/.github/workflows/ci.yml"
  FX="$ROOT/tests/fixtures/ci-local"
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "CL1: --list classifies every ci.yml step as RUN or SKIP (no step uncovered, no stale skip rule)" {
  # Mutation: CL1 -- a parser that drops `run: |` blocks lists fewer steps than ci.yml has.
  run "$CL" --list --ci-file "$CI"
  [ "$status" -eq 0 ]
  [[ "$output" != *UNCOVERED* ]]
  [[ "$output" != *STALE* ]]
  local items listed name
  items="$(grep -cE '^[[:space:]]+- (name|uses|run):' "$CI")"
  listed="$(grep -cE '^(RUN|SKIP) ' <<< "$output")"
  [ "$items" -eq "$listed" ]
  while IFS= read -r name; do
    grep -qxE "(RUN|SKIP) $(printf '%s' "$name" | sed 's/[][\.*^$+?(){}|]/\\&/g')( — .*)?" <<< "$output" \
      || { echo "not listed: $name"; return 1; }
  done < <(sed -nE 's/^[[:space:]]+- name: //p' "$CI")
}

@test "CL2: a run: step that needs a system install and has no skip rule is UNCOVERED (exit 2)" {
  # Mutation: CL2 -- running every unmatched step treats `sudo apt-get` as an ordinary local step.
  run "$CL" --list --ci-file "$FX/uncovered-sudo.yml"
  [ "$status" -eq 2 ]
  [[ "$output" == *"UNCOVERED Install foo"* ]]
}

@test "CL3: an unknown uses: action and a step-level env/shell key are UNCOVERED (exit 2)" {
  run "$CL" --list --ci-file "$FX/uncovered-uses.yml"
  [ "$status" -eq 2 ]
  [[ "$output" == *"UNCOVERED some/action@v1"* ]]
  [[ "$output" == *"UNCOVERED with env"*"env"* ]]
  [[ "$output" == *"UNCOVERED with shell"*"shell"* ]]
}

@test "CL4: a run: form the parser does not read (folded >) fails with its line, never vanishes (exit 2)" {
  # Mutation: CL4 -- skipping unknown block styles drops the step from --list and from the run.
  run "$CL" --list --ci-file "$FX/folded.yml"
  [ "$status" -eq 2 ]
  [[ "$output" == *"folded.yml:9:"* ]]
}

@test "CL5: a cached shellcheck tarball with the wrong sha256 is rejected (exit 3) before any step runs" {
  # Mutation: CL5 -- checking the hash only after a fresh download trusts a tampered cache.
  mkdir -p "$TMPDIR_T/cache"
  printf 'not a tarball\n' > "$TMPDIR_T/cache/shellcheck-v0.10.0.linux.x86_64.tar.xz"
  run "$CL" --ci-file "$FX/shellcheck.yml" --cache-dir "$TMPDIR_T/cache"
  [ "$status" -eq 3 ]
  [[ "$output" == *"sha256 mismatch"* ]]
  [[ "$output" != *"== marker"* ]]
  [[ "$output" != *"marker-ran"* ]]
}

@test "CL6: steps run under bash -e without pipefail, and a failure does not stop later steps (exit 1)" {
  # Mutation: CL6 -- `bash -eo pipefail` fails the `false | true` step that passes on GitHub.
  run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache"
  [ "$status" -eq 1 ]
  [[ "$output" == *"PASS pipe without pipefail"* ]]
  [[ "$output" == *"FAIL plain false"* ]]
  [[ "$output" == *"FAIL errexit stops the block"* ]]
  [[ "$output" != *"should-not-print"* ]]
  [[ "$output" == *"PASS runs after a failure"* ]]
  [[ "$output" == *"ran-after"* ]]
  [[ "$output" == *"PASS multi-line block"*  ]]
  [[ "$output" == *"line-two-ok"* ]]
  [ ! -e "$TMPDIR_T/cache/shellcheck-v0.10.0.linux.x86_64.tar.xz" ]
}

@test "CL7: --only runs just the steps whose name contains the substring" {
  run "$CL" --ci-file "$FX/semantics.yml" --cache-dir "$TMPDIR_T/cache" --only 'runs after'
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS runs after a failure"* ]]
  [[ "$output" != *"plain false"* ]]
}

@test "CL8: -j N turns 'bats tests/*.bats' into 'bats -j N tests/*.bats' when parallel is on PATH" {
  mkdir -p "$TMPDIR_T/bin"
  printf '#!/bin/sh\necho "bats-args: $*"\n' > "$TMPDIR_T/bin/bats"
  printf '#!/bin/sh\nexit 0\n' > "$TMPDIR_T/bin/parallel"
  chmod +x "$TMPDIR_T/bin/bats" "$TMPDIR_T/bin/parallel"
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$FX/bats.yml" --cache-dir "$TMPDIR_T/cache" -j 2
  [ "$status" -eq 0 ]
  [[ "$output" == *"bats-args: -j 2 tests/"* ]]
}

@test "CL9: an unknown option is a usage error (exit 2)" {
  run "$CL" --bogus
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage"* ]]
}

# CI runs `bats -j 4 --no-parallelize-within-files` (four files at a time); the replay must run it with the
# local -j N, and as plain bats where GNU parallel is missing (bats -j without parallel runs no test at all,
# and bats refuses --no-parallelize-within-files without -j).
# [ev: retro r10-ci-wall-time]
@test "CL16: -j N turns CI's 'bats -j 4 --no-parallelize-within-files ' into -j N, and plain bats without parallel" {
  # Mutation: CL16 -- rewriting only the literal 'bats tests/*.bats' runs CI's -j 4 locally, or -j with no parallel.
  mkdir -p "$TMPDIR_T/bin"
  printf '#!/bin/sh\necho "bats-args: $*"\n' > "$TMPDIR_T/bin/bats"
  chmod +x "$TMPDIR_T/bin/bats"
  printf 'name: CI\non: pull_request\njobs:\n  bats:\n    runs-on: ubuntu-latest\n    steps:\n      - name: bats j4\n        run: bats -j 4 --no-parallelize-within-files --formatter tap tests/*.bats\n' > "$TMPDIR_T/j4.yml"
  # A PATH without GNU parallel that keeps every other tool: a shadow dir linking each executable on PATH
  # (first hit wins, as PATH lookup does) except parallel. Dropping whole PATH dirs instead loses the tools
  # that share a dir with parallel (CI installs it in /usr/local/bin beside bats and others).
  local d f shadow="$TMPDIR_T/no-parallel"
  mkdir -p "$shadow"
  while IFS= read -r -d: d; do
    [ -d "$d" ] || continue
    for f in "$d"/*; do
      [ -x "$f" ] && [ ! -d "$f" ] && [ "${f##*/}" != parallel ] && [ ! -e "$shadow/${f##*/}" ] \
        && ln -s "$f" "$shadow/${f##*/}"
    done
  done <<< "$PATH:"
  PATH="$TMPDIR_T/bin:$shadow" run "$CL" --ci-file "$TMPDIR_T/j4.yml" --cache-dir "$TMPDIR_T/cache" -j 2
  [ "$status" -eq 0 ]
  [[ "$output" == *"bats-args: --formatter tap tests/"* ]]
  printf '#!/bin/sh\nexit 0\n' > "$TMPDIR_T/bin/parallel"
  chmod +x "$TMPDIR_T/bin/parallel"
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$TMPDIR_T/j4.yml" --cache-dir "$TMPDIR_T/cache" -j 2
  [ "$status" -eq 0 ]
  [[ "$output" == *"bats-args: -j 2 --no-parallelize-within-files --formatter tap tests/"* ]]
}

# fake_bats <tap lines...> — a bats that prints the TAP lines and exits 0, as a silent parallel run would. It
# answers --version with ci.yml's pin, so the replay runs it instead of fetching the pinned bats.
fake_bats() {
  mkdir -p "$TMPDIR_T/bin"
  local pin; pin="$(sed -nE 's/^[[:space:]]*BATS_VER=([0-9.]+)$/\1/p' "$CI")"
  # shellcheck disable=SC2016  # the stub's own $1, expanded when it runs
  { printf '#!/bin/sh\n[ "$1" != --version ] || { echo "Bats %s"; exit 0; }\ncat <<"TAP"\n' "$pin"
    printf '%s\n' "$@"; printf 'TAP\n'; } > "$TMPDIR_T/bin/bats"
  printf '#!/bin/sh\nexit 0\n' > "$TMPDIR_T/bin/parallel"
  chmod +x "$TMPDIR_T/bin/bats" "$TMPDIR_T/bin/parallel"
}

@test "CL17: ci.yml's bats step fails when fewer tests report than the TAP plan holds, and passes on a full run" {
  # Mutation: CL17 -- trusting bats' exit status passes a parallel run that executed no test (no GNU parallel).
  fake_bats '1..3' 'ok 1 a' 'ok 2 b'
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$CI" --cache-dir "$TMPDIR_T/cache" --only 'bats (all unit tests)'
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL bats (all unit tests)"* ]]
  fake_bats '1..0'
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$CI" --cache-dir "$TMPDIR_T/cache" --only 'bats (all unit tests)'
  [ "$status" -eq 1 ]
  fake_bats 'no plan, no results'
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$CI" --cache-dir "$TMPDIR_T/cache" --only 'bats (all unit tests)'
  [ "$status" -eq 1 ]
  # the skip reason is one tests/bats-skip-allowlist.txt allows (CL18 pins the global SKIP allowlist)
  fake_bats '1..3' 'ok 1 a' 'not ok 2 b' '# out: ok 9 not a result' 'ok 3 c # skip launcher not installed on this machine'
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$CI" --cache-dir "$TMPDIR_T/cache" --only 'bats (all unit tests)'
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS bats (all unit tests)"* ]]
}

@test "CL18: ci.yml's bats step fails on a skip whose reason the global allowlist does not hold (audit A6)" {
  # Mutation: CL18 -- dropping the check-bats-skips.sh call lets a suite skip for a new reason and stay green.
  fake_bats '1..2' 'ok 1 a' 'ok 2 b # skip a reason nobody reviewed'
  PATH="$TMPDIR_T/bin:$PATH" run "$CL" --ci-file "$CI" --cache-dir "$TMPDIR_T/cache" --only 'bats (all unit tests)'
  [ "$status" -eq 1 ]
  [[ "$output" == *"UNLISTED skip: ok 2 b — a reason nobody reviewed"* ]]
  [[ "$output" == *"FAIL bats (all unit tests)"* ]]
}

@test "CL19: CI installs the npm tools with npm ci from the committed lockfile, which pins package.json's exact versions" {
  # Mutation: CL19 -- `npm install` in CI (or an ignored lockfile) resolves the transitive tree fresh on every run.
  local dir="$ROOT/build-n4-module-kit/toolbelt/eslint"
  run git -C "$ROOT" ls-files --error-unmatch build-n4-module-kit/toolbelt/eslint/package-lock.json
  [ "$status" -eq 0 ]
  # lifecycle scripts stay disabled: a dependency's install script never runs in CI
  grep -qE '^[[:space:]]+npm ci --prefix build-n4-module-kit/toolbelt/eslint --ignore-scripts( |$)' "$CI"
  run grep -qE '^[[:space:]]+npm install --prefix build-n4-module-kit/toolbelt/eslint' "$CI"
  [ "$status" -ne 0 ]
  grep -qE 'cache-dependency-path: build-n4-module-kit/toolbelt/eslint/package-lock.json' "$CI"
  run grep -qx 'package-lock.json' "$dir/.gitignore"
  [ "$status" -ne 0 ]
  # the lockfile's root and resolved versions match the exact pins of package.json. CI's bats job sets up node, so
  # this skip is deliberately NOT in tests/bats-skip-allowlist.txt: a CI run that reaches it fails the skip gate.
  command -v node >/dev/null || skip "node not installed"
  run node -e '
    const d = process.argv[1], pkg = require(d + "/package.json"), lock = require(d + "/package-lock.json");
    let bad = 0;
    for (const [n, v] of Object.entries(pkg.devDependencies)) {
      if (lock.packages[""].devDependencies[n] !== v) { console.log("root " + n); bad = 1; }
      if ((lock.packages["node_modules/" + n] || {}).version !== v) { console.log("resolved " + n); bad = 1; }
    }
    process.exit(bad);' "$dir"
  [ "$status" -eq 0 ]
}
