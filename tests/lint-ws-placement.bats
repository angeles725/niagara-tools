#!/usr/bin/env bats
# lint-ws-placement.bats — a method that adds a freshly created component and never gives it a wire-sheet
# position (wsAnnotation): Workbench staggers every such sibling onto the same diagonal. [ev: corpus B1172]
# [ev: retro ws-placement]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  WSP="$KIT/toolbelt/lint-ws-placement.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-ws-placement"
  SRC="$TMPDIR_T/Mod/src/com/x"
  mkdir -p "$SRC"
}
teardown() { rm -rf "$TMPDIR_T"; }

# wrap <body-lines...> — one class, one method holding the given body
wrap() {
  printf 'package com.x;\npublic class BSvc extends BComponent {\n  void seed() {\n'
  printf '    %s\n' "$@"
  printf '  }\n}\n'
}

@test "WSP-usage: no argument, an unknown flag or a non-directory exits 3" {
  run "$WSP"
  [ "$status" -eq 3 ]
  run "$WSP" --bogus "$TMPDIR_T/Mod"
  [ "$status" -eq 3 ]
  run "$WSP" "$TMPDIR_T/does-not-exist"
  [ "$status" -eq 3 ]
}

@test "WSP1: parent.add(name, create(type)) with no wsAnnotation WARNs at the add line (DashboardFord shape)" {
  # Mutation: WSP1 -- dropping the creation rule passes parent.add(name, create(type)) clean
  run "$WSP" "$FX/positive"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN  lint-ws-placement  $FX/positive/com/x/BSyncService.java:12  add(op.name, create(...)) in doSyncInventory() never sets a wsAnnotation"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
}

@test "WSP1-strict: --strict promotes the WARN to exit 1" {
  run "$WSP" --strict "$FX/positive"
  [ "$status" -eq 1 ]
  [[ "$output" == *"BSyncService.java:12"* ]]
}

@test "WSP-guard: a method that places its children on a grid is clean (chihuahua shape)" {
  # Mutation: WSP-guard -- dropping the wsAnnotation guard WARNs a method that already places its children
  run "$WSP" --strict "$FX/annotated"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WSP1-local: a component created into a local and added later WARNs, naming the local" {
  # Mutation: WSP1-local -- dropping the local tracking passes `BFoo f = new BFoo(); add("f", f);` clean
  wrap 'BFolder f = new BFolder();' 'add("f", f);' > "$SRC/BSvc.java"
  run "$WSP" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BSvc.java:5  add(\"f\", f = new BFolder()) in seed() never sets a wsAnnotation"* ]]
}

@test "WSP-scope: a sibling method's wsAnnotation does not silence an unplaced add" {
  # Mutation: WSP-scope -- a file-wide guard lets a sibling method's wsAnnotation silence an unplaced add
  { printf 'package com.x;\npublic class BSvc extends BComponent {\n'
    printf '  void place(BComponent c) {\n    c.add("wsAnnotation", BWsAnnotation.make(1, 1, 8), Flags.READONLY);\n  }\n'
    printf '  void seed() {\n    add("n", new BFolder());\n  }\n}\n'; } > "$SRC/BSvc.java"
  run "$WSP" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BSvc.java:7  add(\"n\", new BFolder()) in seed()"* ]]
  [ "$(printf '%s\n' "$output" | grep -c '^WARN')" -eq 1 ]
}

@test "WSP-list: collection adds, value-typed properties and addAll are not flagged" {
  # Mutation: WSP-list -- reading the last argument of a one-argument add WARNs list.add(new BFoo())
  run "$WSP" --strict "$FX/list"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WSP-value: a BStatus* / BAbsTime value added as a property is not a glyph and is not flagged" {
  # Mutation: WSP-value -- dropping the value-type exclusion WARNs add("st", new BStatusNumeric())
  wrap 'add("st", new BStatusNumeric());' 'add("tm", new BAbsTime());' > "$SRC/BSvc.java"
  run "$WSP" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WSP-cast-wrap: a cast creation and arguments wrapped over lines still WARN at the add line" {
  wrap 'parent.add(name,' '    (BComponent) template.newCopy());' > "$SRC/BSvc.java"
  run "$WSP" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BSvc.java:4  add(name, template.newCopy(...)) in seed()"* ]]
}

@test "WSP-comment: a wsAnnotation mentioned only in a comment does not guard the method" {
  wrap '// TODO set a wsAnnotation' 'add("n", new BFolder());' > "$SRC/BSvc.java"
  run "$WSP" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BSvc.java:5  add(\"n\", new BFolder())"* ]]
}

@test "WSP-literal: an add( or an assignment quoted inside a string literal is not code" {
  # Mutation: WSP-literal -- matching the unmasked Strip view WARNs log("add(x, new BFolder())")
  # Go golden: go/internal/lint/rules/wsplacement/testdata/truth-literals-1.txt
  wrap 'log("add(x, new BFolder())");' 'log("f = new BFolder()");' 'add("f", f);' > "$SRC/BSvc.java"
  run "$WSP" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WSP-widget: a bajaui widget added to a pane is not a wire-sheet node" {
  # Mutation: WSP-widget -- dropping the javax.baja.ui import check WARNs buttons.add(null, new BButton(cmd))
  # Go golden: go/internal/lint/rules/wsplacement/testdata/truth-widgets-1.txt
  { printf 'package com.x;\nimport javax.baja.ui.BButton;\nimport javax.baja.ui.pane.BGridPane;\n'
    printf 'public class BMgr extends BWbView {\n  void build() {\n'
    printf '    BGridPane buttons = new BGridPane(4);\n    buttons.add(null, new BButton(cmd));\n    add(null, buttons);\n'
    printf '  }\n}\n'; } > "$SRC/BMgr.java"
  run "$WSP" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "WSP-roots: every src root given is scanned" {
  run "$WSP" "$FX/annotated" "$FX/positive"
  [ "$status" -eq 0 ]
  [[ "$output" == *"BSyncService.java:12"* ]]
}

@test "WSP-awkfail: an unreadable source file is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: WSP-awkfail -- ignoring the awk exit status reports an unreadable source file as clean
  printf 'x\n' > "$SRC/U.java"
  chmod 000 "$SRC/U.java"
  if [ -r "$SRC/U.java" ]; then chmod 644 "$SRC/U.java"; skip "running as root: chmod 000 does not block reads"; fi
  run "$WSP" "$TMPDIR_T/Mod"
  chmod 644 "$SRC/U.java"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/U.java"* ]]
}

@test "WSP-finderr: an unreadable sub-directory is an env error (exit 3, named on stderr), never a clean pass" {
  # Mutation: WSP-finderr -- ignoring the find exit status skips an unreadable sub-directory and reports clean
  mkdir -p "$SRC/locked"
  printf 'class B {}\n' > "$SRC/locked/B.java"
  chmod 000 "$SRC/locked"
  if [ -r "$SRC/locked" ]; then chmod 755 "$SRC/locked"; skip "running as root: chmod 000 does not block reads"; fi
  run "$WSP" "$TMPDIR_T/Mod"
  chmod 755 "$SRC/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"Mod/src/com/x/locked"* ]]
}
