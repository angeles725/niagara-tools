#!/usr/bin/env bats
# install-client-hooks.bats — scripts/install-client-hooks.sh and the client pre-push hook template
# build-n4-module-kit/templates/client-pre-push (audit-2026-10-03 A142, issue #142 option 1).
#
# The client repositories have no CI, so the split-package check runs at push time: the installer copies the
# hook into <client-repo>/.githooks/, points core.hooksPath at it and records the kit toolbelt in the clone's
# LOCAL git config (n4kit.toolbelt, never committed). The hook blocks a push on a split package (checker exit
# 1) AND when the check cannot run (exit 3, no toolbelt, an unreadable commit) — it never passes silently.
# git is REAL here (throwaway repos); the installer and the hook live outside toolbelt/ (kit-links L2).

setup() {
  REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
  KIT="$REPO/build-n4-module-kit"
  INSTALLER="$REPO/scripts/install-client-hooks.sh"
  HOOK_TPL="$KIT/templates/client-pre-push"
  export GIT_CONFIG_GLOBAL="$BATS_TEST_TMPDIR/gitconfig"   # no host global hooksPath leaks in
  CR="$BATS_TEST_TMPDIR/client"
  mkdir -p "$CR"
  git -C "$CR" init -q
  git -C "$CR" config user.email t@e.st
  git -C "$CR" config user.name test
}

# _mod <dir> <package> <class> — one N4 module (module-include.xml + src/) declaring <package>
_mod() {
  mkdir -p "$CR/$1/src/x"
  printf '<module/>\n' > "$CR/$1/module-include.xml"
  printf 'package %s;\nclass %s {}\n' "$2" "$3" > "$CR/$1/src/x/$3.java"
}
_commit() { git -C "$CR" add -A && git -C "$CR" commit -q -m "$1"; }
_zero=0000000000000000000000000000000000000000

@test "ICH1: a directory that is not a git work tree is refused (exit 2), nothing written" {
  # Mutation: ICH1 -- dropping the work-tree check writes .githooks/ into a plain directory.
  mkdir -p "$BATS_TEST_TMPDIR/plain"
  run bash "$INSTALLER" "$BATS_TEST_TMPDIR/plain"
  [ "$status" -eq 2 ]
  [ ! -e "$BATS_TEST_TMPDIR/plain/.githooks" ]
}

@test "ICH2: install copies the hook (executable), sets core.hooksPath and the local toolbelt path" {
  run bash "$INSTALLER" "$CR"
  [ "$status" -eq 0 ]
  [ -x "$CR/.githooks/pre-push" ]
  cmp "$HOOK_TPL" "$CR/.githooks/pre-push"
  [ "$(git -C "$CR" config --local --get core.hooksPath)" = ".githooks" ]
  [ "$(git -C "$CR" config --local --get n4kit.toolbelt)" = "$KIT/toolbelt" ]
}

@test "ICH3: re-running install is idempotent (exit 0, same hook, same config)" {
  bash "$INSTALLER" "$CR"
  run bash "$INSTALLER" "$CR"
  [ "$status" -eq 0 ]
  cmp "$HOOK_TPL" "$CR/.githooks/pre-push"
  [ "$(git -C "$CR" config --local --get-all core.hooksPath | wc -l)" -eq 1 ]
}

@test "ICH4: a DIFFERENT existing pre-push is never overwritten without --force (exit 3, untouched)" {
  # Mutation: ICH4 -- dropping the content comparison overwrites the client's own hook.
  mkdir -p "$CR/.githooks"
  printf '#!/bin/sh\necho mine\n' > "$CR/.githooks/pre-push"
  run bash "$INSTALLER" "$CR"
  [ "$status" -eq 3 ]
  grep -q 'echo mine' "$CR/.githooks/pre-push"
  run bash "$INSTALLER" --force "$CR"
  [ "$status" -eq 0 ]
  cmp "$HOOK_TPL" "$CR/.githooks/pre-push"
}

@test "ICH5: a custom core.hooksPath is never replaced without --force (exit 3, untouched)" {
  # Mutation: ICH5 -- dropping the hooksPath check silently disables the client's custom hooks directory.
  git -C "$CR" config --local core.hooksPath .husky
  run bash "$INSTALLER" "$CR"
  [ "$status" -eq 3 ]
  [ "$(git -C "$CR" config --local --get core.hooksPath)" = ".husky" ]
  [ ! -e "$CR/.githooks/pre-push" ]
}

@test "ICH6: usage errors (no argument, unknown option) exit 2" {
  run bash "$INSTALLER"
  [ "$status" -eq 2 ]
  run bash "$INSTALLER" --bogus "$CR"
  [ "$status" -eq 2 ]
}

@test "CPH1: a pushed commit with no split package passes (exit 0)" {
  _mod A-rt com.acme.a A
  _mod B-rt com.acme.b B
  _commit one
  bash "$INSTALLER" "$CR"
  sha=$(git -C "$CR" rev-parse HEAD)
  run bash -c "cd '$CR' && printf 'refs/heads/main %s refs/heads/main %s\n' $sha $_zero | .githooks/pre-push origin /remote"
  [ "$status" -eq 0 ]
}

@test "CPH2: a pushed commit with a split package blocks the push (exit 1) and names the package" {
  # Mutation: CPH2 -- ignoring the checker's exit 1 lets a split package through.
  _mod A-rt com.acme.shared A
  _mod B-rt com.acme.shared B
  _commit split
  bash "$INSTALLER" "$CR"
  sha=$(git -C "$CR" rev-parse HEAD)
  run bash -c "cd '$CR' && printf 'refs/heads/main %s refs/heads/main %s\n' $sha $_zero | .githooks/pre-push origin /remote"
  [ "$status" -eq 1 ]
  [[ "$output" == *"com.acme.shared"* ]]
  [[ "$output" == *"push blocked"* ]]
}

@test "CPH3: the check cannot run (checker exit 3: no module in the commit) — the push is blocked, never passed" {
  # Mutation: CPH3 -- treating exit 3 as a pass lets an unchecked commit through silently.
  printf 'readme\n' > "$CR/README"
  _commit nomod
  bash "$INSTALLER" "$CR"
  sha=$(git -C "$CR" rev-parse HEAD)
  run bash -c "cd '$CR' && printf 'refs/heads/main %s refs/heads/main %s\n' $sha $_zero | .githooks/pre-push origin /remote"
  [ "$status" -ne 0 ]
  [[ "$output" == *"cannot verify"* ]]
}

@test "CPH4: no kit toolbelt (config unset, no env) blocks the push with a clear message" {
  # Mutation: CPH4 -- exiting 0 when the checker is missing skips the gate silently.
  _mod A-rt com.acme.a A
  _commit one
  bash "$INSTALLER" "$CR"
  git -C "$CR" config --local --unset n4kit.toolbelt
  sha=$(git -C "$CR" rev-parse HEAD)
  run bash -c "cd '$CR' && unset N4_KIT_TOOLBELT; printf 'refs/heads/main %s refs/heads/main %s\n' $sha $_zero | .githooks/pre-push origin /remote"
  [ "$status" -ne 0 ]
  [[ "$output" == *"split-package-check.sh"* ]]
}

@test "CPH5: every pushed ref is checked, not only the first stdin line" {
  # Mutation: CPH5 -- reading only the first ref line lets a split package on the second ref through.
  _mod A-rt com.acme.a A
  _commit clean
  clean=$(git -C "$CR" rev-parse HEAD)
  _mod B-rt com.acme.a B
  _commit split
  split=$(git -C "$CR" rev-parse HEAD)
  bash "$INSTALLER" "$CR"
  run bash -c "cd '$CR' && printf 'refs/heads/a %s refs/heads/a %s\nrefs/heads/b %s refs/heads/b %s\n' $clean $_zero $split $_zero | .githooks/pre-push origin /remote"
  [ "$status" -eq 1 ]
  [[ "$output" == *"com.acme.a"* ]]
}

@test "CPH6: a branch deletion pushes nothing to check (exit 0)" {
  _mod A-rt com.acme.a A
  _commit one
  bash "$INSTALLER" "$CR"
  run bash -c "cd '$CR' && printf '(delete) %s refs/heads/old %s\n' $_zero \$(git rev-parse HEAD) | .githooks/pre-push origin /remote"
  [ "$status" -eq 0 ]
}

@test "CPH7: the commit is checked, not the working tree (an uncommitted split does not block, a committed one does)" {
  _mod A-rt com.acme.a A
  _commit one
  bash "$INSTALLER" "$CR"
  _mod B-rt com.acme.a B   # uncommitted split in the working tree
  sha=$(git -C "$CR" rev-parse HEAD)
  run bash -c "cd '$CR' && printf 'refs/heads/main %s refs/heads/main %s\n' $sha $_zero | .githooks/pre-push origin /remote"
  [ "$status" -eq 0 ]
}

@test "CGA1: the CI template runs split-package-check --strict on the checkout and pins the kit to a tag" {
  T="$KIT/templates/split-package-check.yml"
  [ -f "$T" ]
  grep -q 'split-package-check.sh --strict' "$T"
  grep -qE 'ref: v[0-9]+\.[0-9]+\.[0-9]+' "$T"
  grep -q 'actions/checkout' "$T"
}

@test "ICH7: active hooks in the default hooks dir are never silently disabled (exit 3, nothing changed) (A142b)" {
  # Mutation: ICH7 -- dropping the default-hooks scan sets core.hooksPath and silently stops .git/hooks/pre-commit.
  printf '#!/bin/sh\nexit 0\n' > "$CR/.git/hooks/pre-commit"
  chmod +x "$CR/.git/hooks/pre-commit"
  run bash "$INSTALLER" "$CR"
  [ "$status" -eq 3 ]
  [[ "$output" == *"pre-commit"* ]]
  [ -z "$(git -C "$CR" config --local --get core.hooksPath || true)" ]
  [ ! -e "$CR/.githooks/pre-push" ]
  run bash "$INSTALLER" --force "$CR"
  [ "$status" -eq 0 ]
}

@test "ICH8: a GLOBAL core.hooksPath is never silently overridden by the local one (exit 3) (A142b)" {
  # Mutation: ICH8 -- reading only the local hooksPath lets the install shadow the global hooks directory.
  export GIT_CONFIG_GLOBAL="$BATS_TEST_TMPDIR/gitconfig"
  git config --global core.hooksPath "$BATS_TEST_TMPDIR/global-hooks"
  run bash "$INSTALLER" "$CR"
  [ "$status" -eq 3 ]
  [ -z "$(git -C "$CR" config --local --get core.hooksPath || true)" ]
}

@test "CPH8: a pushed commit that cannot be exported is reported as such and blocked (A142b)" {
  # Pin (GREEN on the base: tar also fails on an empty stream). pipefail makes a git archive failure win even when tar
  # would accept a truncated stream.
  _mod A-rt com.acme.a A
  _commit one
  bash "$INSTALLER" "$CR"
  bogus=1234567890123456789012345678901234567890
  run bash -c "cd '$CR' && printf 'refs/heads/main %s refs/heads/main %s\n' $bogus $_zero | .githooks/pre-push origin /remote"
  [ "$status" -eq 3 ]
  [[ "$output" == *"could not be exported"* ]]
}
