#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
# The live-doctrine citation gate of .githooks/pre-push: a range that changes a live build-n4-module card
# (scripts/lib/live-doctrine-set.sh) or the resolver itself runs kit-cite-resolve --strict on the live set of
# the pushed tree, before every early exit. 1 = findings → refuse; 3 with a research/modules/clients/memory
# root absent → a named SKIPPED line and the push proceeds; any other non-zero → refuse fail-closed.
# Hermetic: git is a fakebin and the resolver is a stub in a fixture toplevel (no research roots read).

setup() {
  # the client-identifier step of the hook: hermetic whatever this host has installed (no denylist → it skips)
  export NIAGARA_CLIENT_DENYLIST="$BATS_TEST_TMPDIR/no-client-denylist"
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  HOOK="$REPO/.githooks/pre-push"
  TOP="$BATS_TEST_TMPDIR/top"; K="$TOP/build-n4-module-kit"
  mkdir -p "$K/types" "$K/toolbelt" "$K/skill" "$TOP/go/internal/citeresolve"
  printf '# logic\n' > "$K/types/logic.md"
  printf '# skill\n' > "$K/skill/SKILL.md"
  printf '# readme\n' > "$TOP/README.md"
  # the stub resolver: records its argv, prints one row per STUB_RC
  cat > "$K/toolbelt/kit-cite-resolve.sh" <<'STUB'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$STUB_CALLS"
case "${STUB_RC:-0}" in
  1) printf 'build-n4-module-kit/types/logic.md\t3\tpath\tnope.java\t\tMISSING_FILE\tno file\n'
     printf 'build-n4-module-kit/types/logic.md\t4\tretro\tok\tΔ1\tRESOLVED\tfine\n'
     echo 'kit-cite-resolve: strict findings: LIVE 1, HISTORY 0' >&2 ;;
  3) echo 'kit-cite-resolve: cannot read research root /no/research: open /no/research: no such file or directory' >&2 ;;
  4) echo 'kit-cite-resolve: cannot read memory root /m: open /m: permission denied' >&2; exit 3 ;;
esac
exit "${STUB_RC:-0}"
STUB
  STUB_CALLS="$BATS_TEST_TMPDIR/calls"; : > "$STUB_CALLS"; export STUB_CALLS
  FAKEBIN="$BATS_TEST_TMPDIR/bin"; mkdir -p "$FAKEBIN"
  cat > "$FAKEBIN/git" <<'GIT'
#!/usr/bin/env bash
case "$1" in
  rev-parse) echo "$FAKE_TOPLEVEL" ;;
  diff)      [[ " $* " == *" --quiet "* ]] && exit "${FAKE_TREE_DIFF_RC:-0}"; cat "$FAKE_CHANGED_FILE" ;;
  log)       cat "$FAKE_LOG_FILE" ;;
  ls-tree)   (cd "$FAKE_TOPLEVEL" && find . -type f | sed 's|^\./||' | sort) ;;
  *)         exit 0 ;;
esac
GIT
  chmod +x "$FAKEBIN/git"
  printf 'feat: x\n\nRetro: none (trivial: test)\n' > "$BATS_TEST_TMPDIR/log"
}

push() { # push <changed paths...>: the range changes these files
  printf '%s\n' "$@" > "$BATS_TEST_TMPDIR/changed"
  FAKE_TOPLEVEL="$TOP" FAKE_CHANGED_FILE="$BATS_TEST_TMPDIR/changed" FAKE_LOG_FILE="$BATS_TEST_TMPDIR/log" \
  run env PATH="$FAKEBIN:$PATH" bash "$HOOK" origin https://example/repo.git <<<'refs/heads/x aaaa refs/heads/x bbbb'
}

@test "LC1: a live card with findings (resolver exit 1) refuses the push and names the finding rows" {
  STUB_RC=1 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISSING_FILE"* ]]
  [[ "$output" == *"nope.java"* ]]
  [[ "$output" != *"	RESOLVED	"* ]]
}

@test "LC2: a live card that resolves (exit 0) is allowed, and the strict live set is what was checked" {
  STUB_RC=0 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 0 ]
  run cat "$STUB_CALLS"
  [[ "$output" == *"--strict"* ]]
  [[ "$output" == *"build-n4-module-kit/types/logic.md"* ]]
  [[ "$output" == *"build-n4-module-kit/skill/SKILL.md"* ]]
  [[ "$output" == *"README.md"* ]]
  [[ "$output" != *"kit-cite-resolve.sh "* ]]   # the tool file is no card
}

@test "LC3: exit 3 with a root absent prints the named SKIPPED line and the push proceeds" {
  STUB_RC=3 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 0 ]
  [[ "$output" == *"pre-push: live cite check SKIPPED — roots absent (research"* ]]
}

@test "LC4: exit 3 for another reason (a root unreadable, not absent) refuses fail-closed naming the cause" {
  STUB_RC=4 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"permission denied"* ]]
  [[ "$output" != *"SKIPPED"* ]]
}

@test "LC5: any other non-zero exit refuses fail-closed" {
  STUB_RC=7 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"exit 7"* ]]
}

@test "LC6: a range touching no live file never calls the resolver" {
  STUB_RC=1 push scripts/ng-deploy.sh build-n4-module-kit/toolbelt/build.sh
  [ "$status" -eq 0 ]
  [ ! -s "$STUB_CALLS" ]
}

@test "LC7: the trivial waiver does not waive the check (live card, findings → refused)" {
  grep -q 'Retro: none (trivial:' "$BATS_TEST_TMPDIR/log"
  STUB_RC=1 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 1 ]
  [ -s "$STUB_CALLS" ]
}

@test "LC8: a non-build-relevant live file (root README.md) is checked before the pass_early exit" {
  STUB_RC=1 push README.md
  [ "$status" -eq 1 ]
}

@test "LC9: a change to the resolver itself re-checks the live set" {
  STUB_RC=1 push go/internal/citeresolve/resolve.go
  [ "$status" -eq 1 ]
}

@test "LC10: a working tree that differs from the pushed commit on the live set refuses (never checks other bytes)" {
  FAKE_TREE_DIFF_RC=1 STUB_RC=0 push build-n4-module-kit/types/logic.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"differs from the pushed"* ]]
  [ ! -s "$STUB_CALLS" ]
}
