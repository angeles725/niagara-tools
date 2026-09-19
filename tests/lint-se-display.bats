#!/usr/bin/env bats
# Tests for build-n4-module-kit/toolbelt/lint-se-display.sh
# A -se part loads on the headless JACE daemon; display classes fail at runtime.
# [ev: retro 2026-09-19-se-profile]

setup() {
  TMPDIR_T="$(mktemp -d)"; export TMPDIR_T
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  SED_L="$KIT/toolbelt/lint-se-display.sh"
  mkdir -p "$TMPDIR_T/Mod-se/src/com/x"
}
teardown() { rm -rf "$TMPDIR_T"; }

@test "SED-usage: no arg exits 3" { run "$SED_L"; [ "$status" -eq 3 ]; }
@test "SED-nondir: bad dir exits 3" { run "$SED_L" "$TMPDIR_T/nope"; [ "$status" -eq 3 ]; }

@test "SED1: headless-safe AWT (java.awt.print) is clean (exit 0)" {
  printf 'import java.awt.print.PrinterJob;\nclass P { }\n' > "$TMPDIR_T/Mod-se/src/com/x/P.java"
  run "$SED_L" "$TMPDIR_T/Mod-se"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "SED2: a JFrame import is flagged (exit 1)" {
  printf 'import javax.swing.JFrame;\nclass S { }\n' > "$TMPDIR_T/Mod-se/src/com/x/S.java"
  run "$SED_L" "$TMPDIR_T/Mod-se"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"lint-se-display"* ]]
  # Named mutation: drop the JFrame case -> SED2 FAIL vanishes.
}

@test "SED3: new JDialog(...) is flagged (exit 1)" {
  printf 'class S { void f(){ Object d = new JDialog(); } }\n' > "$TMPDIR_T/Mod-se/src/com/x/S.java"
  run "$SED_L" "$TMPDIR_T/Mod-se"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
}

@test "SED4: a COMMENTED JFrame import is not flagged (exit 0)" {
  printf 'class S { } // import javax.swing.JFrame;\n' > "$TMPDIR_T/Mod-se/src/com/x/S.java"
  run "$SED_L" "$TMPDIR_T/Mod-se"
  [ "$status" -eq 0 ]
}
