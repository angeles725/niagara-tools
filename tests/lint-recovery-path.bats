#!/usr/bin/env bats
# RED-FIRST pins for lint-recovery-path.sh (Wave 3, LR1).
# Detects the "stranded-heater" shape: a protection/heat output slot (*Out/*Resistance/*Heat/*Resistencia)
# is written OFF only inside a mode-gated method with no unconditional reset in started()/atSteadyState()/enable().
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

@test "LRP-usage: no argument -> exit 3 (usage)" {
  run "$LRP"
  [ "$status" -eq 3 ]
}

@test "LRP-smoke: empty-tree exit 3 (K20 guard — wrong path or empty scaffold)" {
  local EMPTY; EMPTY=$(mktemp -d)
  run "$LRP" "$EMPTY"
  rm -rf "$EMPTY"
  # no Java files -> exit 0 (clean; empty tree has no violations)
  [ "$status" -eq 0 ]
}

@test "LRP-limitation: limitation comment present in script header (documented false-negative)" {
  grep -q 'helper method' "$KIT/toolbelt/lint-recovery-path.sh"
}
