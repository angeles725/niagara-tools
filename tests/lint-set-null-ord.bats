#!/usr/bin/env bats
# lint-set-null-ord.bats — a getSlotPathOrd() result (null on an unmounted component) passed to a setter / set()
# with no null guard (deferred-lints D4). [ev: retro wb-mapping-ord-npe-and-wsl-windows-jdk Δ1]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SNO="$KIT/toolbelt/lint-set-null-ord.sh"
  SRC="$TMPDIR_T/Mod/src/com/x"
  mkdir -p "$SRC"
}
teardown() { rm -rf "$TMPDIR_T"; }

# wrap <body-lines...> — one class, one method holding the given body
wrap() {
  printf 'package com.x;\npublic class BMgr {\n  void addMapping(BComponent point, BImportMap m) {\n'
  printf '    %s\n' "$@"
  printf '  }\n}\n'
}

@test "SNO1: getSlotPathOrd() passed straight to a setter WARNs with the setter name" {
  # Mutation: SNO1 -- dropping the direct-argument rule passes setTargetOrd(point.getSlotPathOrd()) clean
  wrap 'm.setTargetOrd(point.getSlotPathOrd());' > "$SRC/BMgr.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-set-null-ord  "*"BMgr.java:4  getSlotPathOrd() passed straight to setTargetOrd(...)"* ]]
}

@test "SNO1-guard: a direct pass behind a getSlotPathOrd() null check is not a WARN" {
  wrap 'if (point.getSlotPathOrd() != null) m.setTargetOrd(point.getSlotPathOrd());' > "$SRC/BMgr.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SNO2: a local assigned from getSlotPathOrd() reaching set(prop, v) with no guard WARNs" {
  wrap 'BOrd ord = point.getSlotPathOrd();' 'm.set(BImportMap.targetOrd, ord);' > "$SRC/BMgr.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BMgr.java:5  'ord' (getSlotPathOrd() at line 4) reaches set(...) with no null guard"* ]]
}

@test "SNO2-single: the local as the ONLY setter argument, setTargetOrd(ord), WARNs too" {
  wrap 'BOrd ord = point.getSlotPathOrd();' 'm.setTargetOrd(ord);' > "$SRC/BMgr.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BMgr.java:5  'ord' (getSlotPathOrd() at line 4) reaches setTargetOrd(...) with no null guard"* ]]
}

@test "SNO2-guard: a null / isNull() guard or a ternary between the assignment and the set is not a WARN" {
  # Mutation: SNO2-guard -- ignoring the null guard WARNs a local that is checked before the set
  wrap 'BOrd ord = point.getSlotPathOrd();' 'if (ord == null || ord.isNull()) return;' 'm.setTargetOrd(ord);' > "$SRC/BMgr.java"
  { printf 'package com.x;\npublic class BTern {\n  void a(BComponent p, BImportMap m) {\n'
    printf '    BOrd o = p.getSlotPathOrd();\n    m.setTargetOrd(o == null ? BOrd.NULL : o);\n  }\n}\n'; } > "$SRC/BTern.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SNO2-scope: a same-named local in a sibling method is not tracked across methods" {
  # Mutation: SNO2-scope -- matching the variable outside its own method WARNs a same-named local in a sibling method
  { printf 'package com.x;\npublic class BTwo {\n  void a(BComponent p) {\n    BOrd ord = p.getSlotPathOrd();\n    log(ord);\n  }\n'
    printf '  void b(BImportMap m, BOrd ord) {\n    m.setTargetOrd(ord);\n  }\n}\n'; } > "$SRC/BTwo.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SNO3: getSlotPathOrd() that is not the setter argument itself (a call on it) is out of scope" {
  wrap 'String s = point.getSlotPathOrd().toString();' 'm.setName(s);' > "$SRC/BMgr.java"
  run "$SNO" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "SNO4: --strict exits 1 on a WARN; usage errors exit 3" {
  wrap 'm.setTargetOrd(point.getSlotPathOrd());' > "$SRC/BMgr.java"
  run "$SNO" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  run "$SNO" --bogus "$TMPDIR_T/Mod"
  [ "$status" -eq 3 ]
  run "$SNO"
  [ "$status" -eq 3 ]
  run "$SNO" "$TMPDIR_T/nope"
  [ "$status" -eq 3 ]
}

@test "SNO-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: SNO-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
  printf 'x\n' > "$SRC/U.java"
  chmod 000 "$SRC/U.java"
  if [ -r "$SRC/U.java" ]; then chmod 644 "$SRC/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SNO" "$TMPDIR_T/Mod"
  chmod 644 "$SRC/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "SNO-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: SNO-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
  mkdir -p "$SRC/locked"
  printf 'class B {}\n' > "$SRC/locked/B.java"
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$SNO" "$TMPDIR_T/Mod"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/locked"* ]]
}
