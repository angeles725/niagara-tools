#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# ci-local-kitlinks.bats — `scripts/ci-local.sh --changed` runs tests/kit-links.bats when a kit doc, a ledger.d or a
# changelog.d fragment changes (CL30-CL32). #706 failed CI on kit-links L1 ("dangling reference: prompt.md"): a
# ledger.d fragment named a bare prompt.md, the fragment reaches the generated BUILD-STATE.md, and L1 scans it.
# The pre-push replay did not run kit-links, so the writer saw it first in CI. (retro 2026-11-15 coordination-stretch T25)
# The fixture is a throwaway repository holding this checkout's HEAD (go/ is a link) with this work tree's ci-local
# files on top; its workflow shellchecks scripts/*.sh only and no case changes a .sh, so nothing is downloaded.

setup_file() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  # shellcheck disable=SC1091  # go/ helper, resolved at runtime
  . "$REPO/go/go-tool.sh"
  local t; for t in ci-steps host-budget kit-state; do go_tool_bin "$REPO/go" "$t" >/dev/null; done
}

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  R="$BATS_TEST_TMPDIR/r"
  mkdir -p "$R" "$BATS_TEST_TMPDIR/cg/init.scope" "$BATS_TEST_TMPDIR/gocache"
  git -C "$REPO" archive HEAD -- ':(exclude)go' | tar -x -C "$R"
  ln -s "$REPO/go" "$R/go"
  rm -rf "$R/scripts/lib"
  cp "$REPO/scripts/ci-local.sh" "$R/scripts/"
  cp -R "$REPO/scripts/lib" "$R/scripts/"
  cp "$REPO/tests/kit-links.bats" "$R/tests/"
  printf 'name: CI\non: pull_request\njobs:\n  lint:\n    runs-on: ubuntu-latest\n    steps:\n      - name: shellcheck\n        run: shellcheck scripts/*.sh .githooks/*\n' \
    > "$R/.github/workflows/ci.yml"
  printf '16\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.current"
  printf '48129\n' > "$BATS_TEST_TMPDIR/cg/init.scope/pids.max"
  export CI_LOCAL_CGROUP_ROOT="$BATS_TEST_TMPDIR/cg"
  export CI_LOCAL_HOST_BUDGET_ARGS="--proc-root $REPO/tests/fixtures/host-budget/proc --gocache $BATS_TEST_TMPDIR/gocache --disk-min-free 1"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid
  git -C "$R" init -q -b main
  git -C "$R" add -A
  git -C "$R" commit -q -m base
  git -C "$R" update-ref refs/remotes/origin/main HEAD
  BASE="$(git -C "$R" rev-parse HEAD)"
}

# dangling_fragment — a valid ledger.d fragment whose log line names a bare nothere.md (the #706 form).
dangling_fragment() {
  printf '%s\n' '<!-- ledger: kit -->' '<!-- date: 2026-11-16 -->' '<!-- last-change-note: a fixture fragment -->' \
    'log: 2026-11-16 · a fixture change that names nothere.md in prose.' \
    > "$R/build-n4-module-kit/ledger.d/2026-11-16-fixture.md"
}

@test "CL30: a ledger.d fragment naming a bare nothere.md fails --changed with the kit-links L1 message" {
  # Mutation: CL30 -- dropping the run_kit_links call from scripts/ci-local.sh --changed lets this change through.
  dangling_fragment
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"dangling reference: nothere.md"* ]]
  [[ "$output" == *"FAIL kit-links"* ]]
  # the pushed form (.githooks/pre-push passes the range) reads the bytes of the range's end commit
  git add -A
  git commit -q -m "docs: fixture"
  run bash scripts/ci-local.sh --changed --range "$BASE..$(git rev-parse HEAD)"
  [ "$status" -eq 1 ]
  [[ "$output" == *"dangling reference: nothere.md"* ]]
}

@test "CL31: a kit doc change whose references all resolve passes --changed and writes nothing in the checkout" {
  # Mutation: CL31 -- materializing BUILD-STATE.md in the checkout instead of the snapshot leaves it here.
  printf '\nSee METHODOLOGY.md.\n' >> "$R/build-n4-module-kit/README.md"
  rm -f "$R/build-n4-module-kit/BUILD-STATE.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS kit-links"* ]]
  [ ! -e "$R/build-n4-module-kit/BUILD-STATE.md" ]
}

@test "CL32: a change outside the kit docs, ledger.d and changelog.d does not run kit-links (unchanged)" {
  # Mutation: CL32 -- running kit-links on every change costs every push a kit snapshot.
  printf 'x\n' > "$R/scripts/notes.txt"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"no changed file needs shellcheck, the bats lints or gofmt"* ]]
  [[ "$output" != *"kit-links"* ]]
}

@test "CL33: a changelog.d-only change runs kit-links" {
  # Mutation: CL33 -- dropping changelog.d from kit_docs_changed's pathspec skips kit-links on a fragment-only push.
  mkdir -p "$R/changelog.d"
  printf '%s\n' '### Fixed' '- a fixture fragment.' > "$R/changelog.d/2026-11-16-fixture.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS kit-links"* ]]
}

# stub_provisioner <body> — redefine provision_bats in the fixture's copy of lib/ci-local-tools.sh.
stub_provisioner() {
  printf 'provision_bats() { %s; }\n' "$1" >> "$R/scripts/lib/ci-local-tools.sh"
}

@test "CL34: a failed bats provisioning exits 3 with one line and runs no kit-links" {
  # Mutation: CL34 -- ignoring provision_bats' status runs kit-links with whatever bats is on PATH.
  stub_provisioner 'printf "stub: provisioning failed\n" >&2; return 1'
  printf '\nSee METHODOLOGY.md.\n' >> "$R/build-n4-module-kit/README.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 3 ]
  [ "$(grep -c 'kit-links: bats could not be provisioned' <<< "$output")" -eq 1 ]
  [[ "$output" != *"== kit-links"* ]]
}

@test "CL35: a provisioning that leaves another bats version first on PATH is a failure, not a pass" {
  # Mutation: CL35 -- dropping the pinned-version check passes kit-links on a bats CI does not run.
  printf '      - name: Install bats\n        run: |\n          BATS_VER=0.0.1\n          BATS_SHA256=%s\n' \
    "$(printf '0%.0s' $(seq 64))" >> "$R/.github/workflows/ci.yml"
  stub_provisioner 'return 0'
  printf '\nSee METHODOLOGY.md.\n' >> "$R/build-n4-module-kit/README.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 3 ]
  [[ "$output" == *"kit-links: bats could not be provisioned"* ]]
  [[ "$output" != *"PASS kit-links"* ]]
}

@test "CL36: a git error while detecting a kit doc change is reported, not read as no change" {
  # Mutation: CL36 -- reading a failed git diff as "no kit change" skips kit-links silently.
  # Through the CLI, run_changed exits 3 on a bad range first; the guard is pinned on run_kit_links itself.
  ROOT="$R" RANGE="$BASE..refs/heads/no-such-branch" WORK="$BATS_TEST_TMPDIR/w"
  # shellcheck disable=SC1091  # the fixture's copy of the lib
  . "$R/scripts/lib/ci-local-kitlinks.sh"
  run run_kit_links
  [ "$status" -eq 3 ]
  [[ "$output" == *"kit-links: cannot tell whether the change touches a kit doc"* ]]
}

@test "CL37: --range runs the range end's kit-state launcher, not the checkout's" {
  # Mutation: CL37 -- running the checkout's toolbelt/kit-state.sh checks a launcher the push does not send.
  printf '\nSee METHODOLOGY.md.\n' >> "$R/build-n4-module-kit/README.md"
  printf '#!/usr/bin/env bash\nprintf "pushed launcher\\n" >&2\nexit 1\n' > "$R/build-n4-module-kit/toolbelt/kit-state.sh"
  cd "$R"
  git add -A
  git commit -q -m "docs: fixture"
  git checkout -q "$BASE" -- build-n4-module-kit/toolbelt/kit-state.sh
  run bash scripts/ci-local.sh --changed --range "$BASE..$(git rev-parse HEAD)"
  [ "$status" -eq 1 ]
  [[ "$output" == *"pushed launcher"* ]]
  [[ "$output" == *"FAIL kit-state"* ]]
}

@test "CL38: kit-links runs the bats binary whose version was checked, not one a relative PATH entry finds in the snapshot" {
  # Mutation: CL38 -- running a bare "bats" inside the snapshot lets a relative PATH entry swap in an unchecked binary.
  local pin; pin="$(bats --version | awk '{print $2}')"
  printf '      - name: Install bats\n        run: |\n          BATS_VER=%s\n          BATS_SHA256=%s\n' \
    "$pin" "$(printf '0%.0s' $(seq 64))" >> "$R/.github/workflows/ci.yml"
  stub_provisioner 'return 0'
  mkdir -p "$R/tests/stubbin"
  printf '#!/bin/sh\n: > "%s/stub-ran"\nprintf "Bats 0.0.1\\n"\n' "$BATS_TEST_TMPDIR" > "$R/tests/stubbin/bats"
  chmod +x "$R/tests/stubbin/bats"
  printf '\nSee METHODOLOGY.md.\n' >> "$R/build-n4-module-kit/README.md"
  mkdir -p "$BATS_TEST_TMPDIR/elsewhere"
  cd "$BATS_TEST_TMPDIR/elsewhere"
  PATH="tests/stubbin:$PATH" run bash "$R/scripts/ci-local.sh" --changed
  [ "$status" -eq 0 ]
  [[ "$output" == *"PASS kit-links"* ]]
  [ ! -e "$BATS_TEST_TMPDIR/stub-ran" ]
}

@test "CL39: a bats that is not the pinned version is named with its path in the one failure line" {
  # Mutation: CL39 -- a bare "could not be provisioned" leaves the writer guessing which bats ran.
  printf '      - name: Install bats\n        run: |\n          BATS_VER=0.0.1\n          BATS_SHA256=%s\n' \
    "$(printf '0%.0s' $(seq 64))" >> "$R/.github/workflows/ci.yml"
  stub_provisioner 'return 0'
  printf '\nSee METHODOLOGY.md.\n' >> "$R/build-n4-module-kit/README.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 3 ]
  [ "$(grep -c 'kit-links: bats could not be provisioned' <<< "$output")" -eq 1 ]
  [[ "$output" == *"pins 0.0.1, found \"$(bats --version)\" at $(command -v bats)"* ]]
}

@test "CL40: a kit doc citing a corpus block the corpus index does not hold fails --changed (corpus-index CI-ALL)" {
  # Mutation: CL40 -- dropping the corpus-index check from run_kit_links lets this through to CI (#948 failed there).
  # shellcheck disable=SC2016  # backticks are markdown, not a command substitution
  printf '\nA new rule. `[ev: corpus B9998]`\n' >> "$R/build-n4-module-kit/BUILD-LOOP.md"
  cd "$R"
  run bash scripts/ci-local.sh --changed
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL corpus-index"* ]] || { echo "$output" | tail -15; false; }
  [[ "$output" == *"B9998"* ]]
}
