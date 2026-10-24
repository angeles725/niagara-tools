#!/usr/bin/env bats
# lint-machine-default-path.bats — G43: a default value that is an absolute machine path (developer home,
# temporary directory, scratchpad) in Java, rc JavaScript or .properties is a WARN row; the rule's table tests live in
# go/internal/lint/rules/machinedefaultpath. [ev: retro go-lint-machine-default-path Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  MDP="$KIT/toolbelt/lint-machine-default-path.sh"
  SRC="$BATS_TEST_TMPDIR/Mod/src/com/x"
  mkdir -p "$SRC"
}

@test "MDP-usage: no argument prints the usage and exits 3" {
  run "$MDP"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-machine-default-path.sh [--strict] <module-root>..." ]
}

@test "MDP-K20: an empty root is the ERROR row, exit 3, never a clean pass" {
  # Mutation: MDP-K20 -- dropping the engine empty-root guard reports an empty root as a clean exit 0
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-machine-default-path  $BATS_TEST_TMPDIR/Mod  no *.java or *.js or *.properties sources found (wrong path or empty source tree)" ]
}

HINT=" -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)"

@test "MDP1: a home-directory field default is one WARN row, exit 0" {
  # Mutation: MDP1 -- dropping the field-initializer context passes static final String DIR = "/home/me/x" clean
  printf 'package com.x;\npublic class BCfg {\n  static final String DIR = "/home/me/x";\n}\n' > "$SRC/BCfg.java"
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-machine-default-path  $SRC/BCfg.java:3  machine-specific default \"/home/me/x\" in a field initializer$HINT" ]
}

@test "MDP-strict: the same row exits 1 under --strict" {
  # Mutation: MDP-strict -- giving the rule a non-WARN severity exits 0 under --strict with a row
  printf 'package com.x;\npublic class BCfg {\n  static final String DIR = "/home/me/x";\n}\n' > "$SRC/BCfg.java"
  run "$MDP" --strict "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == "WARN  lint-machine-default-path  $SRC/BCfg.java:3  "* ]]
}

@test "MDP4: an rc JavaScript const default is a WARN row; the same line outside rc/ is not" {
  # Mutation: MDP4 -- dropping the rc JavaScript pass passes const DIR = "/home/me/x" in src/rc clean
  mkdir -p "$BATS_TEST_TMPDIR/Mod/src/rc/js" "$BATS_TEST_TMPDIR/Mod/tools"
  printf '(function () {\n  const DIR = "/home/me/x";\n})();\n' > "$BATS_TEST_TMPDIR/Mod/src/rc/js/app.js"
  printf 'const DIR = "/home/me/x";\n' > "$BATS_TEST_TMPDIR/Mod/tools/gen.js"
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-machine-default-path  $BATS_TEST_TMPDIR/Mod/src/rc/js/app.js:2  machine-specific default \"/home/me/x\" in a const/let/var initializer$HINT" ]
}

@test "MDP5: a Windows user home in gradle.properties is a WARN row with the loaded value" {
  # Mutation: MDP5 -- dropping the .properties pass passes niagara_user_home=C:\\Users\\... clean
  printf 'niagara_home=C:\\\\Niagara\\\\Niagara-4.14\nniagara_user_home=C:\\\\Users\\\\me\\\\N4\n' > "$BATS_TEST_TMPDIR/Mod/gradle.properties"
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-machine-default-path  $BATS_TEST_TMPDIR/Mod/gradle.properties:2  machine-specific default \"C:\\Users\\me\\N4\" in a .properties value$HINT" ]
}

@test "MDP2: a path quoted in a Javadoc and a station-relative default are clean" {
  # Mutation: MDP2 -- reading comment text flags a /home/ path quoted in a Javadoc
  printf 'package com.x;\n/** Was String DIR = "/home/me/x". */\npublic class BCfg {\n  static final String DIR = "file:^data/x";\n}\n' > "$SRC/BCfg.java"
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "MDP3: a local variable in a method is not a default" {
  # Mutation: MDP3 -- dropping the method-body check flags a local variable in a method
  printf 'package com.x;\npublic class BCfg {\n  void f() {\n    String d = "/tmp/x";\n  }\n}\n' > "$SRC/BCfg.java"
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}
