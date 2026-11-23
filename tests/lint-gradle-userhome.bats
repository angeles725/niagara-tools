#!/usr/bin/env bats
# lint-gradle-userhome.bats — a gradle.properties that sets user.home through org.gradle.jvmargs or systemProp.user.home
# WARNs (GU1, GU2). The rule's golden cases live in go/internal/lint/rules/gradleuserhome. [ev: retro r8-mirror-userhome]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  L="$KIT/toolbelt/lint-gradle-userhome.sh"
  ROOT="$BATS_TEST_TMPDIR/root"
  mkdir -p "$ROOT"
}

@test "GU-usage: no argument prints the usage and exits 3" {
  run "$L"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-gradle-userhome.sh [--strict] <root>" ]
}

@test "GU-fixture: the MinimalPan gradle.properties are clean under --strict" {
  run "$L" --strict "$KIT/fixtures/MinimalPan"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "GU1: jvmargs with -Duser.home= WARNs, also after a comment ending in a backslash; plain jvmargs and a commented line are clean" {
  # Mutation: GU1 -- continuing a comment line that ends in a backslash swallows the next line, a live jvmargs WARN.
  printf '%s\n' 'org.gradle.jvmargs=-Xmx2g' '# org.gradle.jvmargs=-Duser.home=/x' > "$ROOT/gradle.properties"
  run "$L" --strict "$ROOT"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  printf '%s\n' 'org.gradle.jvmargs=-Xmx2g -Duser.home=/tmp/x' >> "$ROOT/gradle.properties"
  run "$L" "$ROOT"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-gradle-userhome  $ROOT/gradle.properties:3  GU1 org.gradle.jvmargs sets -Duser.home=: an already-running gradle daemon never sees it; set it through JAVA_TOOL_OPTIONS and run with --no-daemon (toolbelt/build.sh mirror mode does)" ]
  run "$L" --strict "$ROOT"
  [ "$status" -eq 1 ]
  printf '%s\n' "# a comment \\" 'org.gradle.jvmargs=-Duser.home=/tmp/x' > "$ROOT/gradle.properties"
  run "$L" "$ROOT"
  [[ "$output" == "WARN  lint-gradle-userhome  $ROOT/gradle.properties:2  GU1 "* ]]
}

@test "GU2: systemProp.user.home WARNs; an unrelated key that mentions -Duser.home= is clean" {
  # Mutation: GU2 -- matching -Duser.home= in any property's value WARNs a key that is not jvmargs.
  printf '%s\n' 'note=use -Duser.home=/x on the cli' 'systemProp.user.home=/tmp/x' > "$ROOT/gradle.properties"
  run "$L" "$ROOT"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-gradle-userhome  $ROOT/gradle.properties:2  GU2 systemProp.user.home sets the home of no already-running gradle daemon; set -Duser.home= through JAVA_TOOL_OPTIONS and run with --no-daemon (toolbelt/build.sh mirror mode does)" ]
}

@test "GU-K20: a root with no gradle.properties is the ERROR row, exit 3" {
  run "$L" "$ROOT"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-gradle-userhome  $ROOT  no gradle.properties found (wrong path: pass a module or gradle root)" ]
}
