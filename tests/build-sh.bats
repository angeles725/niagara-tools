#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/build.sh — preflight, profile selection, gate call.
# gradle is never run: a fake gradlew records its argv. verify-module.sh is stubbed to isolate build.sh.

load helpers/n4-fixtures

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  # copy build.sh next to a stub gate so the sibling lookup resolves to the stub
  mkdir -p "$TMPDIR_T/kit"; cp "$KIT/toolbelt/build.sh" "$TMPDIR_T/kit/build.sh"
  # shellcheck disable=SC2016
  # why: $* and ${FAKE_VERIFY_EXIT} must reach the generated stub unexpanded
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" > "%s/verify.args"\nexit "${FAKE_VERIFY_EXIT:-0}"\n' "$TMPDIR_T" > "$TMPDIR_T/kit/verify-module.sh"
  chmod +x "$TMPDIR_T/kit/verify-module.sh"
  B="$TMPDIR_T/kit/build.sh"
  ROOT="$TMPDIR_T/mod"; mkdir -p "$ROOT"
  make_profile "$ROOT" Foo rt 1; make_profile "$ROOT" Foo ux 1; make_profile "$ROOT" Foo wb 0
  make_fake_gradlew "$ROOT"
  make_niagara_home "$TMPDIR_T/nh" 7.6.17
  mkdir -p "$TMPDIR_T/j8"; export JAVA8="$TMPDIR_T/j8"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "B1: stub -wb (gradle file, no sources) is skipped; rt+ux get clean+slotomatic+jar on Java 8 (DashboardPan-wb regression)" {
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$output" == *"skipping Foo-wb"* ]]
  args="$(cat "$TMPDIR_T/gradlew.calls.log")"
  [[ "$args" == *":Foo-rt:clean :Foo-rt:slotomatic :Foo-rt:jar :Foo-ux:clean :Foo-ux:slotomatic :Foo-ux:jar"* ]]
  [[ "$args" != *":Foo-wb:"* ]]
  [[ "$args" == *"-Porg.gradle.java.installations.paths=$TMPDIR_T/j8"* ]]
  [[ "$args" == *"-Pniagara_home=$TMPDIR_T/nh"* ]]
}

@test "B2: no args prints usage and exits 2 (bare \${1:?} abort)" {
  run "$B"
  [ "$status" -eq 2 ]
  [[ "$output" == *"Usage:"* ]]
  [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
}

@test "B3: a target without etc/m2 exits 10 before gradle (plugin-not-found failure)" {
  mkdir -p "$TMPDIR_T/fake"
  run "$B" "$ROOT" Foo "$TMPDIR_T/fake"
  [ "$status" -eq 10 ]
  [[ "$output" == *"not a niagara_home"* ]]
  [ ! -e "$TMPDIR_T/gradlew.calls.log" ]
}

@test "B4: --profiles overrides detection entirely (only rt tasks run)" {
  run "$B" --profiles rt "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  args="$(cat "$TMPDIR_T/gradlew.calls.log")"
  [[ "$args" == *":Foo-rt:jar"* ]] && [[ "$args" != *":Foo-ux:"* ]] && [[ "$args" != *":Foo-wb:"* ]]
}

@test "B5: verify-module.sh runs after gradle with --src, --target-version and the jars; its failure exits 50 (gate skip)" {
  run "$B" --target-version 4.14 --plugin-version 7.6.17 "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  v="$(cat "$TMPDIR_T/verify.args")"
  [[ "$v" == *"--src $ROOT/Foo"* ]] && [[ "$v" == *"--target-version 4.14"* ]]
  [[ "$v" == *"Foo-rt/build/libs/Foo-rt.jar"* ]] && [[ "$v" == *"Foo-ux/build/libs/Foo-ux.jar"* ]]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"-PniagaraPluginVersion=7.6.17"* ]]
  FAKE_VERIFY_EXIT=1 run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 50 ]
  [[ "$output" == *"do not deploy"* ]]
}

@test "B6: gradle failure exits 30 and the gate is not invoked" {
  FAKE_GRADLEW_EXIT=1 run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 30 ]
  [ ! -e "$TMPDIR_T/verify.args" ]
}
