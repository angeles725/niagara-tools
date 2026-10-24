#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-wb-file-chooser.sh
# A BWbFieldEditor.dialog(...BOrd.NULL) in a -wb class opens the file-system chooser (C:\),
# useless in a station context. The correct pattern is BComponentChooser or the targetType facet.
# [ev: retro apillm-headless-servlet-rt-4.14-deltas Δ8] [ev: types/wb-widgets.md]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  WFC="$KIT/toolbelt/lint-wb-file-chooser.sh"
  mkdir -p "$TMPDIR_T/Mod/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "WFC-usage: no arg exits 3" {
  run "$WFC"
  [ "$status" -eq 3 ]
}

@test "WFC-nondir: a non-directory arg exits 3" {
  run "$WFC" "$TMPDIR_T/does-not-exist"
  [ "$status" -eq 3 ]
}

@test "WFC1: dialog(...BOrd.NULL) with BComponentChooser present in the same file is clean (exit 0, no WARN)" {
  printf 'class A {\n  void s(){ BWbFieldEditor.dialog(this, BOrd.NULL); BComponentChooser c; }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$WFC" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
  # Named mutation WFC2: removing the BComponentChooser-absent guard makes this file false-WARN.
}

@test "WFC2: dialog(...BOrd.NULL) with NO BComponentChooser/targetType is WARNed (exit 0 advisory)" {
  printf 'class A {\n  void s(){ BWbFieldEditor.dialog(this, BOrd.NULL); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$WFC" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" == *"WARN"* ]]
  [[ "$output" == *"lint-wb-file-chooser"* ]]
  [[ "$output" == *"A.java:2"* ]]
}

@test "WFC2-strict: --strict promotes the WARN to exit 1" {
  printf 'class A {\n  void s(){ BWbFieldEditor.dialog(this, BOrd.NULL); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$WFC" --strict "$TMPDIR_T/Mod"
  [ "$status" -eq 1 ]
  [[ "$output" == *"WARN"* ]]
}

@test "WFC3: a file using the targetType facet is clean (exit 0, no WARN)" {
  printf 'class A {\n  void s(){ BWbFieldEditor.dialog(this, BOrd.NULL); setFacets(BFacets.make("targetType=baja:Component")); }\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$WFC" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "WFC4: commented-out call does not trigger (comment strip)" {
  printf 'class A {\n  // BWbFieldEditor.dialog(this, BOrd.NULL);\n  void f(){}\n}\n' \
    > "$TMPDIR_T/Mod/src/com/x/A.java"
  run "$WFC" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [[ "$output" != *"WARN"* ]]
}

@test "WFC-locked: an unreadable sub-directory is an env error (exit 3), never a clean pass" {
  # Mutation: WFC-locked -- a raw find walk skips the locked directory and passes clean.
  mkdir -p "$TMPDIR_T/Mod/locked"
  chmod 000 "$TMPDIR_T/Mod/locked"
  if [ -r "$TMPDIR_T/Mod/locked" ]; then chmod 755 "$TMPDIR_T/Mod/locked"; skip "running as root"; fi
  run "$WFC" "$TMPDIR_T/Mod"
  chmod 755 "$TMPDIR_T/Mod/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"lint-wb-file-chooser: cannot list every file under $TMPDIR_T/Mod: "* ]]
}

@test "WFC5: a commented dialog is clean; a commented or quoted BComponentChooser never exempts; a dialog after a // inside a string is WARNed" {
  # Mutation: WFC5 -- matching the raw line (no comment/literal blanking) exempts A.java and F.java.
  # [ev: truth audit 2026-10-22 T3]
  printf 'class A {\n  /* BComponentChooser c; */\n  void f() { BWbFieldEditor.dialog(p, t, BOrd.NULL); }\n}\n' > "$TMPDIR_T/Mod/src/com/x/A.java"
  printf 'class C {\n  String u = "http://h"; void f() { BWbFieldEditor.dialog(p, t, BOrd.NULL); }\n}\n' > "$TMPDIR_T/Mod/src/com/x/C.java"
  printf 'class E {\n  /* BWbFieldEditor.dialog(p, t, BOrd.NULL); */\n}\n' > "$TMPDIR_T/Mod/src/com/x/E.java"
  printf 'class F {\n  void f() { BWbFieldEditor.dialog(p, t, BOrd.NULL); }\n  String s = "BComponentChooser";\n}\n' > "$TMPDIR_T/Mod/src/com/x/F.java"
  run "$WFC" "$TMPDIR_T/Mod"
  [ "$status" -eq 0 ]
  [ "$output" = "WARN  lint-wb-file-chooser  $TMPDIR_T/Mod/src/com/x/A.java:3  void f() { BWbFieldEditor.dialog(p, t, BOrd.NULL); }
WARN  lint-wb-file-chooser  $TMPDIR_T/Mod/src/com/x/C.java:2  String u = \"http://h\"; void f() { BWbFieldEditor.dialog(p, t, BOrd.NULL); }
WARN  lint-wb-file-chooser  $TMPDIR_T/Mod/src/com/x/F.java:2  void f() { BWbFieldEditor.dialog(p, t, BOrd.NULL); }" ]
}
