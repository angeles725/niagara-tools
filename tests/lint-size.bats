#!/usr/bin/env bats
# lint-size.bats — advisory class/method size smell (deferred-lints D2).
# [ev: retro decision-logic-decomposition Δ2] Fixtures are generated at test time (sizes, not content, matter).

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LSZ="$KIT/toolbelt/lint-size.sh"
  SRC="$TMPDIR_T/Mod/src/com/x"
  mkdir -p "$SRC"
}
teardown() { rm -rf "$TMPDIR_T"; }

# lines <n> <text> — n copies of a code line
lines() { local i; for ((i = 1; i <= $1; i++)); do printf '  %s%d;\n' "$2" "$i"; done; }
BEGIN_M='//region /*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/'
END_M='//endregion /*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/'

@test "LSZ1: a class over 800 hand-written lines WARNs with its count, exit 0 without --strict" {
  { printf 'package com.x;\npublic class Big {\n'; lines 800 'int f'; printf '}\n'; } > "$SRC/Big.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-size  "*"Big.java:2  class Big has 803 hand-written lines (> 800"* ]]
}

@test "LSZ1-edge: exactly 800 hand-written lines is not a WARN (strictly greater)" {
  { printf 'package com.x;\npublic class Edge {\n'; lines 797 'int f'; printf '}\n'; } > "$SRC/Edge.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LSZ2: slotomatic region lines (the //region-prefixed form slot-o-matic writes) are not hand-written" {
  # Mutation: LSZ2 -- counting the slotomatic region lines as hand-written WARNs a small class with a large generated block
  { printf 'package com.x;\n@NiagaraType\npublic class BGen extends BComponent {\n%s\n' "$BEGIN_M"
    lines 900 'int g'; printf '%s\n' "$END_M"; lines 10 'int h'; printf '}\n'; } > "$SRC/BGen.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LSZ2-bare: the bare /*+ BEGIN … +*/ marker form is excluded too" {
  { printf 'package com.x;\npublic class Gen2 {\n/*+ ------------ BEGIN BAJA AUTO GENERATED CODE ------------ +*/\n'
    lines 900 'int g'; printf '/*+ ------------ END BAJA AUTO GENERATED CODE -------------- +*/\n}\n'; } > "$SRC/Gen2.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ -z "$output" ]
}

@test "LSZ3: a method over 80 lines in a pure class WARNs with its span" {
  { printf 'package com.x;\npublic class Core {\n  int step() {\n'; lines 80 'a'; printf '  }\n}\n'; } > "$SRC/Core.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Core.java:3  method step() in pure class Core spans 82 lines (> 80)"* ]]
}

@test "LSZ3-edge: a pure method of exactly 80 lines is not a WARN" {
  { printf 'package com.x;\npublic class Core2 {\n  int step() {\n'; lines 78 'a'; printf '  }\n}\n'; } > "$SRC/Core2.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ -z "$output" ]
}

@test "LSZ4: a long method of a Baja class (@NiagaraType / extends B<Upper>) is not a method WARN" {
  # Mutation: LSZ4 -- applying the method rule to a Baja (BComponent) class WARNs its long callback
  { printf 'package com.x;\npublic class BUnit extends BComponent {\n  public void changed(Property p, Context cx) {\n'
    lines 120 'a'; printf '  }\n}\n'; } > "$SRC/BUnit.java"
  { printf 'package com.x;\n@NiagaraType\npublic class BOther extends Object {\n  void run() {\n'
    lines 120 'a'; printf '  }\n}\n'; } > "$SRC/BOther.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LSZ5: an unclosed BEGIN marker is a WARN and its lines count as hand-written (fail closed)" {
  # Mutation: LSZ5 -- dropping the unclosed-region row lets an unterminated BEGIN marker pass silently
  { printf 'package com.x;\npublic class Open {\n%s\n' "$BEGIN_M"; lines 3 'int g'; printf '}\n'; } > "$SRC/Open.java"
  run "$LSZ" --max-class 4 "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Open.java:3  unclosed-region"* ]]
  [[ "$output" == *"class Open has 6 hand-written lines (> 4"* ]]
}

@test "LSZ6: --max-class / --max-method override the limits; comments and blank lines are not counted" {
  { printf 'package com.x;\n// a comment\n\n/* block\n   comment */\npublic class Small {\n  int a() {\n    x();\n  }\n}\n'; } > "$SRC/Small.java"
  run "$LSZ" --max-class 5 --max-method 2 "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"class Small has 6 hand-written lines (> 5"* ]]
  [[ "$output" == *"method a() in pure class Small spans 3 lines (> 2)"* ]]
}

@test "LSZ7: --strict exits 1 on a WARN; a usage error exits 3" {
  { printf 'package com.x;\npublic class Core {\n  int step() {\n'; lines 90 'a'; printf '  }\n}\n'; } > "$SRC/Core.java"
  run "$LSZ" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  run "$LSZ" --max-class abc "$TMPDIR_T/Mod"
  [ "$status" -eq 3 ]
  run "$LSZ"
  [ "$status" -eq 3 ]
  run "$LSZ" "$TMPDIR_T/nope"
  [ "$status" -eq 3 ]
}

@test "LSZ-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: LSZ-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
  printf 'x\n' > "$SRC/U.java"
  chmod 000 "$SRC/U.java"
  if [ -r "$SRC/U.java" ]; then chmod 644 "$SRC/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$LSZ" "$TMPDIR_T/Mod"
  chmod 644 "$SRC/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "LSZ-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: LSZ-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
  mkdir -p "$SRC/locked"
  printf 'class B {}\n' > "$SRC/locked/B.java"
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$LSZ" "$TMPDIR_T/Mod"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/locked"* ]]
}

@test "LSZ8: a BEGIN inside an open region is its own WARN, never silently merged (D2b, R3-002)" {
  # Mutation: LSZ8 -- ignoring a nested BEGIN lets a stray END close an unterminated region and hide its lines
  { printf 'package com.x;\npublic class Nest {\n%s\n' "$BEGIN_M"; lines 3 'int a'
    printf '%s\n' "$BEGIN_M"; lines 2 'int b'; printf '%s\n}\n' "$END_M"; } > "$SRC/Nest.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [[ "$output" == *"Nest.java:7  nested-region"* ]]
}

@test "LSZ9: an END with no open region is its own WARN (D2b, R3-002)" {
  { printf 'package com.x;\npublic class Stray {\n  int a;\n%s\n}\n' "$END_M"; } > "$SRC/Stray.java"
  run "$LSZ" "$TMPDIR_T/Mod"
  [[ "$output" == *"Stray.java:4  stray-end"* ]]
}
