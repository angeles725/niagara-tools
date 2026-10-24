#!/usr/bin/env bats
# lint-inline-site-literal.bats — G42: site or station data (an ORD, an IPv4 address, a --site-pattern name) inline
# outside the site table is a WARN row; the rule's table tests live in go/internal/lint/rules/inlinesiteliteral.
# [ev: retro go-lint-inline-site-literal Δ1]

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  ISL="$KIT/toolbelt/lint-inline-site-literal.sh"
  MOD="$BATS_TEST_TMPDIR/Mod"
  RT="$MOD/Mod-rt/src/com/x"; UX="$MOD/Mod-ux/src/com/x/ux"; RC="$MOD/Mod-ux/src/rc"
  mkdir -p "$RT" "$UX" "$RC"
  ORD='station:|slot:/Rooms/Room1'
}

@test "ISL-usage: no argument prints the usage and exits 3" {
  run "$ISL"
  [ "$status" -eq 3 ]
  [ "$output" = "usage: lint-inline-site-literal.sh [--strict] [--site-table <path>]... [--site-pattern <regex>]... <module-root>" ]
}

@test "ISL-K20: a root with no part src/ or rc/ file is the ERROR row, exit 3, never a clean pass" {
  # Mutation: ISL-K20 -- dropping the no-part ErrEmptyRoot reports a wrong-path root as a clean exit 0
  mkdir -p "$MOD/tools"; printf 'class T { String o = "%s"; }\n' "$ORD" > "$MOD/tools/T.java"
  rm -rf "$MOD/Mod-rt" "$MOD/Mod-ux"
  run "$ISL" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "ERROR  lint-inline-site-literal  $MOD  no *.java under a -rt or -ux part's src/ and no *.js under an rc/ (wrong path: pass the module root)" ]
}

@test "ISL1: an ORD in a logic class is a WARN row, exit 0" {
  # Mutation: ISL1 -- dropping the ORD match passes BOrd.make("station:|slot:/Rooms/Room1") in a logic class clean
  printf 'package com.x;\nclass BRoomControl {\n  BOrd room = BOrd.make("%s");\n}\n' "$ORD" > "$RT/BRoomControl.java"
  run "$ISL" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-inline-site-literal  $RT/BRoomControl.java:3  ord literal outside the site table: \"$ORD\"" ]
}

@test "ISL2: the same ORD in the --site-table class or a // site-table file is clean" {
  # Mutation: ISL2 -- dropping the site-table check flags the same ORD in the --site-table class
  printf 'package com.x;\nclass Site {\n  static final String ROOM1 = "%s";\n}\n' "$ORD" > "$RT/Site.java"
  printf '// site-table\nvar ROOM1 = "%s";\n' "$ORD" > "$RC/site.js"
  run "$ISL" --site-table "$RT/Site.java" "$MOD"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "ISL3: an IPv4 address in a servlet is a WARN row" {
  # Mutation: ISL3 -- dropping the IPv4 match passes "192.168.1.20" in a servlet clean
  printf 'package com.x.ux;\nclass BRoomServlet {\n  static final String HOST = "192.168.1.20";\n}\n' > "$UX/BRoomServlet.java"
  run "$ISL" "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-inline-site-literal  $UX/BRoomServlet.java:3  ip literal outside the site table: \"192.168.1.20\"" ]
}

@test "ISL4: a --site-pattern room name in rc/app.js is a WARN row; a comment and a test source are not read" {
  # Mutation: ISL4 -- dropping --site-pattern passes "Room3" in rc/app.js clean
  printf 'var name = "Room3";\n// fetch("%s");\n' "$ORD" > "$RC/app.js"
  mkdir -p "$MOD/Mod-rt/srcTest/com/x"; printf 'class T { String o = "%s"; }\n' "$ORD" > "$MOD/Mod-rt/srcTest/com/x/T.java"
  run "$ISL" --site-pattern 'Room[0-9]+' "$MOD"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-inline-site-literal  $RC/app.js:1  site literal outside the site table: \"Room3\"" ]
}

@test "ISL-strict: a row exits 1 under --strict; a missing --site-table exits 3" {
  # Mutation: ISL-strict -- giving the rule a non-WARN severity exits 0 under --strict with a row
  printf 'class B { String h = "10.0.0.7"; }\n' > "$RT/B.java"
  run "$ISL" --strict "$MOD"
  [ "$status" -eq 1 ]
  run "$ISL" --site-table "$MOD/nope.java" "$MOD"
  [ "$status" -eq 3 ]
  [ "$output" = "lint-inline-site-literal: --site-table file not found: $MOD/nope.java" ]
}
