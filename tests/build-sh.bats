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

@test "B7: --help prints only the header comment, no code line leaks (usage sed range drift)" {
  run "$B" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
  [[ "$output" != *"set -euo pipefail"* ]]
  [[ "$output" != *"usage()"* ]]
}

# ================= Campaign 3 C3-PR2 (RED-first) =================
# Three owed build.sh impls. Each mutation-bites; fake gradlew logs argv (setup), verify-module stubbed.

@test "B8 (lesson B6, module-palette-and-build-target): auto-detect the plugin version from the module's OWN gradle files" {
  # gradle.properties can lie about the target, but the pinned niagara plugin version is the module's own
  # config — build.sh must read it and forward -PniagaraPluginVersion WITHOUT requiring --plugin-version.
  printf 'niagaraPluginVersion=7.6.22\n' > "$ROOT/gradle.properties"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"                       # NO --plugin-version
  [ "$status" -eq 0 ]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"-PniagaraPluginVersion=7.6.22"* ]]   # detected, not empty
  # explicit --plugin-version still WINS over the detected one (precedence)
  : > "$TMPDIR_T/gradlew.calls.log"
  run "$B" --plugin-version 7.3.40 "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *"-PniagaraPluginVersion=7.3.40"* ]]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" != *"7.6.22"* ]]
}

@test "B9 (lesson B7, dashboardpan-ux-direct-build): walk UP to the gradle root when ROOT has no ./gradlew (nested multi-project)" {
  # A client repo keeps ./gradlew at the repo root; the module dir passed as ROOT has none.
  # Today build.sh exits 10 'no executable ./gradlew'; it must walk up to find gradlew + settings.gradle*.
  proj="$TMPDIR_T/proj"; mkdir -p "$proj"; make_fake_gradlew "$proj"; : > "$proj/settings.gradle.kts"
  root2="$proj/client"; mkdir -p "$root2"; make_profile "$root2" Foo rt 1
  run "$B" "$root2" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 0 ]
  [[ "$(cat "$TMPDIR_T/gradlew.calls.log")" == *":Foo-rt:jar"* ]]   # gradle ran from the resolved root
}

@test "B10 (lesson soft-start): a modules/<jar> clean-lock prints an ACTIONABLE message + distinct exit 31, not a raw exit 30" {
  # A running station locks modules/<mod>.jar so :clean fails with an IOException. build.sh must detect
  # that specific failure and tell the operator how to fix it (free the lock / build/libs / mirror),
  # instead of the generic 'gradle failed' exit 30.
  cat > "$ROOT/gradlew" <<'GRADLEW'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$TMPDIR_T/gradlew.calls.log"
echo "> Task :Foo-rt:clean FAILED" >&2
echo "Unable to delete file '$TMPDIR_T/nh/modules/Foo-rt.jar'" >&2
exit 1
GRADLEW
  chmod +x "$ROOT/gradlew"
  run "$B" "$ROOT" Foo "$TMPDIR_T/nh"
  [ "$status" -eq 31 ]                                   # distinct from generic gradle-fail 30
  { [[ "$output" == *"locked"* ]] || [[ "$output" == *"Unable to delete"* ]]; }
  { [[ "$output" == *"Workbench"* ]] || [[ "$output" == *"mirror"* ]] || [[ "$output" == *"build/libs"* ]]; }
  [ ! -e "$TMPDIR_T/verify.args" ]                       # gate not reached on a failed build
}
