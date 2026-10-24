#!/usr/bin/env bats
# Tests for scripts/check-installed-line.sh — refuse a client branch that would downgrade a module group already
# installed. Fixture = the PANCCADIA incident (fix/r10-coldroompan-token-leak branched from an old commit):
# installed Dashboard 2.10.0 / Compresores 2.8.0 / Paccadia 2.4.2, branch Dashboard 2.9.2 / 2.8.0 / 2.4.4.
# [ev: retro r10-installed-line-guard Δ1]
# Named mutations: drop the -1 branch -> ILG-downgrade red; read "incomparable" as ok -> ILG-incomparable red;
# skip groups missing on the branch -> ILG-missing red.

setup() {
  TMPDIR_T="$(mktemp -d)"
  CK="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/scripts/check-installed-line.sh"
  export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1
  export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.invalid
  export GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.invalid
  R="$TMPDIR_T/r"
  git init -q -b main "$R"
  _versions 2.10.0 2.8.0 2.4.2
  git -C "$R" add -A; git -C "$R" commit -q -m installed; git -C "$R" tag installed
  git -C "$R" checkout -q -b fix
}
teardown() { rm -rf "$TMPDIR_T"; }

# _versions <dashboard> <compresores> <paccadia> — write each GROUP build.gradle.kts with that version
_versions() {
  local g v
  for g in Dashboard:"$1" Compresores:"$2" Paccadia:"$3"; do
    v=${g#*:}; g=${g%%:*}; mkdir -p "$R/$g"
    printf 'niagaraModule {\n  // defaultModuleVersion sets the "vendorVersion" attribute on all modules\n  defaultModuleVersion("%s")\n}\n' \
      "$v" > "$R/$g/build.gradle.kts"
  done
}
_commit() { git -C "$R" add -A; git -C "$R" commit -q -m "$1"; }

@test "ILG-downgrade: the incident shape -> exit 1, DOWNGRADE names the file and both versions" {
  _versions 2.9.2 2.8.0 2.4.4; _commit old-base
  run "$CK" "$R" fix installed
  [ "$status" -eq 1 ]
  [[ "$output" == *"DOWNGRADE Dashboard/build.gradle.kts 2.10.0 -> 2.9.2"* ]]
  [[ "$output" == *"ok Compresores/build.gradle.kts 2.8.0 -> 2.8.0"* ]]
  [[ "$output" == *"ok Paccadia/build.gradle.kts 2.4.2 -> 2.4.4"* ]]
}

@test "ILG-equal-higher: every group equal or higher (numeric: 2.10.0 -> 2.10.1, 2.4.2 -> 2.4.10) -> exit 0" {
  _versions 2.10.1 2.8.0 2.4.10; _commit bump
  run "$CK" "$R" fix installed
  [ "$status" -eq 0 ]
  [[ "$output" != *DOWNGRADE* ]]
  [[ "$output" == *"ok Paccadia/build.gradle.kts 2.4.2 -> 2.4.10"* ]]
}

@test "ILG-missing: a group installed but absent on the branch -> exit 1, MISSING names it" {
  git -C "$R" rm -q -r Compresores; _commit drop
  run "$CK" "$R" fix installed
  [ "$status" -eq 1 ]
  [[ "$output" == *"MISSING Compresores/build.gradle.kts 2.8.0"* ]]
}

@test "ILG-incomparable: a differing suffix (2.10.0 -> 2.10.0-rc) refuses, exit 1, named" {
  _versions 2.10.0-rc 2.8.0 2.4.2; _commit rc
  run "$CK" "$R" fix installed
  [ "$status" -eq 1 ]
  [[ "$output" == *"INCOMPARABLE Dashboard/build.gradle.kts 2.10.0 -> 2.10.0-rc"* ]]
}

@test "ILG-unknown-ref: an unknown branch or installed ref -> exit 2; wrong arg count -> exit 2" {
  run "$CK" "$R" no-such-branch installed
  [ "$status" -eq 2 ]; [[ "$output" == *"unknown ref no-such-branch"* ]]
  run "$CK" "$R" fix no-such-ref
  [ "$status" -eq 2 ]
  run "$CK" "$R" fix
  [ "$status" -eq 2 ]
}

@test "ILG-no-groups: an installed ref with no defaultModuleVersion -> exit 2, never a silent pass" {
  git -C "$R" checkout -q --orphan empty; git -C "$R" rm -q -r -f .; printf 'x\n' > "$R/README"; _commit empty
  run "$CK" "$R" fix empty
  [ "$status" -eq 2 ]; [[ "$output" == *"no defaultModuleVersion"* ]]
}
