#!/usr/bin/env bats
# RED-FIRST pins for lint-recovery-path.sh (Wave 3, LR1).
# Detects the "stranded-heater" shape: a protection/heat output slot (*Out/*Resistance/*Heat/*Resistencia)
# is written OFF only inside a mode-gated method with no unconditional reset in started()/atSteadyState()/changed()/
# enable()/start().
#
# Real commissioning defect (PANCCADIA 2026-09): resistanceOut was turned off inside execute() guarded
# by `if (inDefrost) return;` but started() never reset it unconditionally — when the mode cleared,
# the heater stayed off and froze the room silently.
#
# RED today: lint-recovery-path.sh does not exist -> every pin fails for the right reason (tool absent).
#
# NAMED MUTATION (post-green): remove the guarded-only classification so that mode-gated-only safe-offs
# pass instead of FAIL -> LRP-noguard stops firing, the real stranded-heater shape passes, which is
# exactly the production defect this check guards.

setup() {
  KIT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)/build-n4-module-kit"
  LRP="$KIT/toolbelt/lint-recovery-path.sh"
  FX="$BATS_TEST_DIRNAME/fixtures/lint-recovery-path"
  ONE="$BATS_TEST_TMPDIR/one"; mkdir -p "$ONE"
}
only() { rm -f "$ONE"/*.java; cp "$FX/$1" "$ONE/"; }

@test "LRP-noguard: GuardedOnly.java (resistanceOut set off only in execute, not in started) -> FAIL exit 1" {
  only GuardedOnly.java
  run "$LRP" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL"* ]]
  [[ "$output" == *"ResistanceOut"* ]]
}

@test "LRP-clean: WithUnconditional.java (unconditional reset in started) -> exit 0, no FAIL" {
  only WithUnconditional.java
  run "$LRP" "$ONE"
  [ "$status" -eq 0 ]
  [[ "$output" != *"FAIL"* ]]
}

@test "LRP-comment: a safe-off in a comment or a string is not a write, and a \"{\" in a string keeps the method scope" {
  # Mutation: LRP-comment -- matching the raw line cut at "//" FAILs a commented-out write and misreads started()
  # Go golden: go/internal/lint/rules/recoverypath/testdata/truth-comment-literal-1.txt
  printf '%s\n' 'package com.x;' 'public class C extends BComponent {' '  void execute() {' '    /* setAOut(false); */' \
    '    String b = "{";' '    setDOut(false);' '  }' '  void started() {' '    setDOut(false);' '  }' '}' > "$ONE/C.java"
  run "$LRP" "$ONE"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LRP-scope: a nested class's started() reset counts (method names come from the shared method scanner)" {
  # Mutation: LRP-scope -- naming methods by a 5-line back-scan gives Inner.started() the name of the call above it
  # Go golden: go/internal/lint/rules/recoverypath/testdata/truth-method-scope-1.txt
  printf '%s\n' 'package com.x;' 'public class Nested extends BComponent {' '  void execute() {' '    setHeatOut(false);' '  }' \
    '  static class Inner {' '    void started() {' '      setHeatOut(false);' '    }' '  }' '}' > "$ONE/Nested.java"
  run "$LRP" "$ONE"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LRP-word: resetX(false) is not a write of slot X, and a second write on a line counts" {
  # Mutation: LRP-word -- matching set/get inside an identifier FAILs resetBoilerHeat(false) as slot BoilerHeat
  # Go golden: go/internal/lint/rules/recoverypath/testdata/truth-slot-names-1.txt
  printf '%s\n' 'package com.x;' 'public class S extends BComponent {' '  void execute() {' '    resetBoilerHeat(false);' \
    '    getC().setValue(false); setDOut(false);' '  }' '  void started() {' '    setDOut(false);' '  }' '}' > "$ONE/S.java"
  run "$LRP" "$ONE"
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "LRP-text: the row names every method the rule accepts as a reset" {
  # Mutation: LRP-text -- a row naming fewer reset methods than the rule accepts sends the fix to the wrong method
  only GuardedOnly.java
  run "$LRP" "$ONE"
  [ "$status" -eq 1 ]
  [[ "$output" == *"FAIL  lint-recovery-path  $ONE/GuardedOnly.java:"*"  LRP: ResistanceOut has guarded-only safe-off in method \"execute\" -- no unconditional reset in started/atSteadyState/changed/enable/start" ]]
}

@test "LRP-usage: no argument -> exit 3 (usage)" {
  run "$LRP"
  [ "$status" -eq 3 ]
}

@test "LRP-smoke: empty-tree exit 3 (K20 guard — wrong path or empty scaffold)" {
  local EMPTY; EMPTY=$(mktemp -d)
  run "$LRP" "$EMPTY"
  rm -rf "$EMPTY"
  # no Java files -> ERROR row + exit 3: an empty tree is a wrong path, never a clean pass
  [ "$status" -eq 3 ]
  [[ "$output" == "ERROR  lint-recovery-path  $EMPTY  no files to scan"* ]]
}

@test "LRP-limitation: limitation comment present in script header (documented false-negative)" {
  grep -q 'helper method' "$KIT/toolbelt/lint-recovery-path.sh"
}

@test "LRP-locked: an unreadable sub-directory is an env error (exit 3), never a clean pass" {
  # Mutation: LRP-locked -- a raw find walk skips the locked directory and passes clean.
  printf "class A {}\n" > "$ONE/A.java"
  mkdir -p "$ONE/locked"
  chmod 000 "$ONE/locked"
  if [ -r "$ONE/locked" ]; then chmod 755 "$ONE/locked"; skip "running as root"; fi
  run "$LRP" "$ONE"
  chmod 755 "$ONE/locked"
  [ "$status" -eq 3 ]
  [[ "$output" == *"lint-recovery-path: cannot list every file under $ONE: "* ]]
}
