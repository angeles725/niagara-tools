#!/usr/bin/env bats
# lint-machine-default-path.bats — G43: a default value that is an absolute machine path (developer home,
# temporary directory, scratchpad) is an ADVISORY row; the rule's table tests live in
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
  [ "$output" = "usage: lint-machine-default-path.sh [--strict] <src-root>..." ]
}

@test "MDP-K20: an empty root is the ERROR row, exit 3, never a clean pass" {
  # Mutation: MDP-K20 -- dropping the engine empty-root guard reports an empty root as a clean exit 0
  run "$MDP" "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-machine-default-path  $BATS_TEST_TMPDIR/Mod  no *.java sources found (wrong path or empty source tree)" ]
}

@test "MDP1: a home-directory field default is one ADVISORY row and exit 0, --strict included" {
  # Mutation: MDP1 -- dropping the field-initializer context passes static final String DIR = "/home/me/x" clean
  printf 'package com.x;\npublic class BCfg {\n  static final String DIR = "/home/me/x";\n}\n' > "$SRC/BCfg.java"
  run "$MDP" --strict "$BATS_TEST_TMPDIR/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "ADVISORY  lint-machine-default-path  $SRC/BCfg.java:3  machine-specific default path \"/home/me/x\" in a field initializer -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)" ]
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
